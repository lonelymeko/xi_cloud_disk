package handler

import (
	"net/http"

	"cloud_disk/core/common"
	"cloud_disk/core/internal/logic"
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func AgentSessionDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AgentSessionDetailRequest
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(r, w, nil, err)
			return
		}
		l := logic.NewAgentChatLogic(r.Context(), svcCtx)
		resp, err := l.GetSessionDetail(req.SessionID)
		common.Response(r, w, resp, err)
	}
}
