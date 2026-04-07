package logic

import (
	"cloud_disk/core/internal/cache"
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
)

// QueryUploadTaskStatusLogic 查询上传任务状态逻辑。
type QueryUploadTaskStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewQueryUploadTaskStatusLogic 创建查询上传任务状态逻辑。
func NewQueryUploadTaskStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryUploadTaskStatusLogic {
	return &QueryUploadTaskStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// QueryUploadTaskStatus 查询上传任务状态。
func (l *QueryUploadTaskStatusLogic) QueryUploadTaskStatus(req *types.QueryUploadTaskStatusRequest) (*types.QueryUploadTaskStatusResponse, error) {
	userIdentity, ok := l.ctx.Value("user_identity").(string)
	if !ok {
		return nil, errors.New("用户身份验证失败")
	}

	metaList, err := cache.ListUploadTaskMeta(l.ctx, l.svcCtx.RedisClient, userIdentity)
	if err != nil {
		return nil, err
	}

	resp := &types.QueryUploadTaskStatusResponse{List: make([]*types.UploadTaskStatusItem, 0, len(metaList))}
	for _, meta := range metaList {
		if req.Hash != "" && req.Hash != meta.Hash {
			continue
		}
		state, _ := cache.GetUploadTaskState(l.ctx, l.svcCtx.RedisClient, userIdentity, meta.Hash)
		parts, partErr := cache.GetUploadedPartIndexes(l.ctx, l.svcCtx.RedisClient, meta.Hash)
		if partErr != nil {
			parts = []int{}
		}
		resp.List = append(resp.List, &types.UploadTaskStatusItem{
			Hash:          meta.Hash,
			TaskKey:       meta.TaskKey,
			Name:          meta.Name,
			Ext:           meta.Ext,
			Size:          meta.Size,
			State:         state,
			UploadedParts: parts,
			UpdatedAt:     meta.UpdatedAt,
		})
	}

	return resp, nil
}
