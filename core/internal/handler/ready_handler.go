package handler

import (
  "cloud_disk/core/internal/svc"
  "cloud_disk/core/utils"
  "context"
  "net/http"
  "time"
  "github.com/zeromicro/go-zero/rest/httpx"
)

func ReadyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
  return func(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
    defer cancel()
    dbErr := svcCtx.DBEngine.Ping()
    redisErr := svcCtx.RedisClient.Ping(ctx).Err()
    ossErr := utils.OSSConnectivity(ctx)
    ready := dbErr == nil && redisErr == nil && ossErr == nil
    if r.Method == http.MethodHead {
      if ready {
        w.WriteHeader(http.StatusNoContent)
      } else {
        w.WriteHeader(http.StatusServiceUnavailable)
      }
      return
    }
    if ready {
      httpx.WriteJson(w, http.StatusOK, map[string]any{"ready": true})
      return
    }
    httpx.WriteJson(w, http.StatusServiceUnavailable, map[string]any{
      "ready":  false,
      "errors": map[string]any{"database": dbErr, "redis": redisErr, "oss": ossErr},
    })
  }
}

