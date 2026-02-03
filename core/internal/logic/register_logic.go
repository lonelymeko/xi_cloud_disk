// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"cloud_disk/core/models"
	"cloud_disk/core/utils"
	"context"
	"errors"

	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"fmt"

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
	has, err := l.svcCtx.DBEngine.Where("name = ? or email = ?", req.Name, req.Email).Get(user)
	if err != nil {
		return nil, err
	}
	if has {
		return nil, errors.New("用户已存在")
	}
	// 查询验证码
	var code string
	err = l.svcCtx.RedisClient.Get(l.ctx, fmt.Sprintf("verification_code:%s", req.Email)).Scan(&code)
	if err != nil {
		logx.Errorf("查询验证码失败: %v", err)
		return nil, err
	}
	if code == "" {
		return nil, errors.New("验证码已过期或无效")
	}
	// 验证码正确，创建用户
	if req.Code != code {
		return nil, errors.New("验证码错误")
	}
	uuid := utils.UUID()

	// 创建用户模型对象
	userModel := &models.UserBasic{
		Name:     req.Name,
		Password: utils.Md5(req.Password),
		Email:    req.Email,
		Identity: uuid, // 可以生成一个唯一标识
	}
	// 插入数据库
	affected, err := l.svcCtx.DBEngine.InsertOne(userModel)
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, errors.New("用户创建失败")
	}

	// 直接使用插入的用户数据生成token，无需再次查询
	// xorm 的 InsertOne 会自动将自增 ID 填充到 userModel.Id 中
	token, err := utils.GenToken(utils.JwtPayLoad{
		Id:       userModel.Id,
		Identity: userModel.Identity,
		Name:     userModel.Name,
	}, l.svcCtx.Config.Auth.AccessSecret, l.svcCtx.Config.Auth.AccessExpire)
	if err != nil {
		return nil, err
	}
	return &types.RegisterResponse{
		Token: token,
		Name:  req.Name,
	}, nil

}
