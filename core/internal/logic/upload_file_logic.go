// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

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

func (l *UploadFileLogic) UploadFile(req *types.UploadFileRequest) (resp *types.UploadFileResponse, err error) {
	rp := &models.RepositoryPool{
		Name:     req.Name,
		Hash:     req.Hash,
		Ext:      req.Ext,
		Size:     req.Size,
		Path:     req.Path,
		Identity: utils.UUID(),
	}
	_, err = l.svcCtx.DBEngine.Insert(rp)
	if err != nil {
		return nil, err
	}

	_, err = l.InsertInToUserRepository(rp.Identity, rp.Ext, rp.Name, req.ParentId)
	if err != nil {
		return nil, err
	}
	return &types.UploadFileResponse{
		Identity: rp.Identity,
		Name:     rp.Name,
		Ext:      rp.Ext,
	}, nil
}

func (l *UploadFileLogic) InsertInToUserRepository(repositoryIdentity, ext, name string, parentId int64) (userRepositoryIdentity string, err error) {
	ur := &models.UserRepository{
		Identity:           utils.UUID(),
		UserIdentity:       l.ctx.Value("user_identity").(string),
		RepositoryIdentity: repositoryIdentity,
		ParentId:           parentId,
		Ext:                ext,
		Name:               name,
	}
	_, err = l.svcCtx.DBEngine.Insert(ur)
	if err != nil {
		return "", err
	}
	return ur.Identity, nil
}
