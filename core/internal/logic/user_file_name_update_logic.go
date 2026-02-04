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

type UserFileNameUpdateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUserFileNameUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserFileNameUpdateLogic {
	return &UserFileNameUpdateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UserFileNameUpdateLogic) UserFileNameUpdate(req *types.UserFileNameUpdateRequest) (resp *types.UserFileNameUpdateResponse, err error) {
	data := models.UserRepository{Name: req.Name}
	// 获取用户身份
	userIdentity, ok := l.ctx.Value("user_identity").(string)
	if !ok {
		return nil, errors.New("用户身份验证失败")
	}
	// 先查询该层级是否有同名文件
	cnt, err := l.svcCtx.DBEngine.Table("user_repository").Where("name = ? AND parent_id = (SELECT parent_id FROM user_repository WHERE identity = ? AND user_identity = ?) AND user_identity = ?", req.Name, req.Identity, userIdentity, userIdentity).Count(new(models.UserRepository))
	if err != nil {
		return nil, err
	}
	if cnt > 0 {
		return nil, errors.New("该目录下已存在同名文件")
	}
	// 修改文件名
	_, err = l.svcCtx.DBEngine.Table("user_repository").Where("identity = ? AND user_identity = ?", req.Identity, userIdentity).Update(data)
	if err != nil {
		return nil, err
	}

	return &types.UserFileNameUpdateResponse{}, nil
}
