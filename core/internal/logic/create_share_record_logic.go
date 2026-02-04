// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateShareRecordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateShareRecordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateShareRecordLogic {
	return &CreateShareRecordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateShareRecordLogic) CreateShareRecord(req *types.CreateShareRecordRequest) (resp *types.CreateShareRecordResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
