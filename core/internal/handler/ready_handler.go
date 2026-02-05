package handler

import (
	"context"
	"net/http"
	"time"

	"cloud_disk/core/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ReadyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		dbOk := true
		if svcCtx.DBEngine != nil {
			if err := svcCtx.DBEngine.Ping(); err != nil {
				dbOk = false
			}
		} else {
			dbOk = false
		}
		redisOk := true
		if svcCtx.RedisClient != nil {
			if err := svcCtx.RedisClient.Ping(ctx).Err(); err != nil {
				redisOk = false
			}
		} else {
			redisOk = false
		}
		status := "ready"
		if !dbOk || !redisOk {
			status = "degraded"
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"status": status,
			"checks": map[string]interface{}{
				"database": map[string]bool{"ok": dbOk},
				"redis":    map[string]bool{"ok": redisOk},
			},
		})
	}
}
