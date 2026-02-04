// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"cloud_disk/core/global"
	"cloud_disk/core/internal/config"
	"cloud_disk/core/internal/middleware"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/rest"

	"xorm.io/xorm"
)

type ServiceContext struct {
	Config             config.Config
	DBEngine           *xorm.Engine
	RedisClient        *redis.Client
	FileAuthMiddleware rest.Middleware
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:             c,
		DBEngine:           global.Init(c.MySQL.DataSource),
		RedisClient:        global.InitRedis(c.Redis.Addr, c.Redis.Password, c.Redis.DB),
		FileAuthMiddleware: middleware.NewFileAuthMiddleware(c.Auth.AccessSecret, c.Auth.AccessExpire).Handle,
	}
}
