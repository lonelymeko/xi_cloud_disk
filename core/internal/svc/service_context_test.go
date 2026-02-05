package svc

import (
	"cloud_disk/core/internal/config"
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/rest"
	"xorm.io/xorm"
)

// TestNewServiceContextWithDeps 验证使用自定义依赖构建上下文。
func TestNewServiceContextWithDeps(t *testing.T) {
	cfg := config.Config{}
	ctx := NewServiceContextWithDeps(cfg, nil, nil, func(next http.HandlerFunc) http.HandlerFunc {
		return next
	})
	if ctx == nil {
		t.Fatal("context is nil")
	}
	if !reflect.DeepEqual(ctx.Config, cfg) {
		t.Fatal("config mismatch")
	}
}

// TestNewServiceContextUsesDeps 验证默认依赖被正确调用。
func TestNewServiceContextUsesDeps(t *testing.T) {
	oldDeps := deps
	t.Cleanup(func() { deps = oldDeps })

	cfg := config.Config{}
	cfg.MySQL.DataSource = "ds"
	cfg.Redis.Addr = "addr"
	cfg.Redis.Password = "pwd"
	cfg.Redis.DB = 2
	cfg.Auth.AccessSecret = "secret"
	cfg.Auth.AccessExpire = 3600

	calledInitDB := false
	calledEnsureSchema := false
	calledEnsureTablesHealth := false
	calledEnsureDefaultAdmin := false
	calledInitRedis := false
	calledNewFileAuth := false

	fakeDB := &xorm.Engine{}
	fakeRedis := &fakeRedisClient{}

	deps = serviceDeps{
		initDB: func(dataSource string) *xorm.Engine {
			if dataSource != "ds" {
				t.Fatalf("dataSource mismatch: %s", dataSource)
			}
			calledInitDB = true
			return fakeDB
		},
		initRedis: func(addr, password string, db int) RedisClient {
			if addr != "addr" || password != "pwd" || db != 2 {
				t.Fatalf("redis args mismatch: %s %s %d", addr, password, db)
			}
			calledInitRedis = true
			return fakeRedis
		},
		newFileAuth: func(secret string, expire int64) rest.Middleware {
			if secret != "secret" || expire != 3600 {
				t.Fatalf("auth args mismatch: %s %d", secret, expire)
			}
			calledNewFileAuth = true
			return func(next http.HandlerFunc) http.HandlerFunc { return next }
		},
		ensureSchema: func(eng *xorm.Engine) error {
			if eng != fakeDB {
				t.Fatal("ensure schema engine mismatch")
			}
			calledEnsureSchema = true
			return nil
		},
		ensureTablesHealth: func(eng *xorm.Engine) error {
			if eng != fakeDB {
				t.Fatal("ensure tables health engine mismatch")
			}
			calledEnsureTablesHealth = true
			return nil
		},
		ensureDefaultAdmin: func(eng *xorm.Engine) error {
			if eng != fakeDB {
				t.Fatal("ensure admin engine mismatch")
			}
			calledEnsureDefaultAdmin = true
			return nil
		},
	}

	ctx := NewServiceContext(cfg)
	if ctx == nil {
		t.Fatal("context is nil")
	}
	if ctx.DBEngine != fakeDB {
		t.Fatal("db engine mismatch")
	}
	if ctx.RedisClient != fakeRedis {
		t.Fatal("redis client mismatch")
	}
	if ctx.FileAuthMiddleware == nil {
		t.Fatal("middleware is nil")
	}
	if !calledInitDB || !calledEnsureSchema || !calledEnsureTablesHealth || !calledEnsureDefaultAdmin || !calledInitRedis || !calledNewFileAuth {
		t.Fatal("deps not fully used")
	}
}

// fakeRedisClient Redis 客户端测试替身。
type fakeRedisClient struct{}

func (f *fakeRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	return redis.NewStringResult("", nil)
}
func (f *fakeRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return redis.NewStatusResult("", nil)
}
func (f *fakeRedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	return redis.NewBoolResult(true, nil)
}
func (f *fakeRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (f *fakeRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	return redis.NewStatusResult("PONG", nil)
}
