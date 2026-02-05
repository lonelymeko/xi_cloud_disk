// goctl 生成代码，可安全编辑。
// goctl 1.9.2

package logic

import (
	"context"

	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

// GetShareRecordLogic 获取分享记录逻辑。
type GetShareRecordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetShareRecordLogic 创建获取分享记录逻辑。
func NewGetShareRecordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetShareRecordLogic {
	return &GetShareRecordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetShareRecord 获取分享记录。
func (l *GetShareRecordLogic) GetShareRecord(req *types.GetShareRecordRequest) (resp *types.GetShareRecordResponse, err error) {
	resp = &types.GetShareRecordResponse{}
	_, err = l.svcCtx.DBEngine.Table("share_basic").
		Select("share_basic.identity, user_repository.name, repository_pool.ext,repository_pool.size,repository_pool.path").
		Where("share_basic.identity = ?", req.Identity).
		Join("LEFT", "repository_pool", "share_basic.repository_identity = repository_pool.identity").
		Join("LEFT", "user_repository", "repository_pool.identity = user_repository.repository_identity").
		Get(resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
