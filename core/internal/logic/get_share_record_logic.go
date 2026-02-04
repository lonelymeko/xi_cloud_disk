// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetShareRecordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetShareRecordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetShareRecordLogic {
	return &GetShareRecordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetShareRecordLogic) GetShareRecord(req *types.GetShareRecordRequest) (resp *types.GetShareRecordResponse, err error) {
	_, err = l.svcCtx.DBEngine.Table("share_basic").
		Select("share_basic.identity, user_repository.name, repository_pool.ext,repository_pool.size,repository_pool.path").
		Where("identity = ?", req.Identity).
		Join("LEFT", "repository_pool", "share_basic.repository_identity = repository_pool.identity").
		Join("LEFT", "user_repository", "repository_pool.identity = user_repository.repository_identity").
		Get(&resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
