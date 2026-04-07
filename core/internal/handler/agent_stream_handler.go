package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"cloud_disk/core/common"
	"cloud_disk/core/internal/agent/einoagent"
	"cloud_disk/core/internal/logic"
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
)

type agentStreamPayload struct {
	Type             string                       `json:"type"`
	Role             string                       `json:"role,omitempty"`
	Content          string                       `json:"content,omitempty"`
	ToolName         string                       `json:"tool_name,omitempty"`
	ArgumentsJSON    string                       `json:"arguments_json,omitempty"`
	Session          *types.AgentSession          `json:"session,omitempty"`
	PendingInterrupt *types.AgentPendingInterrupt `json:"pending_interrupt,omitempty"`
	ReferencedFiles  []*types.AgentFileReference  `json:"referenced_files,omitempty"`
	Reply            string                       `json:"reply,omitempty"`
	Error            string                       `json:"error,omitempty"`
}

func AgentChatStreamHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AgentChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			common.Response(r, w, nil, err)
			return
		}
		if !prepareSSE(w) {
			common.Response(r, w, nil, fmt.Errorf("streaming is not supported by the server"))
			return
		}

		l := logic.NewAgentChatLogic(r.Context(), svcCtx)
		if _, err := l.StreamChat(&req, func(chunk einoagent.StreamChunk) error {
			return writeSSE(w, toAgentStreamPayload(chunk))
		}); err != nil {
			_ = writeSSE(w, agentStreamPayload{Type: "error", Error: err.Error()})
		}
	}
}

func AgentResumeStreamHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AgentResumeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			common.Response(r, w, nil, err)
			return
		}
		if !prepareSSE(w) {
			common.Response(r, w, nil, fmt.Errorf("streaming is not supported by the server"))
			return
		}

		l := logic.NewAgentChatLogic(r.Context(), svcCtx)
		if _, err := l.StreamResume(&req, func(chunk einoagent.StreamChunk) error {
			return writeSSE(w, toAgentStreamPayload(chunk))
		}); err != nil {
			_ = writeSSE(w, agentStreamPayload{Type: "error", Error: err.Error()})
		}
	}
}

func prepareSSE(w http.ResponseWriter) bool {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return false
	}
	headers := w.Header()
	headers.Set("Content-Type", "text/event-stream")
	headers.Set("Cache-Control", "no-cache")
	headers.Set("Connection", "keep-alive")
	headers.Set("X-Accel-Buffering", "no")
	flusher.Flush()
	return true
}

func writeSSE(w http.ResponseWriter, payload agentStreamPayload) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming is not supported by the server")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", raw); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

func toAgentStreamPayload(chunk einoagent.StreamChunk) agentStreamPayload {
	payload := agentStreamPayload{
		Type:          chunk.Type,
		Role:          chunk.Role,
		Content:       chunk.Content,
		ToolName:      chunk.ToolName,
		ArgumentsJSON: chunk.ArgumentsJSON,
		Reply:         chunk.Reply,
		Error:         chunk.Error,
	}
	if chunk.Session != nil {
		payload.Session = &types.AgentSession{
			ID:                 chunk.Session.ID,
			Title:              chunk.Session.Title,
			PendingInterruptID: chunk.Session.PendingInterruptID,
			UpdatedAt:          chunk.Session.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
	}
	if chunk.PendingInterrupt != nil {
		payload.PendingInterrupt = &types.AgentPendingInterrupt{
			InterruptID:   chunk.PendingInterrupt.InterruptID,
			ToolName:      chunk.PendingInterrupt.ToolName,
			ArgumentsJSON: chunk.PendingInterrupt.ArgumentsJSON,
		}
	}
	if len(chunk.ReferencedFiles) > 0 {
		payload.ReferencedFiles = make([]*types.AgentFileReference, 0, len(chunk.ReferencedFiles))
		for _, item := range chunk.ReferencedFiles {
			file := item
			payload.ReferencedFiles = append(payload.ReferencedFiles, &types.AgentFileReference{
				Name:     file.Name,
				URL:      file.URL,
				MIMEType: file.MIMEType,
			})
		}
	}
	return payload
}
