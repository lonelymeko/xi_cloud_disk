// goctl 生成代码，可安全编辑。
// goctl 1.9.2

package logic

import (
	"context"
	"errors"

	"cloud_disk/core/common"
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"cloud_disk/core/models"
	"cloud_disk/core/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

// UserFolderCreateLogic 用户文件夹创建逻辑。
type UserFolderCreateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewUserFolderCreateLogic 创建用户文件夹创建逻辑。
func NewUserFolderCreateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserFolderCreateLogic {
	return &UserFolderCreateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// UserFolderCreate 创建用户文件夹。
func (l *UserFolderCreateLogic) UserFolderCreate(req *types.UserFolderCreateRequest) (resp *types.UserFolderCreateResponse, err error) {
	// 获取用户身份
	userIdentity, ok := l.ctx.Value("user_identity").(string)
	if !ok {
		return nil, errors.New("用户身份验证失败")
	}
	// 先查询该层级是否有同名文件
	cnt, err := l.svcCtx.DBEngine.Table("user_repository").Where("name = ? AND parent_id = ? AND user_identity = ?", req.Name, req.ParentId, userIdentity).Count(new(models.UserRepository))
	if err != nil {
		return nil, err
	}
	if cnt > 0 {
		return nil, errors.New("该目录下已存在同名文件")
	}
	// 创建文件夹
	data := &models.UserRepository{
		Identity:     utils.UUID(),
		UserIdentity: userIdentity,
		ParentId:     req.ParentId,
		Name:         req.Name,
		Status:       common.StatusActive,
	}
	_, err = l.svcCtx.DBEngine.Table("user_repository").Insert(data)
	if err != nil {
		return nil, err
	}
	if data.Id == 0 {
		_, err = l.svcCtx.DBEngine.Table("user_repository").Where("identity = ?", data.Identity).Get(data)
		if err != nil {
			return nil, err
		}
	}

	return &types.UserFolderCreateResponse{Id: int64(data.Id), Identity: data.Identity}, nil
}
