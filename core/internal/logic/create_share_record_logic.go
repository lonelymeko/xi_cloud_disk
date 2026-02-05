// goctl 生成代码，可安全编辑。
// goctl 1.9.2

package logic

import (
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"cloud_disk/core/models"
	"cloud_disk/core/utils"
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
)

// CreateShareRecordLogic 创建分享记录逻辑。
type CreateShareRecordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewCreateShareRecordLogic 创建分享记录逻辑。
func NewCreateShareRecordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateShareRecordLogic {
	return &CreateShareRecordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// CreateShareRecord 创建分享记录。
func (l *CreateShareRecordLogic) CreateShareRecord(req *types.CreateShareRecordRequest) (resp *types.CreateShareRecordResponse, err error) {
	userIdentity, ok := l.ctx.Value("user_identity").(string)
	if !ok {
		return nil, errors.New("用户身份验证失败")
	}
	data := new(models.ShareBasic)
	data.UserIdentity = userIdentity
	data.RepositoryIdentity = req.Identity
	data.ExpiredTime = req.ExpiredTime
	data.Identity = utils.UUID()
	_, err = l.svcCtx.DBEngine.Insert(data)
	if err != nil {
		return nil, err
	}
	return &types.CreateShareRecordResponse{
		Identity: data.Identity,
	}, nil
}
