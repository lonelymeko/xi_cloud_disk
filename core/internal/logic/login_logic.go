// goctl 生成代码，可安全编辑。
// goctl 1.9.2

package logic

import (
	"context"
	"errors"

	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"cloud_disk/core/models"
	"cloud_disk/core/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

// LoginLogic 登录逻辑。
type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewLoginLogic 创建登录逻辑。
func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Login 执行登录。
func (l *LoginLogic) Login(req *types.LoginRequest) (resp *types.LoginResponse, err error) {

	user := new(models.UserBasic)
	// 根据用户名和密码在数据库中查找用户
	has, err := l.svcCtx.DBEngine.Where("name = ? AND password = ?", req.Name, utils.Md5(req.Password)).Get(user)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.New("用户名或密码错误")
	}
	// 生成token
	token, err := utils.GenToken(utils.JwtPayLoad{
		Id:       user.Id,
		Identity: user.Identity,
		Name:     user.Name,
	}, l.svcCtx.Config.Auth.AccessSecret, l.svcCtx.Config.Auth.AccessExpire)
	if err != nil {
		return nil, err
	}
	return &types.LoginResponse{
		Token: token,
		Name:  req.Name,
	}, nil
}
