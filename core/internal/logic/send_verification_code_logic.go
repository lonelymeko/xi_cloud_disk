// goctl 生成代码，可安全编辑。
// goctl 1.9.2

package logic

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"cloud_disk/core/utils"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// SendVerificationCodeLogic 发送验证码逻辑。
type SendVerificationCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewSendVerificationCodeLogic 创建发送验证码逻辑。
func NewSendVerificationCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendVerificationCodeLogic {
	return &SendVerificationCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// SendVerificationCode 发送验证码。
func (l *SendVerificationCodeLogic) SendVerificationCode(req *types.SendVerificationCodeRequest) (resp *types.SendVerificationCodeResponse, err error) {
	// 生成随机 6 位数验证码
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return nil, err
	}
	code := fmt.Sprintf("%06d", n.Int64())

	if !utils.EmailEnabled() {
		logx.Infof("邮箱发送已禁用，验证码: %s", code)
	} else {
		go func() {
			sendErr := utils.SendEmail(req.Email, code)
			if sendErr != nil {
				logx.Errorf("发送验证码邮件失败: %v", sendErr)
			}
		}()
	}
	// 向 Redis 中存储验证码
	err = l.svcCtx.RedisClient.Set(l.ctx, fmt.Sprintf("verification_code:%s", req.Email), code, 5*time.Minute).Err()
	if err != nil {
		logx.Errorf("向 Redis 存储验证码失败: %v", err)
		return nil, err
	}

	return &types.SendVerificationCodeResponse{
		Message: "验证码已发送",
	}, nil
}
