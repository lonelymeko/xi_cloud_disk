// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"cloud_disk/models"
	"cloud_disk/utils"
	"context"
	"errors"

	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(req *types.RegisterRequest) (resp *types.RegisterResponse, err error) {
	user := new(models.UserBasic)
	// 先根据用户名在数据库中查找用户是否存在
	has, err := l.svcCtx.DBEngine.Where("name = ?", req.Name).Get(user)
	if err != nil {
		return nil, err
	}
	if has {
		return nil, errors.New("用户已存在")
	}

	// 创建用户模型对象
	userModel := &models.UserBasic{
		Name:     req.Name,
		Password: utils.Md5(req.Password),
		Email:    req.Email,
		Identity: "", // 可以生成一个唯一标识
	}
	// 插入数据库
	_, err = l.svcCtx.DBEngine.InsertOne(userModel)
	if err != nil {
		return nil, err
	}
	has, err = l.svcCtx.DBEngine.Where("name = ?", req.Name).Get(user)
	if err != nil || user == nil {
		return nil, err
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
	return &types.RegisterResponse{
		Token: token,
		Name:  req.Name,
	}, nil

}
