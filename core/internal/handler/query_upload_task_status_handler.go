package handler

import (
	"net/http"

	"cloud_disk/core/common"
	"cloud_disk/core/internal/logic"
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// QueryUploadTaskStatusHandler 查询上传任务状态入口。
func QueryUploadTaskStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.QueryUploadTaskStatusRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		l := logic.NewQueryUploadTaskStatusLogic(r.Context(), svcCtx)
		resp, err := l.QueryUploadTaskStatus(&req)
		common.Response(r, w, resp, err)
	}
}
