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

	// Собираем слои снизу вверх: pool → Queries → Store (+ кэш).
	store := storage.New(db.New(pool), redisCache, cfg.CodeLength)

	// Роутер: какой путь/метод → какой обработчик.
	r := chi.NewRouter()
	r.Post("/api/links", makeCreateHandler(store, cfg.BaseURL))

	slog.Info("api запущен", "port", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		slog.Error("сервер остановлен", "err", err)
		os.Exit(1)
	}
}
