package ratelimit_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"

	"github.com/perlyer/urlshortener/internal/ratelimit"
)

func newLimiter(t *testing.T, capacity int, refill float64) *ratelimit.Limiter {
	t.Helper()
	ctx := context.Background()
	rc, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Fatalf("redis: %v", err)
	}
	t.Cleanup(func() { _ = rc.Terminate(context.Background()) })

	connStr, err := rc.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("conn: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: strings.TrimPrefix(connStr, "redis://")})
	t.Cleanup(func() { _ = rdb.Close() })

	return ratelimit.NewLimiter(rdb, capacity, refill)
}

func TestBurstThenLimit(t *testing.T) {
	ctx := context.Background()
	lim := newLimiter(t, 5, 1) // ёмкость 5, пополнение 1/сек

	allowed := 0
	for i := 0; i < 5; i++ {
		ok, err := lim.Allow(ctx, "1.2.3.4")
		if err != nil {
			t.Fatalf("Allow: %v", err)
		}
		if ok {
			allowed++
		}
	}
	if allowed != 5 {
		t.Fatalf("burst: пропущено %d из 5, ожидали 5", allowed)
	}

	// корзина пуста - следующий запрос отклоняется
	ok, _ := lim.Allow(ctx, "1.2.3.4")
	if ok {
		t.Fatal("6-й запрос должен быть отклонён (корзина пуста)")
	}
}

func TestPerKeyIsolation(t *testing.T) {
	ctx := context.Background()
	lim := newLimiter(t, 1, 1) // ёмкость 1

	if ok, _ := lim.Allow(ctx, "ip-a"); !ok {
		t.Fatal("первый запрос ip-a должен пройти")
	}
	if ok, _ := lim.Allow(ctx, "ip-a"); ok {
		t.Fatal("второй запрос ip-a должен быть отклонён")
	}
	// другой ключ - своя корзина, не затронута
	if ok, _ := lim.Allow(ctx, "ip-b"); !ok {
		t.Fatal("ip-b должен иметь собственную корзину")
	}
}

func TestRefill(t *testing.T) {
	ctx := context.Background()
	lim := newLimiter(t, 1, 20) // пополнение 20/сек → токен возвращается за ~50мс

	if ok, _ := lim.Allow(ctx, "ip-r"); !ok {
		t.Fatal("первый запрос должен пройти")
	}
	if ok, _ := lim.Allow(ctx, "ip-r"); ok {
		t.Fatal("сразу второй - отклонён")
	}

	time.Sleep(200 * time.Millisecond) // ждём пополнения

	if ok, _ := lim.Allow(ctx, "ip-r"); !ok {
		t.Fatal("после паузы корзина должна пополниться")
	}
}
