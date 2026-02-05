package logic

import (
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"cloud_disk/core/models"
	"cloud_disk/core/utils"
	"context"
	"errors"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
)

type ChangePasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewChangePasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangePasswordLogic {
	return &ChangePasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ChangePasswordLogic) ChangePassword(req *types.ChangePasswordRequest) (resp *types.ChangePasswordResponse, err error) {
	identity, err := resolveChangePasswordIdentity(l.ctx, req)
	if err != nil {
		return nil, err
	}

	logx.Debugf("password update request identity=%s", identity)

	user := new(models.UserBasic)
	has, err := l.svcCtx.DBEngine.Where("identity = ?", identity).Get(user)
	if err != nil {
		logx.Severef("password update query failed identity=%s err=%v", identity, err)
		return nil, err
	}
	if !has {
		logx.Errorf("password update user not found identity=%s", identity)
		return nil, errors.New("用户不存在")
	}
	if user.Password != utils.Md5(req.OldPassword) {
		logx.Errorf("password update old password mismatch identity=%s", identity)
		return nil, errors.New("旧密码错误")
	}
	if req.OldPassword == req.NewPassword {
		return nil, errors.New("新密码不能与旧密码相同")
	}
	if !isPasswordStrong(req.NewPassword) {
		return nil, errors.New("密码强度不足")
	}

	update := &models.UserBasic{Password: utils.Md5(req.NewPassword)}
	affected, err := l.svcCtx.DBEngine.Where("identity = ?", identity).Cols("password").Update(update)
	if err != nil {
		logx.Severef("password update failed identity=%s err=%v", identity, err)
		return nil, err
	}
	if affected == 0 {
		logx.Errorf("password update affected 0 identity=%s", identity)
		return nil, errors.New("更新失败")
	}

	logx.Infof("password update success identity=%s", identity)
	return &types.ChangePasswordResponse{Message: "密码更新成功"}, nil
}

func resolveChangePasswordIdentity(ctx context.Context, req *types.ChangePasswordRequest) (string, error) {
	identity := strings.TrimSpace(req.Identity)
	if v := ctx.Value("user_identity"); v != nil {
		if s, ok := v.(string); ok {
			if identity == "" {
				identity = s
			} else if identity != s {
				logx.Errorf("password update identity mismatch req=%s ctx=%s", identity, s)
				return "", errors.New("身份不匹配")
			}
		}
	}
	if identity == "" || strings.TrimSpace(req.OldPassword) == "" || strings.TrimSpace(req.NewPassword) == "" {
		return "", errors.New("参数不能为空")
	}
	return identity, nil
}

func isPasswordStrong(password string) bool {
	if len(password) < 8 {
		return false
	}
	var hasLetter bool
	var hasNumber bool
	for _, r := range password {
		if r >= '0' && r <= '9' {
			hasNumber = true
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			hasLetter = true
		}
	}
	return hasLetter && hasNumber
}
