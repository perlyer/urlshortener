// Package ratelimit - распределённый token bucket поверх Redis.
package ratelimit

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// tokenBucketScript атомарно пополняет корзину и пытается забрать токен.
// Вся логика в одном Lua-скрипте - значит операция атомарна на стороне Redis,
// и лимит корректен даже когда к нему одновременно ходят несколько инстансов api.
//
// KEYS[1]           - ключ корзины (например "rl:1.2.3.4")
// ARGV[1] capacity  - ёмкость корзины (burst)
// ARGV[2] refill    - сколько токенов добавляется в секунду
// ARGV[3] now       - текущее время, секунды (float)
// ARGV[4] requested - сколько токенов забрать (обычно 1)
// Возвращает 1 - пропустить, 0 - лимит превышен.
var tokenBucketScript = redis.NewScript(`
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

local bucket = redis.call("HMGET", key, "tokens", "ts")
local tokens = tonumber(bucket[1])
local ts = tonumber(bucket[2])
if tokens == nil then
  tokens = capacity
  ts = now
end

-- пополняем пропорционально прошедшему времени, но не выше ёмкости
local elapsed = math.max(0, now - ts)
tokens = math.min(capacity, tokens + elapsed * refill)

local allowed = 0
if tokens >= requested then
  tokens = tokens - requested
  allowed = 1
end

redis.call("HSET", key, "tokens", tokens, "ts", now)
-- корзина сама протухнет, если по ключу перестанут ходить
redis.call("PEXPIRE", key, math.ceil(capacity / refill * 1000) + 1000)
return allowed
`)

// Limiter - token bucket rate limiter на Redis.
type Limiter struct {
	rdb      *redis.Client
	capacity int
	refill   float64 // токенов в секунду
}

// NewLimiter: capacity - burst (сколько запросов можно разом), refillPerSec -
// установившийся темп (запросов в секунду).
func NewLimiter(rdb *redis.Client, capacity int, refillPerSec float64) *Limiter {
	return &Limiter{rdb: rdb, capacity: capacity, refill: refillPerSec}
}

// Allow пытается забрать один токен для ключа (например, IP клиента).
// true - запрос пропускаем, false - лимит превышен.
func (l *Limiter) Allow(ctx context.Context, key string) (bool, error) {
	now := float64(time.Now().UnixNano()) / float64(time.Second)
	res, err := tokenBucketScript.Run(
		ctx, l.rdb, []string{"rl:" + key},
		l.capacity, l.refill, now, 1,
	).Int()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}
