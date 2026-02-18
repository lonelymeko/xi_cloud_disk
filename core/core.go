// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"cloud_disk/core/common"
	"cloud_disk/core/internal/config"
	"cloud_disk/core/internal/handler"
	"cloud_disk/core/internal/logic"
	"cloud_disk/core/internal/mq"
	"cloud_disk/core/internal/svc"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud_disk/core/utils"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "core/etc/core-api.yaml", "the config file")

func main() {
	flag.Parse()

	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		logx.SetLevel(logx.DebugLevel)
	case "error":
		logx.SetLevel(logx.ErrorLevel)
	case "severe":
		logx.SetLevel(logx.SevereLevel)
	default:
		logx.SetLevel(logx.InfoLevel)
	}

	var c config.Config
	conf.MustLoad(*configFile, &c)

	if os.Getenv("DB_HOST") != "" {
		host := os.Getenv("DB_HOST")
		port := os.Getenv("DB_PORT")
		user := os.Getenv("DB_USER")
		pass := os.Getenv("DB_PASSWORD")
		name := os.Getenv("DB_NAME")
		if host != "" && port != "" && user != "" && name != "" {
			c.MySQL.DataSource = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, pass, host, port, name)
		}
	}
	if os.Getenv("REDIS_HOST") != "" {
		rhost := os.Getenv("REDIS_HOST")
		rport := os.Getenv("REDIS_PORT")
		rpass := os.Getenv("REDIS_PASSWORD")
		rdbStr := os.Getenv("REDIS_DB")
		if rhost != "" && rport != "" {
			c.Redis.Addr = fmt.Sprintf("%s:%s", rhost, rport)
		}
		c.Redis.Password = rpass
		if rdbStr != "" {
			if v, err := strconv.Atoi(rdbStr); err == nil {
				c.Redis.DB = v
			}
		}
	}
	if os.Getenv("RABBITMQ_HOST") != "" {
		host := os.Getenv("RABBITMQ_HOST")
		portStr := os.Getenv("RABBITMQ_PORT")
		user := os.Getenv("RABBITMQ_USERNAME")
		pass := os.Getenv("RABBITMQ_PASSWORD")
		vhost := os.Getenv("RABBITMQ_VHOST")
		if host != "" {
			c.RabbitMQ.Host = host
		}
		if portStr != "" {
			if v, err := strconv.Atoi(portStr); err == nil {
				c.RabbitMQ.Port = v
			}
		}
		if user != "" {
			c.RabbitMQ.Username = user
		}
		if pass != "" {
			c.RabbitMQ.Password = pass
		}
		if vhost != "" {
			c.RabbitMQ.Vhost = vhost
		}
	}

	corsEnv := os.Getenv("CORS_ALLOW_ORIGINS")
	origins := make([]string, 0)
	if corsEnv != "" {
		for _, v := range strings.Split(corsEnv, ",") {
			v = strings.TrimSpace(v)
			if v != "" {
				origins = append(origins, v)
			}
		}
	}
	if len(origins) == 0 {
		origins = append(origins, "*")
	}

	server := rest.MustNewServer(
		c.RestConf,
		rest.WithUnauthorizedCallback(JwtUnauthorizedResult),
		rest.WithCustomCors(func(header http.Header) {
			header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Token")
			header.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,HEAD,OPTIONS")
		}, nil, origins...),
	)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)

	if ctx.RabbitMQChannel != nil {
		consumer := mq.NewConsumer(context.Background(), ctx, ctx.RabbitMQChannel)
		consumer.Start()
	} else {
		logx.Info("RabbitMQ disabled: channel not initialized")
	}
	logic.StartRecycleJob(context.Background(), ctx)
	handler.RegisterHandlers(server, ctx)

	checkCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	type res struct {
		name string
		ok   bool
		err  error
	}
	ch := make(chan res, 4)
	go func() { err := ctx.DBEngine.Ping(); ch <- res{"database", err == nil, err} }()
	go func() { err := ctx.RedisClient.Ping(checkCtx).Err(); ch <- res{"redis", err == nil, err} }()
	go func() { err := utils.EmailConnectivity(checkCtx); ch <- res{"email", err == nil, err} }()
	go func() { err := utils.OSSConnectivity(checkCtx); ch <- res{"oss", err == nil, err} }()
	for i := 0; i < 4; i++ {
		r := <-ch
		if r.ok {
			logx.Infof("startup %s ok", r.name)
		} else {
			logx.Infof("startup %s failed", r.name)
			if r.err != nil {
				logx.Errorf("startup %s error: %v", r.name, r.err)
			}
		}
	}

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}

// JwtUnauthorizedResult 鉴权失败
func JwtUnauthorizedResult(w http.ResponseWriter, r *http.Request, err error) {
	fmt.Println("JwtUnauthorizedResult:", err)
	httpx.WriteJson(w, http.StatusOK, common.Body{
		Code: 401,
		Msg:  "鉴权失败",
		Data: nil,
	})
}
