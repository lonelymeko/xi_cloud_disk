package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"

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

var agentStreamHeartbeatInterval = 15 * time.Second

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
		streamAgent(w, r, func(emit func(einoagent.StreamChunk) error) error {
			_, err := l.StreamChat(&req, emit)
			return err
		})
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
		streamAgent(w, r, func(emit func(einoagent.StreamChunk) error) error {
			_, err := l.StreamResume(&req, emit)
			return err
		})
	}
}

func streamAgent(w http.ResponseWriter, r *http.Request, run func(func(einoagent.StreamChunk) error) error) {
	ctx := r.Context()
	msgCh := make(chan agentStreamPayload, 16)
	runErrCh := make(chan error, 1)

	go func() {
		defer close(msgCh)
		if err := sendStreamPayload(ctx, msgCh, agentStreamPayload{Type: "stream_ready"}); err != nil {
			runErrCh <- err
			return
		}
		err := run(func(chunk einoagent.StreamChunk) error {
			return sendStreamPayload(ctx, msgCh, toAgentStreamPayload(chunk))
		})
		if err != nil && !isClientGone(err) {
			_ = sendStreamPayload(ctx, msgCh, agentStreamPayload{Type: "error", Error: err.Error()})
		}
		runErrCh <- err
	}()

	writeErr := writeSSELoop(ctx, w, msgCh)
	runErr := <-runErrCh
	if isClientGone(writeErr) || isClientGone(runErr) {
		return
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

func writeSSELoop(ctx context.Context, w http.ResponseWriter, msgCh <-chan agentStreamPayload) error {
	ticker := time.NewTicker(agentStreamHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case payload, ok := <-msgCh:
			if !ok {
				return nil
			}
			if err := writeSSE(w, payload); err != nil {
				return err
			}
		case <-ticker.C:
			if err := writeSSE(w, agentStreamPayload{Type: "heartbeat"}); err != nil {
				return err
			}
		}
	}
}

func sendStreamPayload(ctx context.Context, msgCh chan<- agentStreamPayload, payload agentStreamPayload) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case msgCh <- payload:
		return nil
	}
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

func isClientGone(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return true
	}
	if errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNRESET) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "broken pipe") || strings.Contains(msg, "connection reset by peer")
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
