package main

import (
	"context"
	"errors"
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

func makeRedirectHandler(store *storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := chi.URLParam(r, "code")

		url, err := store.Resolve(r.Context(), code)
		if errors.Is(err, storage.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			slog.Error("получение ссылки", "err", err)
			http.Error(w, "внутренняя ошибка", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, url, http.StatusFound)
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
	r.Get("/{code}", makeRedirectHandler(store))

	slog.Info("redirector запущен", "port", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		slog.Error("сервер остановлен", "err", err)
		os.Exit(1)
	}
}
