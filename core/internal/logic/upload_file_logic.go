// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"
	"errors"

	"cloud_disk/core/common"
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"cloud_disk/core/models"
	"cloud_disk/core/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type UploadFileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUploadFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadFileLogic {
	return &UploadFileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UploadFileLogic) UploadFile(req *types.UploadFileRequest, isExisted bool, repositoryIdentity string) (resp *types.UploadFileResponse, err error) {
	userIdentity, ok := l.ctx.Value("user_identity").(string)
	if !ok {
		return nil, errors.New("用户身份验证失败")
	}
	ur := new(models.UserRepository)
	if isExisted {
		had, err := l.svcCtx.DBEngine.Table("user_repository").
			Where("repository_identity = ? AND user_identity = ? AND parent_id = ?", repositoryIdentity, userIdentity, req.ParentId).
			Get(ur)
		if err != nil {
			return nil, err
		}
		if had {
			return &types.UploadFileResponse{Message: "文件已存在"}, nil
		}
	} else {
		rp := &models.RepositoryPool{
			Name:      req.Name,
			Hash:      req.Hash,
			Ext:       req.Ext,
			Size:      req.Size,
			ObjectKey: req.ObjectKey,
			Status:    common.StatusActive,
			Identity:  utils.UUID(),
		}
		_, err = l.svcCtx.DBEngine.Insert(rp)
		if err != nil {
			return nil, err
		}
		repositoryIdentity = rp.Identity
	}
	_, err = l.InsertInToUserRepository(repositoryIdentity, req.Ext, req.Name, req.ParentId)
	if err != nil {
		return nil, err
	}

	return &types.UploadFileResponse{Message: "文件上传开始"}, nil
}

func (l *UploadFileLogic) InsertInToUserRepository(repositoryIdentity, ext, name string, parentId int64) (userRepositoryIdentity string, err error) {
	ur := &models.UserRepository{
		Identity:           utils.UUID(),
		UserIdentity:       l.ctx.Value("user_identity").(string),
		RepositoryIdentity: repositoryIdentity,
		ParentId:           parentId,
		Ext:                ext,
		Name:               name,
		Status:             common.StatusActive,
	}
	_, err = l.svcCtx.DBEngine.Insert(ur)
	if err != nil {
		return "", err
	}
	return ur.Identity, nil
}
