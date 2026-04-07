package handler

import (
	"encoding/json"
	"net/http"

	"cloud_disk/core/common"
	"cloud_disk/core/internal/logic"
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
)

func AgentChatHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AgentChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			common.Response(r, w, nil, err)
			return
		}
		l := logic.NewAgentChatLogic(r.Context(), svcCtx)
		resp, err := l.Chat(&req)
		common.Response(r, w, resp, err)
	}
}
