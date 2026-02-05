// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"
	"errors"

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
	// 特判：文件存在且上传的文件在当前父目录下已存在且用户 id 一致
	if isExisted {
		had, err := l.svcCtx.DBEngine.Table("user_repository").Where("repository_identity=? AND Identity", repositoryIdentity, userIdentity).Get(ur)
		if err != nil {
			return nil, err
		}
		// 直接返回：文件已存在
		if had {
			return &types.UploadFileResponse{
				Message: "文件已存在",
			}, nil
		}
	} else {
		// 文件不存在就存入中央数据库
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
	}
	// 最终都要逻辑添加到用户文件表
	_, err = l.InsertInToUserRepository(userIdentity, req.Ext, req.Name, req.ParentId)
	if err != nil {
		return nil, err
	}

	return &types.UploadFileResponse{
		Message: "文件上传开始",
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
