package analytics

import (
	"context"
	"fmt"

	"github.com/oschwald/geoip2-golang"
	"github.com/redis/go-redis/v9"

	"github.com/perlyer/urlshortener/internal/events"
	"github.com/perlyer/urlshortener/internal/storage/db"
)

// Processor агрегирует события кликов: измерения - в Postgres (click_stats),
// уникальных посетителей - в HyperLogLog (Redis).
type Processor struct {
	q   *db.Queries
	rdb *redis.Client
	geo *geoip2.Reader
}

func NewProcessor(q *db.Queries, rdb *redis.Client, geo *geoip2.Reader) *Processor {
	return &Processor{q: q, rdb: rdb, geo: geo}
}

// Process обрабатывает одно событие клика: инкрементит счётчик по каждому
// измерению и добавляет посетителя в HLL-счётчик уникальных.
func (p *Processor) Process(ctx context.Context, e events.ClickEvent) error {
	for _, d := range extractDimensions(e, p.geo) {
		err := p.q.IncrClickStat(ctx, db.IncrClickStatParams{
			Code:      e.Code,
			Dimension: d.Name,
			Value:     d.Value,
		})
		if err != nil {
			return fmt.Errorf("analytics: инкремент %s=%s: %w", d.Name, d.Value, err)
		}
	}

	// Уникальные - HyperLogLog: PFADD добавляет отпечаток посетителя,
	// PFCOUNT (в api) вернёт приближённое число уникальных за ~12 КБ памяти.
	if err := p.rdb.PFAdd(ctx, "unique:"+e.Code, visitorHash(e.IP, e.UserAgent)).Err(); err != nil {
		return fmt.Errorf("analytics: pfadd: %w", err)
	}
	return nil
}
