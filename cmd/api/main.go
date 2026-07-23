// Command api - HTTP-сервис записи: принимает запрос на создание короткой
// ссылки и возвращает её. Точка входа: считать конфиг, поднять зависимости
// (пул к БД, роутер) и запустить сервер. Бизнес-логика живёт в internal/.
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/perlyer/urlshortener/internal/cache"
	"github.com/perlyer/urlshortener/internal/config"
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
	r.Post("/api/links", makeCreateHandler(store, cfg.BaseURL))
	r.Get("/api/links/{code}/stats", makeStatsHandler(store, statsRedis))

	slog.Info("api запущен", "port", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		slog.Error("сервер остановлен", "err", err)
		os.Exit(1)
	}
}
