package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedis(addr string) *RedisCache {
	return &RedisCache{client: redis.NewClient(&redis.Options{Addr: addr}), ttl: 24 * time.Hour}
}

func (c *RedisCache) Get(ctx context.Context, key string) (string, bool, error) {
	val, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return val, true, nil

}

func (c *RedisCache) Set(ctx context.Context, key string, value string) error {
	return c.client.Set(ctx, key, value, c.ttl).Err()
}
