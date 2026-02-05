// goctl 生成代码，可安全编辑。
// goctl 1.9.2

package logic

import (
	"cloud_disk/core/models"
	"context"
	"errors"

	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

// UserDetailLogic 用户详情逻辑。
type UserDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewUserDetailLogic 创建用户详情逻辑。
func NewUserDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserDetailLogic {
	return &UserDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// UserDetail 获取用户详情。
func (l *UserDetailLogic) UserDetail(req *types.UserDetailRequest) (resp *types.UserDetailResponse, err error) {
	resp = &types.UserDetailResponse{}
	ub := new(models.UserBasic)
	has, err := l.svcCtx.DBEngine.Where("identity=?", req.Identity).Get(ub)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.New("用户不存在")
	}
	resp.Name = ub.Name
	resp.Email = ub.Email
	return
}
