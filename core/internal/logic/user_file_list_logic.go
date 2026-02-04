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

	"github.com/zeromicro/go-zero/core/logx"
)

type UserFileListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUserFileListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserFileListLogic {
	return &UserFileListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UserFileListLogic) UserFileList(req *types.UserFileListRequest) (resp *types.UserFileListResponse, err error) {
	uf := make([]*types.UserFile, 0)
	var cnt int64
	resp = new(types.UserFileListResponse)
	size := req.Size
	if size == 0 {
		size = common.PageSize
	}
	page := req.Page
	if page == 0 {
		page = 1
	}
	offset := (page - 1) * size

	// 获取用户身份
	userIdentity, ok := l.ctx.Value("user_identity").(string)
	if !ok {
		return nil, errors.New("用户身份验证失败")
	}

	// 查询文件列表（修复 JOIN 条件和添加 name 字段）
	err = l.svcCtx.DBEngine.Table("user_repository").
		Where("parent_id = ? AND user_identity = ?", req.Id, userIdentity).
		Select("user_repository.id, user_repository.identity, user_repository.name, "+
			"user_repository.repository_identity, user_repository.ext, "+
			"repository_pool.path, repository_pool.size").
		Join("LEFT", "repository_pool", "user_repository.repository_identity = repository_pool.identity").
		Limit(int(size), int(offset)).
		Find(&uf)
	if err != nil {
		return nil, err
	}

	// 查询总数
	// TODO （可优化： 把总数存入 Redis）
	cnt, err = l.svcCtx.DBEngine.Table("user_repository").
		Where("parent_id = ? AND user_identity = ?", req.Id, userIdentity).
		Count(new(models.UserRepository))
	if err != nil {
		return nil, err
	}

	resp.List = uf
	resp.Count = cnt

	return
}
