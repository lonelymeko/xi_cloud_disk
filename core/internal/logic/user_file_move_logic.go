// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"cloud_disk/core/models"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
)

type UserFileMoveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUserFileMoveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserFileMoveLogic {
	return &UserFileMoveLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UserFileMoveLogic) UserFileMove(req *types.UserFileMoveRequest) (resp *types.UserFileMoveResponse, err error) {
	userIdentity, ok := l.ctx.Value("user_identity").(string)
	if !ok {
		return nil, errors.New("用户身份验证失败")
	}
	parentData := new(models.UserRepository)
	has, err := l.svcCtx.DBEngine.Where("identity = ? AND user_identity = ?", req.ParentId, userIdentity).Get(parentData)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.New("目标文件夹不存在")
	}
	// 更新
	l.svcCtx.DBEngine.Table("user_repository").Where("identity = ? AND user_identity = ?", req.Identity, userIdentity).Update(&models.UserRepository{
		ParentId: req.ParentId,
	})

	return
}
