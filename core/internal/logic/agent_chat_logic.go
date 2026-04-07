package logic

import (
	"context"
	"errors"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud_disk/core/common"
	"cloud_disk/core/internal/agent/einoagent"
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"cloud_disk/core/models"
	"cloud_disk/core/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AgentChatLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAgentChatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AgentChatLogic {
	return &AgentChatLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AgentChatLogic) CreateSession() (*types.AgentChatResponse, error) {
	provider, userIdentity, err := l.requireProviderAndUser()
	if err != nil {
		return nil, err
	}
	session, err := provider.CreateSession(l.ctx, userIdentity)
	if err != nil {
		return nil, err
	}
	return &types.AgentChatResponse{
		Session: convertSession(session),
	}, nil
}

func (l *AgentChatLogic) ListSessions() (*types.AgentSessionListResponse, error) {
	provider, userIdentity, err := l.requireProviderAndUser()
	if err != nil {
		return nil, err
	}
	sessions, err := provider.ListSessions(l.ctx, userIdentity)
	if err != nil {
		return nil, err
	}
	list := make([]*types.AgentSession, 0, len(sessions))
	for _, item := range sessions {
		list = append(list, convertSession(item))
	}
	return &types.AgentSessionListResponse{List: list}, nil
}

func (l *AgentChatLogic) GetSessionDetail(sessionID string) (*types.AgentSessionDetailResponse, error) {
	provider, userIdentity, err := l.requireProviderAndUser()
	if err != nil {
		return nil, err
	}
	detail, err := provider.GetSessionDetail(l.ctx, userIdentity, strings.TrimSpace(sessionID))
	if err != nil {
		return nil, err
	}
	resp := &types.AgentSessionDetailResponse{
		Session:  convertSession(detail.Session),
		Messages: make([]*types.AgentConversationMessage, 0, len(detail.Messages)),
	}
	for _, item := range detail.Messages {
		resp.Messages = append(resp.Messages, &types.AgentConversationMessage{
			Role:      item.Role,
			Content:   item.Content,
			CreatedAt: item.CreatedAt.Format(common.DataTimeFormat),
		})
	}
	return resp, nil
}

func (l *AgentChatLogic) Chat(req *types.AgentChatRequest) (*types.AgentChatResponse, error) {
	provider, userIdentity, err := l.requireProviderAndUser()
	if err != nil {
		return nil, err
	}
	attachments, refs, err := l.resolveAttachments(req.Attachments)
	if err != nil {
		return nil, err
	}
	resp, err := provider.Chat(l.ctx, userIdentity, einoagent.ChatRequest{
		SessionID:   strings.TrimSpace(req.SessionID),
		UserName:    userNameFromCtx(l.ctx),
		Message:     strings.TrimSpace(req.Message),
		Attachments: attachments,
	})
	if err != nil {
		return nil, err
	}
	out := convertChatResponse(resp)
	out.ReferencedFiles = refs
	return out, nil
}

func (l *AgentChatLogic) StreamChat(req *types.AgentChatRequest, emit func(einoagent.StreamChunk) error) (*types.AgentChatResponse, error) {
	provider, userIdentity, err := l.requireProviderAndUser()
	if err != nil {
		return nil, err
	}
	attachments, refs, err := l.resolveAttachments(req.Attachments)
	if err != nil {
		return nil, err
	}
	resp, err := provider.StreamChat(l.ctx, userIdentity, einoagent.ChatRequest{
		SessionID:   strings.TrimSpace(req.SessionID),
		UserName:    userNameFromCtx(l.ctx),
		Message:     strings.TrimSpace(req.Message),
		Attachments: attachments,
	}, func(chunk einoagent.StreamChunk) error {
		if chunk.Type == "referenced_files" {
			chunk.ReferencedFiles = attachments
		}
		if emit != nil {
			return emit(chunk)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := convertChatResponse(resp)
	out.ReferencedFiles = refs
	return out, nil
}

func (l *AgentChatLogic) Resume(req *types.AgentResumeRequest) (*types.AgentChatResponse, error) {
	provider, userIdentity, err := l.requireProviderAndUser()
	if err != nil {
		return nil, err
	}
	resp, err := provider.Resume(l.ctx, userIdentity, einoagent.ResumeRequest{
		SessionID:   strings.TrimSpace(req.SessionID),
		UserName:    userNameFromCtx(l.ctx),
		InterruptID: strings.TrimSpace(req.InterruptID),
		Approved:    req.Approved,
		Reason:      strings.TrimSpace(req.Reason),
	})
	if err != nil {
		return nil, err
	}
	return convertChatResponse(resp), nil
}

func (l *AgentChatLogic) StreamResume(req *types.AgentResumeRequest, emit func(einoagent.StreamChunk) error) (*types.AgentChatResponse, error) {
	provider, userIdentity, err := l.requireProviderAndUser()
	if err != nil {
		return nil, err
	}
	resp, err := provider.StreamResume(l.ctx, userIdentity, einoagent.ResumeRequest{
		SessionID:   strings.TrimSpace(req.SessionID),
		UserName:    userNameFromCtx(l.ctx),
		InterruptID: strings.TrimSpace(req.InterruptID),
		Approved:    req.Approved,
		Reason:      strings.TrimSpace(req.Reason),
	}, emit)
	if err != nil {
		return nil, err
	}
	return convertChatResponse(resp), nil
}

func (l *AgentChatLogic) requireProviderAndUser() (einoagent.Provider, string, error) {
	if l.svcCtx.AgentProvider == nil || !l.svcCtx.AgentProvider.Enabled() {
		return nil, "", errors.New("agent is not enabled")
	}
	userIdentity, ok := l.ctx.Value("user_identity").(string)
	if !ok || strings.TrimSpace(userIdentity) == "" {
		return nil, "", errors.New("用户身份验证失败")
	}
	return l.svcCtx.AgentProvider, userIdentity, nil
}

func (l *AgentChatLogic) resolveAttachments(items []types.AgentFileReference) ([]einoagent.Attachment, []*types.AgentFileReference, error) {
	if len(items) == 0 {
		return nil, nil, nil
	}
	userIdentity, _ := l.ctx.Value("user_identity").(string)
	attachments := make([]einoagent.Attachment, 0, len(items))
	refs := make([]*types.AgentFileReference, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.URL) != "" {
			mimeType := strings.TrimSpace(item.MIMEType)
			if mimeType == "" {
				mimeType = mime.TypeByExtension(strings.ToLower(filepathExt(item.Name)))
			}
			attachments = append(attachments, einoagent.Attachment{
				Name:     strings.TrimSpace(item.Name),
				URL:      strings.TrimSpace(item.URL),
				MIMEType: mimeType,
			})
			ref := item
			ref.MIMEType = mimeType
			refs = append(refs, &ref)
			continue
		}

		fileIdentity := strings.TrimSpace(item.FileIdentity)
		if fileIdentity == "" {
			return nil, nil, errors.New("attachment.file_identity or attachment.url is required")
		}
		ref, repo, err := l.lookupUserFile(userIdentity, fileIdentity)
		if err != nil {
			return nil, nil, err
		}
		url, err := l.presignRepository(repo.Identity, repo.ObjectKey, repo.Path, 600)
		if err != nil {
			return nil, nil, err
		}
		name := strings.TrimSpace(ref.Name)
		if name == "" {
			name = repo.Name
		}
		mimeType := item.MIMEType
		if mimeType == "" {
			mimeType = mime.TypeByExtension(strings.ToLower(filepathExt(name)))
		}
		attachments = append(attachments, einoagent.Attachment{
			Name:     name,
			URL:      url,
			MIMEType: mimeType,
		})
		refs = append(refs, &types.AgentFileReference{
			FileIdentity: fileIdentity,
			Name:         name,
			URL:          url,
			MIMEType:     mimeType,
		})
	}
	return attachments, refs, nil
}

func (l *AgentChatLogic) lookupUserFile(userIdentity, fileIdentity string) (*models.UserRepository, *models.RepositoryPool, error) {
	ref := new(models.UserRepository)
	has, err := l.svcCtx.DBEngine.
		Where("identity = ? AND user_identity = ? AND (status != ? OR status IS NULL)", fileIdentity, userIdentity, common.StatusDeleted).
		Get(ref)
	if err != nil {
		return nil, nil, err
	}
	if !has {
		return nil, nil, errors.New("文件不存在或无权限访问")
	}
	if strings.TrimSpace(ref.RepositoryIdentity) == "" {
		return nil, nil, errors.New("当前仅支持引用文件，暂不支持文件夹")
	}
	repo := new(models.RepositoryPool)
	has, err = l.svcCtx.DBEngine.Where("identity = ?", ref.RepositoryIdentity).Get(repo)
	if err != nil {
		return nil, nil, err
	}
	if !has {
		return nil, nil, errors.New("文件存储记录不存在")
	}
	return ref, repo, nil
}

func (l *AgentChatLogic) presignRepository(repositoryIdentity, objectKey, path string, expires int) (string, error) {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		objectKey = utils.ObjectKeyFromPath(path)
	}
	if objectKey == "" {
		return "", errors.New("文件未绑定对象键")
	}

	storageType := strings.ToLower(strings.TrimSpace(os.Getenv("STORAGE_TYPE")))
	if storageType == "tos" {
		storage, err := utils.GetStorage()
		if err != nil {
			return "", err
		}
		return storage.GetPresignedURL(l.ctx, objectKey, time.Duration(expires)*time.Second)
	}
	return utils.PresignGetObject(l.ctx, objectKey, time.Duration(expires)*time.Second)
}

func convertChatResponse(resp *einoagent.ChatResponse) *types.AgentChatResponse {
	if resp == nil {
		return &types.AgentChatResponse{}
	}
	out := &types.AgentChatResponse{
		Reply: resp.Reply,
	}
	out.Session = convertSession(resp.Session)
	if resp.PendingInterrupt != nil {
		out.PendingInterrupt = &types.AgentPendingInterrupt{
			InterruptID:   resp.PendingInterrupt.InterruptID,
			ToolName:      resp.PendingInterrupt.ToolName,
			ArgumentsJSON: resp.PendingInterrupt.ArgumentsJSON,
		}
	}
	if len(resp.Events) > 0 {
		out.Events = make([]*types.AgentChatEvent, 0, len(resp.Events))
		for _, item := range resp.Events {
			ev := item
			out.Events = append(out.Events, &types.AgentChatEvent{
				Type:          ev.Type,
				Role:          ev.Role,
				Content:       ev.Content,
				ToolName:      ev.ToolName,
				ArgumentsJSON: ev.ArgumentsJSON,
			})
		}
	}
	if len(resp.ReferencedFiles) > 0 {
		out.ReferencedFiles = make([]*types.AgentFileReference, 0, len(resp.ReferencedFiles))
		for _, item := range resp.ReferencedFiles {
			ref := item
			out.ReferencedFiles = append(out.ReferencedFiles, &types.AgentFileReference{
				Name:     ref.Name,
				URL:      ref.URL,
				MIMEType: ref.MIMEType,
			})
		}
	}
	return out
}

func convertSession(session einoagent.Session) *types.AgentSession {
	return &types.AgentSession{
		ID:                 session.ID,
		Title:              session.Title,
		PendingInterruptID: session.PendingInterruptID,
		CreatedAt:          session.CreatedAt.Format(common.DataTimeFormat),
		UpdatedAt:          session.UpdatedAt.Format(common.DataTimeFormat),
	}
}

func userNameFromCtx(ctx context.Context) string {
	userName, _ := ctx.Value("user_name").(string)
	return strings.TrimSpace(userName)
}

func filepathExt(name string) string {
	return strings.ToLower(strings.TrimSpace(filepath.Ext(strings.TrimSpace(name))))
}
