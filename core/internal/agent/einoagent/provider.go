package einoagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud_disk/core/common"
	"cloud_disk/core/internal/agent/einoagent/tools"
	"cloud_disk/core/models"
	"cloud_disk/core/utils"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	openaiext "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"xorm.io/xorm"
)

const defaultMaxIteration = 50

// Config 定义 cloud_disk 侧 Agent 的基础配置。
type Config struct {
	Enabled      bool
	ProjectRoot  string
	SessionDir   string
	SkillsDir    string
	MaxIteration int
}

type Dependencies struct {
	DBEngine *xorm.Engine
}

type Attachment struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	MIMEType string `json:"mime_type,omitempty"`
}

type ChatEvent struct {
	Type          string `json:"type"`
	Role          string `json:"role,omitempty"`
	Content       string `json:"content,omitempty"`
	ToolName      string `json:"tool_name,omitempty"`
	ArgumentsJSON string `json:"arguments_json,omitempty"`
}

type PendingInterrupt struct {
	InterruptID   string `json:"interrupt_id"`
	ToolName      string `json:"tool_name"`
	ArgumentsJSON string `json:"arguments_json"`
}

type Session struct {
	ID                 string    `json:"id"`
	Title              string    `json:"title"`
	PendingInterruptID string    `json:"pending_interrupt_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type SessionMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type SessionDetail struct {
	Session  Session          `json:"session"`
	Messages []SessionMessage `json:"messages,omitempty"`
}

type ChatRequest struct {
	SessionID   string
	UserName    string
	Message     string
	Attachments []Attachment
}

type ResumeRequest struct {
	SessionID   string
	UserName    string
	InterruptID string
	Approved    bool
	Reason      string
}

type ChatResponse struct {
	Session          Session           `json:"session"`
	Reply            string            `json:"reply,omitempty"`
	Events           []ChatEvent       `json:"events,omitempty"`
	PendingInterrupt *PendingInterrupt `json:"pending_interrupt,omitempty"`
	ReferencedFiles  []Attachment      `json:"referenced_files,omitempty"`
}

type StreamChunk struct {
	Type             string            `json:"type"`
	Role             string            `json:"role,omitempty"`
	Content          string            `json:"content,omitempty"`
	ToolName         string            `json:"tool_name,omitempty"`
	ArgumentsJSON    string            `json:"arguments_json,omitempty"`
	Session          *Session          `json:"session,omitempty"`
	PendingInterrupt *PendingInterrupt `json:"pending_interrupt,omitempty"`
	ReferencedFiles  []Attachment      `json:"referenced_files,omitempty"`
	Reply            string            `json:"reply,omitempty"`
	Error            string            `json:"error,omitempty"`
}

// Provider 是 ServiceContext 暴露给业务层的 Agent 提供器。
type Provider interface {
	Enabled() bool
	Name() string
	Config() Config
	CreateSession(ctx context.Context, userIdentity string) (Session, error)
	ListSessions(ctx context.Context, userIdentity string) ([]Session, error)
	GetSessionDetail(ctx context.Context, userIdentity, sessionID string) (*SessionDetail, error)
	Chat(ctx context.Context, userIdentity string, req ChatRequest) (*ChatResponse, error)
	Resume(ctx context.Context, userIdentity string, req ResumeRequest) (*ChatResponse, error)
	StreamChat(ctx context.Context, userIdentity string, req ChatRequest, emit func(StreamChunk) error) (*ChatResponse, error)
	StreamResume(ctx context.Context, userIdentity string, req ResumeRequest, emit func(StreamChunk) error) (*ChatResponse, error)
}

type provider struct {
	cfg      Config
	deps     Dependencies
	runner   *adk.Runner
	sessions *sessionStore
}

type sessionState struct {
	id               string
	userIdentity     string
	title            string
	pendingInterrupt string
	createdAt        time.Time
	updatedAt        time.Time
	messages         []*schema.Message
}

type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*sessionState
}

type checkpointStore struct {
	db *xorm.Engine
}

type approvalMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
}

type safeToolMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
}

type approvalInfo struct {
	ToolName        string
	ArgumentsInJSON string
}

type approvalResult struct {
	Approved         bool
	DisapproveReason *string
}

func init() {
	schema.Register[*approvalInfo]()
}

// NewProvider 构建一个最小可注入的 Eino Agent Provider。
func NewProvider(ctx context.Context, cfg Config, deps Dependencies) (Provider, error) {
	cfg.ProjectRoot = cleanAbsPath(cfg.ProjectRoot)
	cfg.SessionDir = cleanAbsPath(cfg.SessionDir)
	cfg.SkillsDir = cleanAbsPath(cfg.SkillsDir)
	if cfg.MaxIteration <= 0 {
		cfg.MaxIteration = defaultMaxIteration
	}

	p := &provider{
		cfg:      cfg,
		deps:     deps,
		sessions: &sessionStore{sessions: make(map[string]*sessionState)},
	}
	if !cfg.Enabled {
		return p, nil
	}

	backend, err := localbk.NewBackend(ctx, &localbk.Config{})
	if err != nil {
		return nil, err
	}

	cm, err := openaiext.NewChatModel(ctx, &openaiext.ChatModelConfig{
		APIKey:  getenv("OPENAI_API_KEY"),
		Model:   firstEnv("OPENAI_MODEL", "VISION_MODEL"),
		BaseURL: getenv("OPENAI_BASE_URL"),
		ByAzure: strings.EqualFold(getenv("OPENAI_BY_AZURE"), "true"),
	})
	if err != nil {
		return nil, err
	}

	agent, err := deep.New(ctx, &deep.Config{
		Name:           "CloudDiskAgent",
		Description:    "An agent that can answer questions about cloud_disk user files and operate on folders through dedicated tools.",
		Instruction:    buildInstruction(cfg.ProjectRoot),
		ChatModel:      cm,
		Backend:        backend,
		StreamingShell: backend,
		MaxIteration:   cfg.MaxIteration,
		Handlers:       []adk.ChatModelAgentMiddleware{&approvalMiddleware{}, &safeToolMiddleware{}},
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []einotool.BaseTool{
					tools.NewDocumentAnswerTool(cm),
					tools.NewImageAnalysisTool(cm),
					tools.NewVideoSummaryTool(cm),
					tools.NewListFilesTool(p.listFiles),
					tools.NewCreateFolderTool(p.createFolder),
					tools.NewMoveFileTool(p.moveFile),
				},
			},
		},
		ModelRetryConfig: &adk.ModelRetryConfig{
			MaxRetries: 3,
			IsRetryAble: func(_ context.Context, err error) bool {
				msg := err.Error()
				return strings.Contains(msg, "429") || strings.Contains(msg, "Too Many Requests")
			},
		},
	})
	if err != nil {
		return nil, err
	}

	p.runner = adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
		CheckPointStore: &checkpointStore{db: deps.DBEngine},
	})

	return p, nil
}

func (p *provider) Enabled() bool {
	return p.cfg.Enabled && p.runner != nil
}

func (p *provider) Name() string {
	return "eino"
}

func (p *provider) Config() Config {
	return p.cfg
}

func (p *provider) CreateSession(_ context.Context, userIdentity string) (Session, error) {
	sessionID := uuid.New().String()
	state := p.sessions.getOrCreate(userIdentity, sessionID)
	state.title = "New Chat"
	state.createdAt = time.Now()
	state.updatedAt = state.createdAt
	if err := p.upsertSessionRecord(userIdentity, state); err != nil {
		return Session{}, err
	}
	return toSession(state), nil
}

func (p *provider) ListSessions(_ context.Context, userIdentity string) ([]Session, error) {
	if p.deps.DBEngine == nil {
		return p.sessions.list(userIdentity), nil
	}
	rows := make([]models.AgentChatSession, 0)
	if err := p.deps.DBEngine.Where("user_identity = ?", userIdentity).Desc("updated_at").Find(&rows); err != nil {
		return nil, err
	}
	items := make([]Session, 0, len(rows))
	for _, row := range rows {
		items = append(items, sessionFromRow(row))
	}
	return items, nil
}

func (p *provider) GetSessionDetail(_ context.Context, userIdentity, sessionID string) (*SessionDetail, error) {
	state, err := p.loadSessionState(userIdentity, sessionID, false)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, errors.New("session not found")
	}
	detail := &SessionDetail{
		Session:  toSession(state),
		Messages: make([]SessionMessage, 0, len(state.messages)),
	}
	for _, msg := range state.messages {
		if msg == nil {
			continue
		}
		if msg.Role != schema.User && msg.Role != schema.Assistant {
			continue
		}
		detail.Messages = append(detail.Messages, SessionMessage{
			Role:      string(msg.Role),
			Content:   strings.TrimSpace(msg.Content),
			CreatedAt: state.updatedAt,
		})
	}
	return detail, nil
}

func (p *provider) Chat(ctx context.Context, userIdentity string, req ChatRequest) (*ChatResponse, error) {
	if !p.Enabled() {
		return nil, errors.New("agent is disabled")
	}
	if strings.TrimSpace(req.Message) == "" {
		return nil, errors.New("message is required")
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	state, err := p.loadSessionState(userIdentity, sessionID, true)
	if err != nil {
		return nil, err
	}
	p.sessions.mu.Lock()
	if state.pendingInterrupt != "" {
		p.sessions.mu.Unlock()
		return nil, errors.New("session has a pending interrupt; resume it before sending a new message")
	}

	userMsg := schema.UserMessage(buildUserPrompt(req.Message, req.Attachments))
	state.messages = append(state.messages, userMsg)
	if state.title == "" || state.title == "New Chat" {
		state.title = summarizeTitle(req.Message)
	}
	state.updatedAt = time.Now()
	history := cloneMessages(state.messages)
	p.sessions.mu.Unlock()
	if err := p.appendSessionMessage(userIdentity, sessionID, userMsg); err != nil {
		return nil, err
	}
	if err := p.upsertSessionRecord(userIdentity, state); err != nil {
		return nil, err
	}

	iter := p.runner.Run(ctx, history, adk.WithCheckPointID(checkpointKey(userIdentity, sessionID)))
	reply, events, interrupt, err := collectRunResult(iter)
	if err != nil {
		return nil, err
	}

	p.sessions.mu.Lock()
	defer p.sessions.mu.Unlock()
	state = p.sessions.getOrCreateLocked(userIdentity, sessionID)
	state.updatedAt = time.Now()
	if interrupt != nil {
		state.pendingInterrupt = interrupt.InterruptID
	} else {
		state.pendingInterrupt = ""
		if strings.TrimSpace(reply) != "" {
			assistant := schema.AssistantMessage(reply, nil)
			state.messages = append(state.messages, assistant)
			if err := p.appendSessionMessage(userIdentity, sessionID, assistant); err != nil {
				return nil, err
			}
		}
	}
	if err := p.upsertSessionRecord(userIdentity, state); err != nil {
		return nil, err
	}

	return &ChatResponse{
		Session:          toSession(state),
		Reply:            reply,
		Events:           events,
		PendingInterrupt: interrupt,
		ReferencedFiles:  req.Attachments,
	}, nil
}

func (p *provider) StreamChat(ctx context.Context, userIdentity string, req ChatRequest, emit func(StreamChunk) error) (*ChatResponse, error) {
	begin := time.Now()
	logx.Infof("agent stream chat start: session_id=%s user=%s attachments=%d message_len=%d", strings.TrimSpace(req.SessionID), userIdentity, len(req.Attachments), len(strings.TrimSpace(req.Message)))
	defer func() {
		logx.Infof("agent stream chat finished: session_id=%s user=%s duration=%s", strings.TrimSpace(req.SessionID), userIdentity, time.Since(begin))
	}()
	if !p.Enabled() {
		return nil, errors.New("agent is disabled")
	}
	if strings.TrimSpace(req.Message) == "" {
		return nil, errors.New("message is required")
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	state, err := p.loadSessionState(userIdentity, sessionID, true)
	if err != nil {
		return nil, err
	}
	logx.Infof("agent stream chat stage=load_session session_id=%s user=%s duration=%s", sessionID, userIdentity, time.Since(begin))
	p.sessions.mu.Lock()
	if state.pendingInterrupt != "" {
		p.sessions.mu.Unlock()
		return nil, errors.New("session has a pending interrupt; resume it before sending a new message")
	}

	userMsg := schema.UserMessage(buildUserPrompt(req.Message, req.Attachments))
	state.messages = append(state.messages, userMsg)
	if state.title == "" || state.title == "New Chat" {
		state.title = summarizeTitle(req.Message)
	}
	state.updatedAt = time.Now()
	history := cloneMessages(state.messages)
	currentSession := toSession(state)
	p.sessions.mu.Unlock()
	if err := p.appendSessionMessage(userIdentity, sessionID, userMsg); err != nil {
		return nil, err
	}
	if err := p.upsertSessionRecord(userIdentity, state); err != nil {
		return nil, err
	}
	logx.Infof("agent stream chat stage=persist_user_message session_id=%s user=%s duration=%s", sessionID, userIdentity, time.Since(begin))

	if emit != nil {
		if err := emit(StreamChunk{Type: "session", Session: &currentSession}); err != nil {
			return nil, err
		}
		if len(req.Attachments) > 0 {
			if err := emit(StreamChunk{Type: "referenced_files", ReferencedFiles: req.Attachments}); err != nil {
				return nil, err
			}
		}
	}

	iter := p.runner.Run(ctx, history, adk.WithCheckPointID(checkpointKey(userIdentity, sessionID)))
	logx.Infof("agent stream chat stage=runner_started session_id=%s user=%s duration=%s", sessionID, userIdentity, time.Since(begin))
	reply, events, interrupt, err := streamRunResult(iter, emit)
	if err != nil {
		logx.Errorf("agent stream chat stage=runner_failed session_id=%s user=%s duration=%s err=%v", sessionID, userIdentity, time.Since(begin), err)
		return nil, err
	}
	logx.Infof("agent stream chat stage=runner_completed session_id=%s user=%s duration=%s events=%d interrupt=%t reply_len=%d", sessionID, userIdentity, time.Since(begin), len(events), interrupt != nil, len(reply))

	p.sessions.mu.Lock()
	defer p.sessions.mu.Unlock()
	state = p.sessions.getOrCreateLocked(userIdentity, sessionID)
	state.updatedAt = time.Now()
	if interrupt != nil {
		state.pendingInterrupt = interrupt.InterruptID
	} else {
		state.pendingInterrupt = ""
		if strings.TrimSpace(reply) != "" {
			assistant := schema.AssistantMessage(reply, nil)
			state.messages = append(state.messages, assistant)
			if err := p.appendSessionMessage(userIdentity, sessionID, assistant); err != nil {
				return nil, err
			}
		}
	}
	if err := p.upsertSessionRecord(userIdentity, state); err != nil {
		return nil, err
	}
	logx.Infof("agent stream chat stage=finalize_session session_id=%s user=%s duration=%s", sessionID, userIdentity, time.Since(begin))

	resp := &ChatResponse{
		Session:          toSession(state),
		Reply:            reply,
		Events:           events,
		PendingInterrupt: interrupt,
		ReferencedFiles:  req.Attachments,
	}
	if emit != nil {
		if err := emit(StreamChunk{Type: "done", Session: &resp.Session, Reply: resp.Reply, PendingInterrupt: resp.PendingInterrupt}); err != nil {
			return nil, err
		}
	}
	return resp, nil
}

func (p *provider) Resume(ctx context.Context, userIdentity string, req ResumeRequest) (*ChatResponse, error) {
	if !p.Enabled() {
		return nil, errors.New("agent is disabled")
	}
	if strings.TrimSpace(req.SessionID) == "" || strings.TrimSpace(req.InterruptID) == "" {
		return nil, errors.New("session_id and interrupt_id are required")
	}

	state, err := p.loadSessionState(userIdentity, req.SessionID, false)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, errors.New("session not found")
	}
	if state.pendingInterrupt == "" {
		return nil, errors.New("session has no pending interrupt")
	}
	if state.pendingInterrupt != req.InterruptID {
		return nil, errors.New("interrupt_id mismatch")
	}

	var denyReason *string
	if strings.TrimSpace(req.Reason) != "" {
		reason := strings.TrimSpace(req.Reason)
		denyReason = &reason
	}
	iter, err := p.runner.ResumeWithParams(ctx, checkpointKey(userIdentity, req.SessionID), &adk.ResumeParams{
		Targets: map[string]any{
			req.InterruptID: &approvalResult{
				Approved:         req.Approved,
				DisapproveReason: denyReason,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	reply, events, interrupt, err := collectRunResult(iter)
	if err != nil {
		return nil, err
	}

	p.sessions.mu.Lock()
	defer p.sessions.mu.Unlock()
	state = p.sessions.getOrCreateLocked(userIdentity, req.SessionID)
	state.updatedAt = time.Now()
	if interrupt != nil {
		state.pendingInterrupt = interrupt.InterruptID
	} else {
		state.pendingInterrupt = ""
		if strings.TrimSpace(reply) != "" {
			assistant := schema.AssistantMessage(reply, nil)
			state.messages = append(state.messages, assistant)
			if err := p.appendSessionMessage(userIdentity, req.SessionID, assistant); err != nil {
				return nil, err
			}
		}
	}
	if err := p.upsertSessionRecord(userIdentity, state); err != nil {
		return nil, err
	}

	return &ChatResponse{
		Session:          toSession(state),
		Reply:            reply,
		Events:           events,
		PendingInterrupt: interrupt,
	}, nil
}

func (p *provider) StreamResume(ctx context.Context, userIdentity string, req ResumeRequest, emit func(StreamChunk) error) (*ChatResponse, error) {
	begin := time.Now()
	sessionID := strings.TrimSpace(req.SessionID)
	interruptID := strings.TrimSpace(req.InterruptID)
	logx.Infof("agent stream resume start: session_id=%s interrupt_id=%s user=%s approved=%t", sessionID, interruptID, userIdentity, req.Approved)
	defer func() {
		logx.Infof("agent stream resume finished: session_id=%s interrupt_id=%s user=%s duration=%s", sessionID, interruptID, userIdentity, time.Since(begin))
	}()
	if !p.Enabled() {
		return nil, errors.New("agent is disabled")
	}
	if sessionID == "" || interruptID == "" {
		return nil, errors.New("session_id and interrupt_id are required")
	}

	state, err := p.loadSessionState(userIdentity, sessionID, false)
	if err != nil {
		return nil, err
	}
	logx.Infof("agent stream resume stage=load_session session_id=%s interrupt_id=%s user=%s duration=%s", sessionID, interruptID, userIdentity, time.Since(begin))
	if state == nil {
		return nil, errors.New("session not found")
	}
	if state.pendingInterrupt == "" {
		return nil, errors.New("session has no pending interrupt")
	}
	if state.pendingInterrupt != interruptID {
		return nil, errors.New("interrupt_id mismatch")
	}

	var denyReason *string
	if strings.TrimSpace(req.Reason) != "" {
		reason := strings.TrimSpace(req.Reason)
		denyReason = &reason
	}
	iter, err := p.runner.ResumeWithParams(ctx, checkpointKey(userIdentity, sessionID), &adk.ResumeParams{
		Targets: map[string]any{
			interruptID: &approvalResult{
				Approved:         req.Approved,
				DisapproveReason: denyReason,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	logx.Infof("agent stream resume stage=runner_resumed session_id=%s interrupt_id=%s user=%s duration=%s", sessionID, interruptID, userIdentity, time.Since(begin))

	reply, events, interrupt, err := streamRunResult(iter, emit)
	if err != nil {
		logx.Errorf("agent stream resume stage=runner_failed session_id=%s interrupt_id=%s user=%s duration=%s err=%v", sessionID, interruptID, userIdentity, time.Since(begin), err)
		return nil, err
	}
	logx.Infof("agent stream resume stage=runner_completed session_id=%s interrupt_id=%s user=%s duration=%s events=%d interrupt=%t reply_len=%d", sessionID, interruptID, userIdentity, time.Since(begin), len(events), interrupt != nil, len(reply))

	p.sessions.mu.Lock()
	defer p.sessions.mu.Unlock()
	state = p.sessions.getOrCreateLocked(userIdentity, sessionID)
	state.updatedAt = time.Now()
	if interrupt != nil {
		state.pendingInterrupt = interrupt.InterruptID
	} else {
		state.pendingInterrupt = ""
		if strings.TrimSpace(reply) != "" {
			assistant := schema.AssistantMessage(reply, nil)
			state.messages = append(state.messages, assistant)
			if err := p.appendSessionMessage(userIdentity, sessionID, assistant); err != nil {
				return nil, err
			}
		}
	}
	if err := p.upsertSessionRecord(userIdentity, state); err != nil {
		return nil, err
	}
	logx.Infof("agent stream resume stage=finalize_session session_id=%s interrupt_id=%s user=%s duration=%s", sessionID, interruptID, userIdentity, time.Since(begin))

	resp := &ChatResponse{
		Session:          toSession(state),
		Reply:            reply,
		Events:           events,
		PendingInterrupt: interrupt,
	}
	if emit != nil {
		if err := emit(StreamChunk{Type: "done", Session: &resp.Session, Reply: resp.Reply, PendingInterrupt: resp.PendingInterrupt}); err != nil {
			return nil, err
		}
	}
	return resp, nil
}

func (m *approvalMiddleware) WrapInvokableToolCall(_ context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	if !requiresApproval(tCtx.Name) {
		return endpoint, nil
	}
	return func(ctx context.Context, args string, opts ...einotool.Option) (string, error) {
		wasInterrupted, _, storedArgs := einotool.GetInterruptState[string](ctx)
		if !wasInterrupted {
			return "", einotool.StatefulInterrupt(ctx, &approvalInfo{ToolName: tCtx.Name, ArgumentsInJSON: args}, args)
		}

		isTarget, hasData, data := einotool.GetResumeContext[*approvalResult](ctx)
		if isTarget && hasData {
			if data.Approved {
				return endpoint(ctx, storedArgs, opts...)
			}
			if data.DisapproveReason != nil {
				return fmt.Sprintf("tool '%s' disapproved: %s", tCtx.Name, *data.DisapproveReason), nil
			}
			return fmt.Sprintf("tool '%s' disapproved", tCtx.Name), nil
		}

		isTarget, _, _ = einotool.GetResumeContext[any](ctx)
		if !isTarget {
			return "", einotool.StatefulInterrupt(ctx, &approvalInfo{ToolName: tCtx.Name, ArgumentsInJSON: storedArgs}, storedArgs)
		}

		return endpoint(ctx, storedArgs, opts...)
	}, nil
}

func (m *approvalMiddleware) WrapStreamableToolCall(_ context.Context, endpoint adk.StreamableToolCallEndpoint, tCtx *adk.ToolContext) (adk.StreamableToolCallEndpoint, error) {
	if !requiresApproval(tCtx.Name) {
		return endpoint, nil
	}
	return func(ctx context.Context, args string, opts ...einotool.Option) (*schema.StreamReader[string], error) {
		wasInterrupted, _, storedArgs := einotool.GetInterruptState[string](ctx)
		if !wasInterrupted {
			return nil, einotool.StatefulInterrupt(ctx, &approvalInfo{ToolName: tCtx.Name, ArgumentsInJSON: args}, args)
		}

		isTarget, hasData, data := einotool.GetResumeContext[*approvalResult](ctx)
		if isTarget && hasData {
			if data.Approved {
				return endpoint(ctx, storedArgs, opts...)
			}
			if data.DisapproveReason != nil {
				return singleChunkReader(fmt.Sprintf("tool '%s' disapproved: %s", tCtx.Name, *data.DisapproveReason)), nil
			}
			return singleChunkReader(fmt.Sprintf("tool '%s' disapproved", tCtx.Name)), nil
		}

		isTarget, _, _ = einotool.GetResumeContext[any](ctx)
		if !isTarget {
			return nil, einotool.StatefulInterrupt(ctx, &approvalInfo{ToolName: tCtx.Name, ArgumentsInJSON: storedArgs}, storedArgs)
		}

		return endpoint(ctx, storedArgs, opts...)
	}, nil
}

func (m *safeToolMiddleware) WrapInvokableToolCall(_ context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	return func(ctx context.Context, args string, opts ...einotool.Option) (string, error) {
		start := time.Now()
		toolName := "unknown"
		if tCtx != nil && strings.TrimSpace(tCtx.Name) != "" {
			toolName = tCtx.Name
		}
		logx.Infof("agent tool start: tool=%s mode=invoke args_len=%d", toolName, len(strings.TrimSpace(args)))
		result, err := endpoint(ctx, args, opts...)
		if err != nil {
			logx.Errorf("agent tool failed: tool=%s mode=invoke duration=%s err=%v", toolName, time.Since(start), err)
			if _, ok := compose.IsInterruptRerunError(err); ok {
				return "", err
			}
			return fmt.Sprintf("[tool error] %v", err), nil
		}
		logx.Infof("agent tool completed: tool=%s mode=invoke duration=%s result_len=%d", toolName, time.Since(start), len(strings.TrimSpace(result)))
		return result, nil
	}, nil
}

func (m *safeToolMiddleware) WrapStreamableToolCall(_ context.Context, endpoint adk.StreamableToolCallEndpoint, tCtx *adk.ToolContext) (adk.StreamableToolCallEndpoint, error) {
	return func(ctx context.Context, args string, opts ...einotool.Option) (*schema.StreamReader[string], error) {
		start := time.Now()
		toolName := "unknown"
		if tCtx != nil && strings.TrimSpace(tCtx.Name) != "" {
			toolName = tCtx.Name
		}
		logx.Infof("agent tool start: tool=%s mode=stream args_len=%d", toolName, len(strings.TrimSpace(args)))
		sr, err := endpoint(ctx, args, opts...)
		if err != nil {
			logx.Errorf("agent tool failed: tool=%s mode=stream duration=%s err=%v", toolName, time.Since(start), err)
			if _, ok := compose.IsInterruptRerunError(err); ok {
				return nil, err
			}
			return singleChunkReader(fmt.Sprintf("[tool error] %v", err)), nil
		}
		logx.Infof("agent tool stream ready: tool=%s mode=stream duration=%s", toolName, time.Since(start))
		return safeWrapReader(sr), nil
	}, nil
}

func collectRunResult(iter *adk.AsyncIterator[*adk.AgentEvent]) (string, []ChatEvent, *PendingInterrupt, error) {
	var (
		reply   strings.Builder
		events  []ChatEvent
		pending *PendingInterrupt
	)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", nil, nil, event.Err
		}
		if event.Action != nil && event.Action.Interrupted != nil {
			for _, ic := range event.Action.Interrupted.InterruptContexts {
				if !ic.IsRootCause {
					continue
				}
				if info, ok := ic.Info.(*approvalInfo); ok {
					pending = &PendingInterrupt{InterruptID: ic.ID, ToolName: info.ToolName, ArgumentsJSON: info.ArgumentsInJSON}
					events = append(events, ChatEvent{Type: "approval_required", ToolName: info.ToolName, ArgumentsJSON: info.ArgumentsInJSON})
					break
				}
			}
			continue
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		mv := event.Output.MessageOutput
		switch mv.Role {
		case schema.Tool:
			content := drainMessageVariant(mv)
			events = append(events, ChatEvent{Type: "tool_result", Role: "tool", Content: content})
		case schema.Assistant, "":
			content, toolCalls, err := collectAssistantOutput(mv)
			if err != nil {
				return "", nil, nil, err
			}
			for _, tc := range toolCalls {
				events = append(events, ChatEvent{Type: "tool_call", ToolName: tc.Function.Name, ArgumentsJSON: tc.Function.Arguments})
			}
			if content != "" {
				reply.WriteString(content)
				events = append(events, ChatEvent{Type: "assistant", Role: "assistant", Content: content})
			}
		}
	}
	return strings.TrimSpace(reply.String()), events, pending, nil
}

func streamRunResult(iter *adk.AsyncIterator[*adk.AgentEvent], emit func(StreamChunk) error) (string, []ChatEvent, *PendingInterrupt, error) {
	var (
		reply   strings.Builder
		events  []ChatEvent
		pending *PendingInterrupt
	)
	start := time.Now()
	lastEventAt := start
	eventCount := 0
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		eventCount++
		now := time.Now()
		logx.Infof("agent stream event: index=%d since_start=%s idle=%s has_err=%t has_action=%t has_output=%t", eventCount, now.Sub(start), now.Sub(lastEventAt), event.Err != nil, event.Action != nil, event.Output != nil)
		lastEventAt = now
		if event.Err != nil {
			if emit != nil {
				_ = emit(StreamChunk{Type: "error", Error: event.Err.Error()})
			}
			return "", nil, nil, event.Err
		}
		if event.Action != nil && event.Action.Interrupted != nil {
			for _, ic := range event.Action.Interrupted.InterruptContexts {
				if !ic.IsRootCause {
					continue
				}
				if info, ok := ic.Info.(*approvalInfo); ok {
					logx.Infof("agent stream approval required: interrupt_id=%s tool=%s since_start=%s", ic.ID, info.ToolName, time.Since(start))
					pending = &PendingInterrupt{InterruptID: ic.ID, ToolName: info.ToolName, ArgumentsJSON: info.ArgumentsInJSON}
					events = append(events, ChatEvent{Type: "approval_required", ToolName: info.ToolName, ArgumentsJSON: info.ArgumentsInJSON})
					if emit != nil {
						if err := emit(StreamChunk{Type: "approval_required", ToolName: info.ToolName, ArgumentsJSON: info.ArgumentsInJSON, PendingInterrupt: pending}); err != nil {
							return "", nil, nil, err
						}
					}
					break
				}
			}
			continue
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		mv := event.Output.MessageOutput
		switch mv.Role {
		case schema.Tool:
			toolStageStart := time.Now()
			content := drainMessageVariant(mv)
			logx.Infof("agent stream tool output drained: duration=%s content_len=%d", time.Since(toolStageStart), len(strings.TrimSpace(content)))
			events = append(events, ChatEvent{Type: "tool_result", Role: "tool", Content: content})
			if emit != nil {
				if err := emit(StreamChunk{Type: "tool_result", Role: "tool", Content: content}); err != nil {
					return "", nil, nil, err
				}
			}
		case schema.Assistant, "":
			assistantStageStart := time.Now()
			content, toolCalls, err := streamAssistantOutput(mv, func(chunk string) error {
				reply.WriteString(chunk)
				if emit != nil {
					return emit(StreamChunk{Type: "assistant_delta", Role: "assistant", Content: chunk})
				}
				return nil
			})
			if err != nil {
				return "", nil, nil, err
			}
			logx.Infof("agent stream assistant output drained: duration=%s content_len=%d tool_calls=%d", time.Since(assistantStageStart), len(strings.TrimSpace(content)), len(toolCalls))
			for _, tc := range toolCalls {
				events = append(events, ChatEvent{Type: "tool_call", ToolName: tc.Function.Name, ArgumentsJSON: tc.Function.Arguments})
				if emit != nil {
					if err := emit(StreamChunk{Type: "tool_call", ToolName: tc.Function.Name, ArgumentsJSON: tc.Function.Arguments}); err != nil {
						return "", nil, nil, err
					}
				}
			}
			if content != "" {
				events = append(events, ChatEvent{Type: "assistant", Role: "assistant", Content: content})
			}
		}
	}
	return strings.TrimSpace(reply.String()), events, pending, nil
}

func collectAssistantOutput(mv *adk.MessageVariant) (string, []schema.ToolCall, error) {
	if mv.IsStreaming && mv.MessageStream != nil {
		mv.MessageStream.SetAutomaticClose()
		var content strings.Builder
		var toolCalls []schema.ToolCall
		for {
			frame, err := mv.MessageStream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return "", nil, err
			}
			if frame == nil {
				continue
			}
			if frame.Content != "" {
				content.WriteString(frame.Content)
			}
			if len(frame.ToolCalls) > 0 {
				toolCalls = append(toolCalls, frame.ToolCalls...)
			}
		}
		return content.String(), mergeToolCalls(toolCalls), nil
	}
	if mv.Message == nil {
		return "", nil, nil
	}
	return mv.Message.Content, mergeToolCalls(mv.Message.ToolCalls), nil
}

func streamAssistantOutput(mv *adk.MessageVariant, onText func(string) error) (string, []schema.ToolCall, error) {
	if mv.IsStreaming && mv.MessageStream != nil {
		mv.MessageStream.SetAutomaticClose()
		var content strings.Builder
		var toolCalls []schema.ToolCall
		for {
			frame, err := mv.MessageStream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return "", nil, err
			}
			if frame == nil {
				continue
			}
			if frame.Content != "" {
				content.WriteString(frame.Content)
				if onText != nil {
					if err := onText(frame.Content); err != nil {
						return "", nil, err
					}
				}
			}
			if len(frame.ToolCalls) > 0 {
				toolCalls = append(toolCalls, frame.ToolCalls...)
			}
		}
		return content.String(), mergeToolCalls(toolCalls), nil
	}
	if mv.Message == nil {
		return "", nil, nil
	}
	if mv.Message.Content != "" && onText != nil {
		if err := onText(mv.Message.Content); err != nil {
			return "", nil, err
		}
	}
	return mv.Message.Content, mergeToolCalls(mv.Message.ToolCalls), nil
}

func mergeToolCalls(chunks []schema.ToolCall) []schema.ToolCall {
	if len(chunks) <= 1 {
		return chunks
	}
	order := make([]string, 0, len(chunks))
	merged := make(map[string]schema.ToolCall, len(chunks))
	for _, chunk := range chunks {
		key := toolCallKey(chunk)
		if _, ok := merged[key]; !ok {
			order = append(order, key)
			merged[key] = chunk
			continue
		}
		current := merged[key]
		if current.ID == "" {
			current.ID = chunk.ID
		}
		if current.Type == "" {
			current.Type = chunk.Type
		}
		if current.Function.Name == "" {
			current.Function.Name = chunk.Function.Name
		}
		current.Function.Arguments += chunk.Function.Arguments
		merged[key] = current
	}
	out := make([]schema.ToolCall, 0, len(order))
	for _, key := range order {
		out = append(out, merged[key])
	}
	return out
}

func toolCallKey(tc schema.ToolCall) string {
	if tc.ID != "" {
		return "id:" + tc.ID
	}
	if tc.Index != nil {
		return fmt.Sprintf("index:%d", *tc.Index)
	}
	return "name:" + tc.Function.Name
}

func drainMessageVariant(mv *adk.MessageVariant) string {
	if mv == nil {
		return ""
	}
	if mv.IsStreaming && mv.MessageStream != nil {
		mv.MessageStream.SetAutomaticClose()
		var sb strings.Builder
		for {
			frame, err := mv.MessageStream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				break
			}
			if frame != nil && frame.Content != "" {
				sb.WriteString(frame.Content)
			}
		}
		return sb.String()
	}
	if mv.Message != nil {
		return mv.Message.Content
	}
	return ""
}

func buildInstruction(projectRoot string) string {
	lines := []string{
		"You are the cloud_disk assistant.",
		"When the user asks about attached text documents, call " + tools.FileAnswerToolName + ".",
		"When the user asks about attached images, call " + tools.ImageAnalysisToolName + ".",
		"When the user asks about attached videos, call " + tools.VideoSummaryToolName + ".",
		"When the user asks to browse or inspect the file list, call " + tools.ListFilesToolName + ".",
		"When the user asks to create a folder, call " + tools.CreateFolderToolName + ".",
		"When the user asks to move a file or folder, call " + tools.MoveFileToolName + ".",
		"Do not guess file contents or execute file operations without using the appropriate tool.",
		"Only answer from the user's accessible files and the visible conversation context.",
	}
	if strings.TrimSpace(projectRoot) != "" {
		lines = append(lines, "Project root: "+projectRoot)
	}
	return strings.Join(lines, "\n")
}

func buildUserPrompt(message string, attachments []Attachment) string {
	if len(attachments) == 0 {
		return message
	}
	var sb strings.Builder
	sb.WriteString(message)
	sb.WriteString("\n\nAttached cloud_disk files:\n")
	allFiles := make([]tools.FileRef, 0, len(attachments))
	imageFiles := make([]tools.FileRef, 0)
	videoFiles := make([]tools.FileRef, 0)
	textFiles := make([]tools.FileRef, 0)
	for _, item := range attachments {
		sb.WriteString("- ")
		sb.WriteString(item.Name)
		if item.URL != "" {
			sb.WriteString(" (")
			sb.WriteString(item.URL)
			sb.WriteString(")")
		}
		if item.MIMEType != "" {
			sb.WriteString(" [")
			sb.WriteString(item.MIMEType)
			sb.WriteString("]")
		}
		sb.WriteString("\n")
		ref := tools.FileRef{Name: item.Name, URL: item.URL, MIMEType: item.MIMEType}
		allFiles = append(allFiles, ref)
		if tools.IsImageLike(item.MIMEType, item.Name) {
			imageFiles = append(imageFiles, ref)
		}
		if tools.IsVideoLike(item.MIMEType, item.Name) {
			videoFiles = append(videoFiles, ref)
		}
		if tools.IsTextLike(item.MIMEType, item.Name) {
			textFiles = append(textFiles, ref)
		}
	}
	writeToolArrayHint(&sb, "text_files", textFiles)
	writeToolArrayHint(&sb, "image_files", imageFiles)
	writeToolArrayHint(&sb, "all_files", allFiles)
	if len(videoFiles) > 0 {
		raw, _ := json.Marshal(videoFiles[0])
		sb.WriteString("\nIf the request is about a video, call ")
		sb.WriteString(tools.VideoSummaryToolName)
		sb.WriteString(" with the user's question and this exact file JSON:\nvideo_file=")
		sb.Write(raw)
		sb.WriteString("\n")
	}
	return sb.String()
}

func writeToolArrayHint(sb *strings.Builder, label string, files []tools.FileRef) {
	raw, _ := json.Marshal(files)
	sb.WriteString("\n")
	sb.WriteString(label)
	sb.WriteString("=")
	sb.Write(raw)
	sb.WriteString("\n")
}

func summarizeTitle(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return "New Chat"
	}
	runes := []rune(message)
	if len(runes) > 24 {
		return string(runes[:24])
	}
	return message
}

func checkpointKey(userIdentity, sessionID string) string {
	return "agent:" + userIdentity + ":" + sessionID
}

func toSession(state *sessionState) Session {
	if state == nil {
		return Session{}
	}
	return Session{
		ID:                 state.id,
		Title:              state.title,
		PendingInterruptID: state.pendingInterrupt,
		CreatedAt:          state.createdAt,
		UpdatedAt:          state.updatedAt,
	}
}

func sessionFromRow(row models.AgentChatSession) Session {
	return Session{
		ID:                 row.SessionID,
		Title:              row.Title,
		PendingInterruptID: row.PendingInterruptID,
		CreatedAt:          parseModelTime(row.CreatedAt),
		UpdatedAt:          parseModelTime(row.UpdatedAt),
	}
}

func (s *sessionStore) getOrCreate(userIdentity, sessionID string) *sessionState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getOrCreateLocked(userIdentity, sessionID)
}

func (s *sessionStore) getOrCreateLocked(userIdentity, sessionID string) *sessionState {
	key := userIdentity + ":" + sessionID
	if state, ok := s.sessions[key]; ok {
		return state
	}
	now := time.Now()
	state := &sessionState{id: sessionID, userIdentity: userIdentity, title: "New Chat", createdAt: now, updatedAt: now}
	s.sessions[key] = state
	return state
}

func (s *sessionStore) get(userIdentity, sessionID string) *sessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[userIdentity+":"+sessionID]
}

func (s *sessionStore) list(userIdentity string) []Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Session, 0)
	prefix := userIdentity + ":"
	for key, state := range s.sessions {
		if strings.HasPrefix(key, prefix) {
			items = append(items, toSession(state))
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	return items
}

func (s *checkpointStore) Set(_ context.Context, key string, value []byte) error {
	if s.db == nil {
		return nil
	}
	userIdentity, sessionID := parseCheckpointKey(key)
	row := &models.AgentChatCheckpoint{
		CheckpointKey: key,
		SessionID:     sessionID,
		UserIdentity:  userIdentity,
		Payload:       append([]byte(nil), value...),
	}
	exists := new(models.AgentChatCheckpoint)
	has, err := s.db.Where("checkpoint_key = ?", key).Get(exists)
	if err != nil {
		return err
	}
	if has {
		_, err = s.db.Where("checkpoint_key = ?", key).Cols("payload", "session_id", "user_identity").Update(row)
		return err
	}
	_, err = s.db.Insert(row)
	return err
}

func (s *checkpointStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	if s.db == nil {
		return nil, false, nil
	}
	row := new(models.AgentChatCheckpoint)
	has, err := s.db.Where("checkpoint_key = ?", key).Get(row)
	if err != nil {
		return nil, false, err
	}
	if !has {
		return nil, false, nil
	}
	buf := make([]byte, len(row.Payload))
	copy(buf, row.Payload)
	return buf, true, nil
}

func (p *provider) loadSessionState(userIdentity, sessionID string, createIfMissing bool) (*sessionState, error) {
	if state := p.sessions.get(userIdentity, sessionID); state != nil {
		return state, nil
	}
	if p.deps.DBEngine == nil {
		if !createIfMissing {
			return nil, nil
		}
		return p.sessions.getOrCreate(userIdentity, sessionID), nil
	}

	row := new(models.AgentChatSession)
	has, err := p.deps.DBEngine.Where("session_id = ? AND user_identity = ?", sessionID, userIdentity).Get(row)
	if err != nil {
		return nil, err
	}
	if !has {
		if !createIfMissing {
			return nil, nil
		}
		state := p.sessions.getOrCreate(userIdentity, sessionID)
		if err := p.upsertSessionRecord(userIdentity, state); err != nil {
			return nil, err
		}
		return state, nil
	}

	messageRows := make([]models.AgentChatMessage, 0)
	if err := p.deps.DBEngine.Where("session_id = ? AND user_identity = ?", sessionID, userIdentity).Asc("id").Find(&messageRows); err != nil {
		return nil, err
	}
	messages := make([]*schema.Message, 0, len(messageRows))
	for _, row := range messageRows {
		msg := new(schema.Message)
		if err := json.Unmarshal([]byte(row.MessageJSON), msg); err != nil {
			continue
		}
		messages = append(messages, msg)
	}
	state := &sessionState{
		id:               row.SessionID,
		userIdentity:     row.UserIdentity,
		title:            row.Title,
		pendingInterrupt: row.PendingInterruptID,
		createdAt:        parseModelTime(row.CreatedAt),
		updatedAt:        parseModelTime(row.UpdatedAt),
		messages:         messages,
	}
	p.sessions.mu.Lock()
	p.sessions.sessions[userIdentity+":"+sessionID] = state
	p.sessions.mu.Unlock()
	return state, nil
}

func (p *provider) upsertSessionRecord(userIdentity string, state *sessionState) error {
	if p.deps.DBEngine == nil || state == nil {
		return nil
	}
	row := &models.AgentChatSession{
		SessionID:          state.id,
		UserIdentity:       userIdentity,
		Title:              state.title,
		PendingInterruptID: state.pendingInterrupt,
	}
	exists := new(models.AgentChatSession)
	has, err := p.deps.DBEngine.Where("session_id = ? AND user_identity = ?", state.id, userIdentity).Get(exists)
	if err != nil {
		return err
	}
	if has {
		_, err = p.deps.DBEngine.Where("session_id = ? AND user_identity = ?", state.id, userIdentity).Cols("title", "pending_interrupt_id").Update(row)
		return err
	}
	_, err = p.deps.DBEngine.Insert(row)
	return err
}

func (p *provider) appendSessionMessage(userIdentity, sessionID string, msg *schema.Message) error {
	if p.deps.DBEngine == nil || msg == nil {
		return nil
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = p.deps.DBEngine.Insert(&models.AgentChatMessage{
		SessionID:    sessionID,
		UserIdentity: userIdentity,
		Role:         string(msg.Role),
		Content:      msg.Content,
		MessageJSON:  string(raw),
	})
	return err
}

func cloneMessages(messages []*schema.Message) []*schema.Message {
	out := make([]*schema.Message, 0, len(messages))
	for _, message := range messages {
		if message != nil {
			cp := *message
			out = append(out, &cp)
		}
	}
	return out
}

func requiresApproval(_ string) bool {
	return true
}

func cleanAbsPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

func getenv(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func singleChunkReader(msg string) *schema.StreamReader[string] {
	r, w := schema.Pipe[string](1)
	_ = w.Send(msg, nil)
	w.Close()
	return r
}

func safeWrapReader(sr *schema.StreamReader[string]) *schema.StreamReader[string] {
	r, w := schema.Pipe[string](64)
	go func() {
		defer w.Close()
		for {
			chunk, err := sr.Recv()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				_ = w.Send(fmt.Sprintf("\n[tool error] %v", err), nil)
				return
			}
			_ = w.Send(chunk, nil)
		}
	}()
	return r
}

func parseCheckpointKey(key string) (userIdentity, sessionID string) {
	parts := strings.SplitN(key, ":", 3)
	if len(parts) == 3 {
		return parts[1], parts[2]
	}
	return "", ""
}

func parseModelTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	layouts := []string{
		common.DataTimeFormat,
		"2006-01-02 15:04:05",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if ts, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return ts
		}
	}
	return time.Now()
}

func (p *provider) createFolder(ctx context.Context, parentFolderIdentity, name string) (tools.FolderCreateResult, error) {
	if p.deps.DBEngine == nil {
		return tools.FolderCreateResult{}, errors.New("database is not configured")
	}
	userIdentity, ok := ctx.Value("user_identity").(string)
	if !ok || strings.TrimSpace(userIdentity) == "" {
		return tools.FolderCreateResult{}, errors.New("用户身份验证失败")
	}
	parentID, err := p.resolveFolderID(ctx, userIdentity, parentFolderIdentity)
	if err != nil {
		return tools.FolderCreateResult{}, err
	}
	cnt, err := p.deps.DBEngine.Table("user_repository").
		Where("name = ? AND parent_id = ? AND user_identity = ? AND (status != ? OR status IS NULL)", name, parentID, userIdentity, common.StatusDeleted).
		Count(new(models.UserRepository))
	if err != nil {
		return tools.FolderCreateResult{}, err
	}
	if cnt > 0 {
		return tools.FolderCreateResult{}, errors.New("该目录下已存在同名文件")
	}
	data := &models.UserRepository{
		Identity:     utils.UUID(),
		UserIdentity: userIdentity,
		ParentId:     parentID,
		Name:         name,
		Status:       common.StatusActive,
	}
	if _, err := p.deps.DBEngine.Table("user_repository").Insert(data); err != nil {
		return tools.FolderCreateResult{}, err
	}
	return tools.FolderCreateResult{ID: data.Id, Identity: data.Identity, Name: data.Name, ParentID: data.ParentId}, nil
}

func (p *provider) moveFile(ctx context.Context, fileIdentity, targetFolderIdentity, desiredName string) error {
	if p.deps.DBEngine == nil {
		return errors.New("database is not configured")
	}
	userIdentity, ok := ctx.Value("user_identity").(string)
	if !ok || strings.TrimSpace(userIdentity) == "" {
		return errors.New("用户身份验证失败")
	}
	item := new(models.UserRepository)
	has, err := p.deps.DBEngine.Where("identity = ? AND user_identity = ? AND (status != ? OR status IS NULL)", fileIdentity, userIdentity, common.StatusDeleted).Get(item)
	if err != nil {
		return err
	}
	if !has {
		return errors.New("待移动文件不存在")
	}
	parentID, err := p.resolveFolderID(ctx, userIdentity, targetFolderIdentity)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(desiredName)
	if name == "" {
		name = item.Name
	}
	if item.Identity == strings.TrimSpace(targetFolderIdentity) {
		return errors.New("不能将文件移动到自身")
	}
	cnt, err := p.deps.DBEngine.Table("user_repository").
		Where("name = ? AND parent_id = ? AND user_identity = ? AND identity <> ? AND (status != ? OR status IS NULL)", name, parentID, userIdentity, item.Identity, common.StatusDeleted).
		Count(new(models.UserRepository))
	if err != nil {
		return err
	}
	if cnt > 0 {
		return errors.New("该目录下已存在同名文件")
	}
	_, err = p.deps.DBEngine.Table("user_repository").
		Where("identity = ? AND user_identity = ? AND (status != ? OR status IS NULL)", item.Identity, userIdentity, common.StatusDeleted).
		Update(&models.UserRepository{ParentId: parentID, Name: name})
	return err
}

func (p *provider) resolveFolderID(_ context.Context, userIdentity, folderIdentity string) (int64, error) {
	folderIdentity = strings.TrimSpace(folderIdentity)
	if folderIdentity == "" {
		return 0, nil
	}
	parent := new(models.UserRepository)
	has, err := p.deps.DBEngine.Where("identity = ? AND user_identity = ? AND repository_identity = '' AND (status != ? OR status IS NULL)", folderIdentity, userIdentity, common.StatusDeleted).Get(parent)
	if err != nil {
		return 0, err
	}
	if !has {
		return 0, errors.New("目标文件夹不存在")
	}
	return parent.Id, nil
}

func (p *provider) listFiles(ctx context.Context, folderIdentity string, page, size int) ([]tools.ListedFile, int64, error) {
	if p.deps.DBEngine == nil {
		return nil, 0, errors.New("database is not configured")
	}
	userIdentity, ok := ctx.Value("user_identity").(string)
	if !ok || strings.TrimSpace(userIdentity) == "" {
		return nil, 0, errors.New("用户身份验证失败")
	}
	parentID, err := p.resolveFolderID(ctx, userIdentity, folderIdentity)
	if err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 50
	}
	if size > 200 {
		size = 200
	}
	offset := (page - 1) * size

	type row struct {
		ID                 int64  `xorm:"'id'"`
		Identity           string `xorm:"'identity'"`
		Name               string `xorm:"'name'"`
		Ext                string `xorm:"'ext'"`
		Size               int64  `xorm:"'size'"`
		RepositoryIdentity string `xorm:"'repository_identity'"`
		UpdatedAt          string `xorm:"'updated_at'"`
	}
	rows := make([]row, 0)
	err = p.deps.DBEngine.Table("user_repository").
		Where("parent_id = ? AND user_identity = ?", parentID, userIdentity).
		Select("user_repository.id as id, user_repository.identity as identity, user_repository.name as name, "+
			"user_repository.repository_identity as repository_identity, user_repository.ext as ext, "+
			"repository_pool.size as size, user_repository.updated_at as updated_at").
		Join("LEFT", "repository_pool", "user_repository.repository_identity <> '' AND user_repository.repository_identity = repository_pool.identity").
		Where("user_repository.status != ? OR user_repository.status IS NULL", common.StatusDeleted).
		Limit(size, offset).
		Find(&rows)
	if err != nil {
		return nil, 0, err
	}

	count, err := p.deps.DBEngine.Table("user_repository").
		Where("parent_id = ? AND user_identity = ?", parentID, userIdentity).
		Where("status != ? OR status IS NULL", common.StatusDeleted).
		Count(new(models.UserRepository))
	if err != nil {
		return nil, 0, err
	}

	list := make([]tools.ListedFile, 0, len(rows))
	for _, item := range rows {
		list = append(list, tools.ListedFile{
			ID:                 item.ID,
			Identity:           item.Identity,
			Name:               item.Name,
			Ext:                item.Ext,
			Size:               item.Size,
			RepositoryIdentity: item.RepositoryIdentity,
			UpdatedAt:          item.UpdatedAt,
			IsFolder:           strings.TrimSpace(item.RepositoryIdentity) == "",
		})
	}
	return list, count, nil
}
