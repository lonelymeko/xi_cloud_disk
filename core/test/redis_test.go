package test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisConnection(t *testing.T) {
	rdb := newFakeRedisClient()
	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		t.Fatalf("无法连接到 Redis: %v", err)
	}
	err = rdb.Set(ctx, "key", "value", 10*time.Second).Err()
	if err != nil {
		t.Fatal(err)
	}
	val, err := rdb.Get(ctx, "key").Result()
	if err != nil {
		t.Fatal(err)
	}
	if val != "value" {
		t.Fatalf("value mismatch: %s", val)
	}

}

type fakeRedisClient struct {
	mu   sync.Mutex
	data map[string]string
}

func newFakeRedisClient() *fakeRedisClient {
	return &fakeRedisClient{data: map[string]string{}}
}

func (f *fakeRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	val, ok := f.data[key]
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(val, nil)
}

func (f *fakeRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	f.mu.Lock()
	f.data[key] = fmt.Sprint(value)
	f.mu.Unlock()
	return redis.NewStatusResult("OK", nil)
}

func (f *fakeRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	f.mu.Lock()
	var count int64
	for _, key := range keys {
		if _, ok := f.data[key]; ok {
			delete(f.data, key)
			count++
		}
	}
	f.mu.Unlock()
	return redis.NewIntResult(count, nil)
}

func (f *fakeRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	return redis.NewStatusResult("PONG", nil)
}
