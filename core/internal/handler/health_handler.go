package handler

import (
	"net/http"

	"cloud_disk/core/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// HealthHandler 健康检查处理入口。
func HealthHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		httpx.WriteJson(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
