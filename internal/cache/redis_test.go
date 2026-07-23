package cache

import (
	"context"
	"strings"
	"testing"
	"time"

	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

// newTestCache поднимает Redis в контейнере и возвращает RedisCache к нему.
func newTestCache(t *testing.T, ctx context.Context) *RedisCache {
	t.Helper()

	rc, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Fatalf("не удалось поднять redis: %v", err)
	}
	t.Cleanup(func() { _ = rc.Terminate(context.Background()) })

	connStr, err := rc.ConnectionString(ctx) // вида redis://host:port
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	addr := strings.TrimPrefix(connStr, "redis://")
	return NewRedis(addr)
}

func TestRedisCacheSetGet(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	c := newTestCache(t, ctx)

	if err := c.Set(ctx, "url:abc", "https://example.com"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, found, err := c.Get(ctx, "url:abc")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("ожидали попадание, получили промах")
	}
	if val != "https://example.com" {
		t.Fatalf("val = %q, ожидали оригинальный URL", val)
	}
}

func TestRedisCacheMiss(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	c := newTestCache(t, ctx)

	_, found, err := c.Get(ctx, "url:nosuch")
	if err != nil {
		t.Fatalf("Get вернул ошибку на промахе: %v", err)
	}
	if found {
		t.Fatal("ожидали промах на несуществующем ключе")
	}
}
