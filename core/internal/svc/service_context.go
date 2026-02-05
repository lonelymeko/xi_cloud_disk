// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"cloud_disk/core/global"
	"cloud_disk/core/internal/config"
	"cloud_disk/core/internal/middleware"
	"cloud_disk/core/utils"
	"context"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/rest"
	"xorm.io/xorm"
)

type ServiceContext struct {
	Config             config.Config
	DBEngine           *xorm.Engine
	RedisClient        RedisClient
	RabbitMQConn       *amqp091.Connection
	RabbitMQChannel    *amqp091.Channel
	FileAuthMiddleware rest.Middleware
}

type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Ping(ctx context.Context) *redis.StatusCmd
}

type serviceDeps struct {
	initDB             func(string) *xorm.Engine
	initRedis          func(string, string, int) RedisClient
	newFileAuth        func(string, int64) rest.Middleware
	ensureSchema       func(*xorm.Engine) error
	ensureDefaultAdmin func(*xorm.Engine) error
	initRabbitMQ       func(string, int, string, string, string) (*amqp091.Connection, *amqp091.Channel)
}

var deps = serviceDeps{
	initDB:       global.Init,
	initRedis:    func(addr, password string, db int) RedisClient { return global.InitRedis(addr, password, db) },
	initRabbitMQ: global.InitRabbitMQ,
	newFileAuth: func(secret string, expire int64) rest.Middleware {
		return middleware.NewFileAuthMiddleware(secret, expire).Handle
	},
	ensureSchema:       utils.EnsureSchema,
	ensureDefaultAdmin: utils.EnsureDefaultAdmin,
}

func NewServiceContext(c config.Config) *ServiceContext {
	eng := deps.initDB(c.MySQL.DataSource)
	_ = deps.ensureSchema(eng)
	_ = deps.ensureDefaultAdmin(eng)
	rmqConn, rmqCh := deps.initRabbitMQ(c.RabbitMQ.Host, c.RabbitMQ.Port, c.RabbitMQ.Username, c.RabbitMQ.Password, c.RabbitMQ.Vhost)
	// //启动消费者
	// consumer := mq.NewConsumer(rmqConn)
	// consumer.Start()
	return &ServiceContext{
		Config:             c,
		DBEngine:           eng,
		RedisClient:        deps.initRedis(c.Redis.Addr, c.Redis.Password, c.Redis.DB),
		RabbitMQConn:       rmqConn,
		RabbitMQChannel:    rmqCh,
		FileAuthMiddleware: deps.newFileAuth(c.Auth.AccessSecret, c.Auth.AccessExpire),
	}
}

func NewServiceContextWithDeps(c config.Config, db *xorm.Engine, redis RedisClient, fileAuth rest.Middleware) *ServiceContext {
	return &ServiceContext{
		Config:             c,
		DBEngine:           db,
		RedisClient:        redis,
		RabbitMQConn:       global.RmqConn,
		RabbitMQChannel:    global.RmqCh,
		FileAuthMiddleware: fileAuth,
	}
}
