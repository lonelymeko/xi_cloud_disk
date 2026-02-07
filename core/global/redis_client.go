package global

import (
	"github.com/redis/go-redis/v9"
)

// InitRedis 初始化 Redis 客户端。
func InitRedis(addr, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}
