// Command api - HTTP-сервис записи: принимает запрос на создание короткой
// ссылки и возвращает её. Точка входа: считать конфиг, поднять зависимости
// (пул к БД, роутер) и запустить сервер. Бизнес-логика живёт в internal/.
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/perlyer/urlshortener/internal/cache"
	"github.com/perlyer/urlshortener/internal/config"
	"github.com/perlyer/urlshortener/internal/metrics"
	"github.com/perlyer/urlshortener/internal/ratelimit"
	"github.com/perlyer/urlshortener/internal/storage"
	"github.com/perlyer/urlshortener/internal/storage/db"
)

// createRequest - тело входящего POST /api/links.
type createRequest struct {
	URL string `json:"url"`
}

// createResponse - что отдаём клиенту.
type createResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
}

// makeCreateHandler возвращает HTTP-обработчик, замкнутый на Store и baseURL.
// Это HTTP-слой: разобрать JSON → позвать Store → упаковать ответ в JSON.
// Про SQL он ничего не знает.
func makeCreateHandler(store *storage.Store, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "невалидный JSON", http.StatusBadRequest)
			return
		}
		if req.URL == "" {
			http.Error(w, "поле url обязательно", http.StatusBadRequest)
			return
		}

		link, err := store.CreateShortLink(r.Context(), req.URL)
		if err != nil {
			slog.Error("создание ссылки", "err", err)
			http.Error(w, "внутренняя ошибка", http.StatusInternalServerError)
			return
		}

		resp := createResponse{
			Code:     link.Code,
			ShortURL: baseURL + "/" + link.Code,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

type statItem struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

type statsResponse struct {
	Code           string                `json:"code"`
	UniqueVisitors int64                 `json:"unique_visitors"`
	Stats          map[string][]statItem `json:"stats"`
}

func makeStatsHandler(store *storage.Store, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := chi.URLParam(r, "code")
		rows, err := store.ClickStats(r.Context(), code)
		if err != nil {
			slog.Error("получение статистики", "err", err)
			http.Error(w, "внутренняя ошибка", http.StatusInternalServerError)
			return
		}
		stats := map[string][]statItem{}
		for _, row := range rows {
			stats[row.Dimension] = append(stats[row.Dimension], statItem{row.Value, row.Count})
		}
		unique, _ := rdb.PFCount(r.Context(), "unique:"+code).Result()

		resp := statsResponse{Code: code, UniqueVisitors: unique, Stats: stats}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// clientIP достаёт IP клиента: за прокси - из X-Forwarded-For (первый в списке),
// иначе - из RemoteAddr (там "host:port").
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func rateLimitMiddleware(limiter *ratelimit.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			ok, err := limiter.Allow(r.Context(), ip)
			if err != nil {
				slog.Error("rate limited", "err", err)
				next.ServeHTTP(w, r)
				return
			}
			if !ok {
				http.Error(w, "слишком много запросов", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("конфиг", "err", err)
		os.Exit(1)
	}

	// Пул соединений к Postgres - один на весь сервис.
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("подключение к БД", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Кэш ссылок в Redis.
	redisCache := cache.NewRedis(cfg.RedisAddr)
	statsRedis := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer statsRedis.Close()

	// Собираем слои снизу вверх: pool → Queries → Store (+ кэш).
	store := storage.New(db.New(pool), redisCache, cfg.CodeLength)

	// Роутер: какой путь/метод → какой обработчик.
	r := chi.NewRouter()

	rateLimiter := ratelimit.NewLimiter(statsRedis, 20, 10)

	r.Use(metrics.Middleware("api"))
	// CORS для фронта (dev). В проде список Origin стоит сузить.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
	}))
	r.Handle("/metrics", metrics.Handler())
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			http.Error(w, "db unavailable", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})
	r.With(rateLimitMiddleware(rateLimiter)).Post("/api/links", makeCreateHandler(store, cfg.BaseURL))
	r.Get("/api/links/{code}/stats", makeStatsHandler(store, statsRedis))

	slog.Info("api запущен", "port", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		slog.Error("сервер остановлен", "err", err)
		os.Exit(1)
	}
}
