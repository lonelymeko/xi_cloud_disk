// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"cloud_disk/core/models"
	"cloud_disk/core/utils"
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateShareRecordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateShareRecordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateShareRecordLogic {
	return &CreateShareRecordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateShareRecordLogic) CreateShareRecord(req *types.CreateShareRecordRequest) (resp *types.CreateShareRecordResponse, err error) {
	userIdentity, ok := l.ctx.Value("user_identity").(string)
	if !ok {
		return nil, errors.New("用户身份验证失败")
	}
	data := new(models.ShareBasic)
	data.UserIdentity = userIdentity
	data.RepositoryIdentity = req.Identity
	data.ExpiredTime = req.ExpiredTime
	data.Identity = utils.UUID()
	_, err = l.svcCtx.DBEngine.Insert(data)
	if err != nil {
		return nil, err
	}
	return &types.CreateShareRecordResponse{
		Identity: data.Identity,
	}, nil
}
