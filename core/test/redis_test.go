package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestRedisConnection(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	defer rdb.Close()

	// 测试连接
	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		t.Fatalf("无法连接到 Redis: %v", err)
	}
	err = rdb.Set(ctx, "key", "value", 10).Err()
	if err != nil {
		panic(err)
	}
	val, err := rdb.Get(ctx, "key").Result()
	if err == redis.Nil {
		fmt.Println("key does not exist")
	} else if err != nil {
		panic(err)
	} else {
		fmt.Println("key", val)
	}

}
