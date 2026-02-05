package utils

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisLockClient interface {
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

func AcquireLock(ctx context.Context, rdb RedisLockClient, key string, ttl time.Duration) (bool, error) {
	cmd := rdb.SetNX(ctx, key, "1", ttl)
	return cmd.Result()
}

func ReleaseLock(ctx context.Context, rdb RedisLockClient, key string) {
	_, _ = rdb.Del(ctx, key).Result()
}
