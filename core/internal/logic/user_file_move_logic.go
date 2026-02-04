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
	has, err := l.svcCtx.DBEngine.Where("id = ? AND user_identity = ?", req.ParentId, userIdentity).Get(parentData)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.New("目标文件夹不存在")
	}
	// 查询该层级是否有同名文件
	cnt, err := l.svcCtx.DBEngine.Table("user_repository").Where("name = ? AND parent_id = ? AND user_identity = ?", req.Name, req.Identity, userIdentity, userIdentity).Count(new(models.UserRepository))
	if err != nil {
		return nil, err
	}
	if cnt > 0 {
		return nil, errors.New("该目录下已存在同名文件")
	}
	// 更新
	l.svcCtx.DBEngine.Table("user_repository").Where("user_identity = ? AND identity = ?", userIdentity, req.Identity).Update(&models.UserRepository{
		ParentId: req.ParentId,
	})

	return
}
