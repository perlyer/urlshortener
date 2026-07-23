// Command analytics - Kafka consumer: читает события кликов из топика,
// обогащает (UA, гео, уникальные) и агрегирует в click_stats.
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oschwald/geoip2-golang"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"

	"github.com/perlyer/urlshortener/internal/analytics"
	"github.com/perlyer/urlshortener/internal/config"
	"github.com/perlyer/urlshortener/internal/events"
	"github.com/perlyer/urlshortener/internal/storage/db"
)

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

	geo, err := geoip2.Open(cfg.GeoIPPath)
	if err != nil {
		slog.Error("открытие GeoIP-базы", "err", err, "path", cfg.GeoIPPath)
		os.Exit(1)
	}
	defer geo.Close()

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer rdb.Close()

	processor := analytics.NewProcessor(db.New(pool), rdb, geo)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: strings.Split(cfg.KafkaBrokers, ","),
		Topic:   events.Topic,
		GroupID: "analytics", // consumer group - offset хранится в Kafka
	})
	defer reader.Close()

	// Грациозная остановка: по Ctrl+C / SIGTERM отменяем контекст,
	// ReadMessage разблокируется и цикл выходит.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("analytics запущен, читаю топик", "topic", events.Topic)
	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("analytics остановлен")
				return
			}
			slog.Error("чтение из Kafka", "err", err)
			continue
		}

		var e events.ClickEvent
		if err := json.Unmarshal(msg.Value, &e); err != nil {
			slog.Warn("битое событие, пропускаю", "err", err)
			continue
		}
		if err := processor.Process(ctx, e); err != nil {
			slog.Error("обработка клика", "err", err, "code", e.Code)
		}
	}
}
