package handler

import (
	"net/http"

	"cloud_disk/core/common"
	"cloud_disk/core/internal/logic"
	"cloud_disk/core/internal/svc"
)

func AgentListSessionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewAgentChatLogic(r.Context(), svcCtx)
		resp, err := l.ListSessions()
		common.Response(r, w, resp, err)
	}
}
