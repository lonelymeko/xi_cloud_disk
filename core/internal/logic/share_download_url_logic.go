package logic

import (
	"context"
	"errors"
	"time"

	"cloud_disk/core/common"
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"cloud_disk/core/models"
	"cloud_disk/core/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type ShareDownloadURLLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewShareDownloadURLLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ShareDownloadURLLogic {
	return &ShareDownloadURLLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ShareDownloadURLLogic) ShareDownloadURL(req *types.ShareDownloadURLRequest) (resp *types.ShareDownloadURLResponse, err error) {
	if req.ShareIdentity == "" {
		return nil, errors.New("分享标识不能为空")
	}
	expires := normalizeExpires(req.Expires)

	share := new(models.ShareBasic)
	has, err := l.svcCtx.DBEngine.Where("identity = ?", req.ShareIdentity).Get(share)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.New("分享不存在")
	}
	if share.ExpiredTime > 0 {
		createdAt, parseErr := time.Parse(common.DataTimeFormat, share.CreatedAt)
		if parseErr != nil {
			return nil, parseErr
		}
		if createdAt.Add(time.Duration(share.ExpiredTime) * time.Second).Before(time.Now()) {
			return nil, errors.New("分享已过期")
		}
	}

	repo := new(models.RepositoryPool)
	has, err = l.svcCtx.DBEngine.Where("identity = ?", share.RepositoryIdentity).Get(repo)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.New("文件不存在")
	}
	objectKey := repo.ObjectKey
	if objectKey == "" {
		objectKey = utils.ObjectKeyFromPath(repo.Path)
	}
	if objectKey == "" {
		return nil, errors.New("文件未绑定对象键")
	}

	url, err := utils.PresignGetObject(l.ctx, objectKey, time.Duration(expires)*time.Second)
	if err != nil {
		return nil, err
	}
	return &types.ShareDownloadURLResponse{URL: url, Expires: expires}, nil
}
