// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package handler

import (
	"net/http"

	"cloud_disk/core/internal/logic"
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"

	"cloud_disk/core/common"
)

func CreateShareRecordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CreateShareRecordRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewCreateShareRecordLogic(r.Context(), svcCtx)
		resp, err := l.CreateShareRecord(&req)
		//if err != nil {
		//	httpx.ErrorCtx(r.Context(), w, err)
		//} else {
		//	httpx.OkJsonCtx(r.Context(), w, resp)
		//}
		common.Response(r, w, resp, err)
	}
}
