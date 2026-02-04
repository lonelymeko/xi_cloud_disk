// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"cloud_disk/core/models"
	"cloud_disk/core/utils"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
)

type SaveResourceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSaveResourceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SaveResourceLogic {
	return &SaveResourceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SaveResourceLogic) SaveResource(req *types.SaveResourceRequest) (resp *types.SaveResourceResponse, err error) {
	userIdentity, ok := l.ctx.Value("user_identity").(string)
	if !ok {
		return nil, errors.New("用户身份验证失败")
	}
	// 查询该层级是否有同名文件
	cnt, err := l.svcCtx.DBEngine.Table("user_repository").Where("name = ? AND parent_id = ? AND user_identity = ?", req.Name, req.ParentId, userIdentity).Count(new(models.UserRepository))
	if err != nil {
		return nil, err
	}
	if cnt > 0 {
		return nil, errors.New("该目录下已存在同名文件")
	}
	rp := new(models.UserRepository)
	// 查询信息
	has, err := l.svcCtx.DBEngine.Where("identity = ?", req.RepositoryIdentity, userIdentity).Get(rp)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.New("资源不存在")
	}

	// 创造结构体并存入
	data := models.UserRepository{
		Identity:           utils.UUID(),
		UserIdentity:       userIdentity,
		ParentId:           rp.ParentId,
		RepositoryIdentity: rp.RepositoryIdentity,
		Ext:                rp.Ext,
		Name:               req.Name,
	}
	_, err = l.svcCtx.DBEngine.Insert(&data)
	if err != nil {
		return nil, err
	}
	return &types.SaveResourceResponse{
		Identity: data.Identity,
	}, nil
}
