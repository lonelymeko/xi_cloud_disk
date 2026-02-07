// goctl 生成代码，可安全编辑。
// goctl 1.9.2

package logic

import (
	"context"

	"cloud_disk/core/common"
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"cloud_disk/core/models"
	"cloud_disk/core/utils"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
)

// SaveResourceLogic 保存资源逻辑。
type SaveResourceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewSaveResourceLogic 创建保存资源逻辑。
func NewSaveResourceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SaveResourceLogic {
	return &SaveResourceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// SaveResource 保存分享资源。
func (l *SaveResourceLogic) SaveResource(req *types.SaveResourceRequest) (resp *types.SaveResourceResponse, err error) {
	userIdentity, ok := l.ctx.Value("user_identity").(string)
	if !ok {
		return nil, errors.New("用户身份验证失败")
	}
	// 查询该层级是否有同名文件
	cnt, err := l.svcCtx.DBEngine.Table("user_repository").Where("name = ? AND parent_id = ? AND user_identity = ? AND (status != ? OR status IS NULL)", req.Name, req.ParentId, userIdentity, common.StatusDeleted).Count(new(models.UserRepository))
	if err != nil {
		return nil, err
	}
	if cnt > 0 {
		return nil, errors.New("该目录下已存在同名文件")
	}
	repo := new(models.RepositoryPool)
	has, err := l.svcCtx.DBEngine.Where("identity = ?", req.RepositoryIdentity).Get(repo)
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
		ParentId:           req.ParentId,
		RepositoryIdentity: repo.Identity,
		Ext:                repo.Ext,
		Name:               req.Name,
		Status:             common.StatusActive,
	}
	_, err = l.svcCtx.DBEngine.Insert(&data)
	if err != nil {
		return nil, err
	}
	return &types.SaveResourceResponse{
		Identity: data.Identity,
	}, nil
}
