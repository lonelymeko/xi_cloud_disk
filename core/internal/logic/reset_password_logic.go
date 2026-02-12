package logic

import (
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"cloud_disk/core/models"
	"cloud_disk/core/utils"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
)

// ResetPasswordLogic 重置密码逻辑。
type ResetPasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewResetPasswordLogic 创建重置密码逻辑。
func NewResetPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetPasswordLogic {
	return &ResetPasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ResetPassword 重置密码。
func (l *ResetPasswordLogic) ResetPassword(req *types.ResetPasswordRequest) (resp *types.ResetPasswordResponse, err error) {
	email, code, newPassword, err := normalizeResetPasswordInput(req)
	if err != nil {
		return nil, err
	}

	var cached string
	err = l.svcCtx.RedisClient.Get(l.ctx, fmt.Sprintf("verification_code:%s", email)).Scan(&cached)
	if err != nil {
		logx.Errorf("password reset query code failed email=%s err=%v", email, err)
		return nil, err
	}
	if cached == "" {
		return nil, errors.New("验证码已过期或无效")
	}
	if cached != code {
		return nil, errors.New("验证码错误")
	}

	user := new(models.UserBasic)
	has, err := l.svcCtx.DBEngine.Where("email = ?", email).Get(user)
	if err != nil {
		logx.Severef("password reset query failed email=%s err=%v", email, err)
		return nil, err
	}
	if !has {
		return nil, errors.New("用户不存在")
	}

    update := &models.UserBasic{Password: utils.Md5(utils.DecodeMaybeBase64(newPassword))}
	affected, err := l.svcCtx.DBEngine.Where("email = ?", email).Cols("password").Update(update)
	if err != nil {
		logx.Severef("password reset update failed email=%s err=%v", email, err)
		return nil, err
	}
	if affected == 0 {
		return nil, errors.New("更新失败")
	}

	_, _ = l.svcCtx.RedisClient.Del(l.ctx, fmt.Sprintf("verification_code:%s", email)).Result()

	logx.Infof("password reset success email=%s", email)
	return &types.ResetPasswordResponse{Message: "密码重置成功"}, nil
}

// normalizeResetPasswordInput 标准化并校验重置密码参数。
func normalizeResetPasswordInput(req *types.ResetPasswordRequest) (string, string, string, error) {
	email := strings.TrimSpace(req.Email)
	code := strings.TrimSpace(req.Code)
	newPassword := strings.TrimSpace(req.NewPassword)
	if email == "" || code == "" || newPassword == "" {
		return "", "", "", errors.New("参数不能为空")
	}
	if !isResetPasswordStrong(newPassword) {
		return "", "", "", errors.New("密码强度不足")
	}
	return email, code, newPassword, nil
}

// isResetPasswordStrong 校验重置密码强度。
func isResetPasswordStrong(password string) bool {
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
