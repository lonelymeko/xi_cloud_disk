// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"cloud_disk/core/models"
	"cloud_disk/core/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type UserRepositoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUserRepositoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserRepositoryLogic {
	return &UserRepositoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UserRepositoryLogic) UserRepository(req *types.UserRepositoryRequest) (resp *types.UserRepositoryResponse, err error) {
	ur := &models.UserRepository{
		Identity:           utils.UUID(),
		UserIdentity:       l.ctx.Value("user_identity").(string),
		RepositoryIdentity: req.RepositoryIdentity,
		ParentId:           req.ParentId,
		Ext:                req.Ext,
		Name:               req.Name,
	}
	_, err = l.svcCtx.DBEngine.Insert(ur)
	if err != nil {
		return nil, err
	}
	resp = &types.UserRepositoryResponse{
		Identity: ur.Identity,
	}
	return
}
