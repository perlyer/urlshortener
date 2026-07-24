// Command redirector - горячий путь чтения: по короткому коду отдаёт 302 на
// оригинальный URL и асинхронно шлёт событие клика в Kafka (не блокируя ответ).
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/perlyer/urlshortener/internal/cache"
	"github.com/perlyer/urlshortener/internal/config"
	"github.com/perlyer/urlshortener/internal/events"
	"github.com/perlyer/urlshortener/internal/metrics"
	"github.com/perlyer/urlshortener/internal/storage"
	"github.com/perlyer/urlshortener/internal/storage/db"
)

// publisher - то, что нужно redirector'у от продьюсера. Интерфейс у
// потребителя, чтобы в тестах подставить заглушку без Kafka.
type publisher interface {
	Publish(ctx context.Context, e events.ClickEvent) error
}

func makeRedirectHandler(store *storage.Store, pub publisher) http.HandlerFunc {
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

		// Собираем событие клика из запроса и шлём его асинхронно -
		// редирект не должен ждать запись аналитики.
		event := events.ClickEvent{
			Code:      code,
			Timestamp: time.Now().UTC(),
			UserAgent: r.UserAgent(),
			Referer:   r.Referer(),
			Language:  primaryLanguage(r.Header.Get("Accept-Language")),
			IP:        clientIP(r),
		}
		go func() {
			// ВАЖНО: свой контекст, а НЕ r.Context() - тот отменится, как
			// только handler вернёт ответ, и публикация оборвётся.
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := pub.Publish(ctx, event); err != nil {
				slog.Warn("публикация клика", "err", err)
			}
		}()

		http.Redirect(w, r, url, http.StatusFound)
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

// primaryLanguage берёт основной язык из Accept-Language:
// "ru-RU,ru;q=0.9,en;q=0.8" → "ru".
func primaryLanguage(header string) string {
	if header == "" {
		return ""
	}
	lang := header
	if i := strings.IndexByte(lang, ','); i >= 0 {
		lang = lang[:i] // первый языковой тег
	}
	if i := strings.IndexByte(lang, ';'); i >= 0 {
		lang = lang[:i] // отбрасываем ;q=...
	}
	if i := strings.IndexByte(lang, '-'); i >= 0 {
		lang = lang[:i] // "ru-RU" → "ru"
	}
	return strings.TrimSpace(lang)
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("конфиг", "err", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("подключение к БД", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	redisCache := cache.NewRedis(cfg.RedisAddr)
	store := storage.New(db.New(pool), redisCache, cfg.CodeLength)

	producer := events.NewProducer(strings.Split(cfg.KafkaBrokers, ","), events.Topic)
	defer producer.Close()

	r := chi.NewRouter()
	r.Use(metrics.Middleware("redirector"))
	r.Handle("/metrics", metrics.Handler())
	r.Get("/{code}", makeRedirectHandler(store, producer))

	slog.Info("redirector запущен", "port", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		slog.Error("сервер остановлен", "err", err)
		os.Exit(1)
	}
}
