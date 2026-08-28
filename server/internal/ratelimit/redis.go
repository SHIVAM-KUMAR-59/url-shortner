package ratelimit

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisLimiter struct {
	client *redis.Client
}

func NewRedisLimiter(client *redis.Client) *RedisLimiter {
	return &RedisLimiter{
		client: client,
	}
}

func (r *RedisLimiter) Allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}

	if count == 1 {
		if err := r.client.Expire(ctx, key, window).Err(); err != nil {
			log.Printf("failed to set expiry for rate limit key %s: %v", key, err)
		}
	}

	if count > limit {
		return false, nil
	}

	return true, nil
}
