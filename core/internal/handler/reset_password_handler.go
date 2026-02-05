package handler

import (
	"net/http"

	"cloud_disk/core/common"
	"cloud_disk/core/internal/logic"
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// ResetPasswordHandler 重置密码处理入口。
func ResetPasswordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ResetPasswordRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewResetPasswordLogic(r.Context(), svcCtx)
		resp, err := l.ResetPassword(&req)
		common.Response(r, w, resp, err)
	}
}
