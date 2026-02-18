// goctl 生成代码，可安全编辑。
// goctl 1.9.2

package svc

import (
	"cloud_disk/core/global"
	"cloud_disk/core/internal/config"
	"cloud_disk/core/internal/filter"
	"cloud_disk/core/internal/middleware"
	"cloud_disk/core/utils"
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"xorm.io/xorm"
)

// ServiceContext 服务上下文。
type ServiceContext struct {
	Config             config.Config
	DBEngine           *xorm.Engine
	RedisClient        RedisClient
	RabbitMQConn       *amqp091.Connection
	RabbitMQChannel    *amqp091.Channel
	FileAuthMiddleware rest.Middleware
	MyBloomFilter      *filter.MyBloomFilter
}

// RedisClient Redis 客户端最小接口。
type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Ping(ctx context.Context) *redis.StatusCmd
}

// serviceDeps 依赖注入集合。
type serviceDeps struct {
	initDB             func(string) *xorm.Engine
	initRedis          func(string, string, int) RedisClient
	newFileAuth        func(string, int64) rest.Middleware
	ensureSchema       func(*xorm.Engine) error
	ensureTablesHealth func(*xorm.Engine) error
	ensureDefaultAdmin func(*xorm.Engine) error
	initRabbitMQ       func(string, int, string, string, string) (*amqp091.Connection, *amqp091.Channel)
}

// deps 默认依赖实现。
var deps = serviceDeps{
	initDB:       global.Init,
	initRedis:    func(addr, password string, db int) RedisClient { return global.InitRedis(addr, password, db) },
	initRabbitMQ: global.InitRabbitMQ,
	newFileAuth: func(secret string, expire int64) rest.Middleware {
		return middleware.NewFileAuthMiddleware(secret, expire).Handle
	},
	ensureSchema:       utils.EnsureSchema,
	ensureTablesHealth: utils.TablesHealthy,
	ensureDefaultAdmin: utils.EnsureDefaultAdmin,
}

// NewServiceContext 创建服务上下文。
func NewServiceContext(c config.Config) *ServiceContext {
	eng := deps.initDB(c.MySQL.DataSource)
	_ = deps.ensureSchema(eng)
	if err := deps.ensureTablesHealth(eng); err != nil {
		logx.Errorf("tables health check failed: %v", err)
	}
	_ = deps.ensureDefaultAdmin(eng)
	var rmqConn *amqp091.Connection
	var rmqCh *amqp091.Channel
	if c.RabbitMQ.Host != "" && c.RabbitMQ.Port != 0 && c.RabbitMQ.Username != "" {
		rmqConn, rmqCh = deps.initRabbitMQ(c.RabbitMQ.Host, c.RabbitMQ.Port, c.RabbitMQ.Username, c.RabbitMQ.Password, c.RabbitMQ.Vhost)
	}
	// 注册布隆过滤器
	bloomFilter := filter.NewBloomFilter(eng)
	// 启动布隆过滤器定期保存任务
	stopBloomTask := startBloomFilterPersistTask(bloomFilter)
	// 注册优雅关闭处理
	registerGracefulShutdown(stopBloomTask, bloomFilter)
	return &ServiceContext{
		Config:             c,
		DBEngine:           eng,
		RedisClient:        deps.initRedis(c.Redis.Addr, c.Redis.Password, c.Redis.DB),
		RabbitMQConn:       rmqConn,
		RabbitMQChannel:    rmqCh,
		FileAuthMiddleware: deps.newFileAuth(c.Auth.AccessSecret, c.Auth.AccessExpire),
		MyBloomFilter:      bloomFilter,
	}
}

// NewServiceContextWithDeps 使用自定义依赖创建服务上下文。
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

// startBloomFilterPersistTask 启动布隆过滤器定期持久化任务
// 返回停止函数用于优雅关闭
func startBloomFilterPersistTask(bloomFilter *filter.MyBloomFilter) func() {
	// 设置定期保存间隔（默认30分钟）
	saveInterval := 30 * time.Minute
	if intervalStr := os.Getenv("BLOOM_FILTER_SAVE_INTERVAL"); intervalStr != "" {
		if interval, err := time.ParseDuration(intervalStr); err == nil {
			saveInterval = interval
			logx.Infof("布隆过滤器保存间隔设置为: %v", saveInterval)
		} else {
			logx.Errorf("解析BLOOM_FILTER_SAVE_INTERVAL失败: %v，使用默认值30分钟", err)
		}
	}

	// 创建停止通道
	stopChan := make(chan struct{})
	var stopped bool

	go func() {
		ticker := time.NewTicker(saveInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if bloomFilter != nil {
					logx.Info("开始定期保存布隆过滤器到文件")
					bloomFilter.SaveFilterToFile()
					logx.Info("布隆过滤器定期保存完成")
				} else {
					logx.Info("布隆过滤器未初始化，跳过定期保存")
				}
			case <-stopChan:
				logx.Info("收到停止信号，正在保存布隆过滤器...")
				if bloomFilter != nil && !stopped {
					bloomFilter.SaveFilterToFile()
					stopped = true
					logx.Info("应用退出时布隆过滤器保存完成")
				}
				return
			case <-context.Background().Done():
				logx.Info("布隆过滤器定期保存任务停止")
				return
			}
		}
	}()

	logx.Infof("布隆过滤器定期保存任务已启动，间隔: %v", saveInterval)

	// 返回停止函数
	return func() {
		close(stopChan)
	}
}

// registerGracefulShutdown 注册优雅关闭处理
func registerGracefulShutdown(stopFunc func(), bloomFilter *filter.MyBloomFilter) {
	go func() {
		// 监听系统信号
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

		// 等待信号
		sig := <-c
		logx.Infof("收到退出信号: %v，开始优雅关闭...", sig)

		// 停止布隆过滤器任务并保存
		if stopFunc != nil {
			stopFunc()
		}

		logx.Info("优雅关闭完成")
	}()
}
