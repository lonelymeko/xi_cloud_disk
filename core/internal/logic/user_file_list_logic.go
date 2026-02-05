// goctl 生成代码，可安全编辑。
// goctl 1.9.2

package logic

import (
	"context"
	"errors"
	"time"

	"cloud_disk/core/common"
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"cloud_disk/core/models"

	"github.com/zeromicro/go-zero/core/logx"
)

// UserFileListLogic 用户文件列表逻辑。
type UserFileListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewUserFileListLogic 创建用户文件列表逻辑。
func NewUserFileListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserFileListLogic {
	return &UserFileListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// UserFileList 获取用户文件列表。
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
		Select("user_repository.id as id, user_repository.identity as identity, user_repository.name as name, "+
			"user_repository.repository_identity as repository_identity, user_repository.ext as ext, "+
			"repository_pool.size as size").
		Join("LEFT", "repository_pool", "user_repository.repository_identity = repository_pool.identity").
		Where("user_repository.status != ? OR user_repository.status IS NULL", common.StatusDeleted).
		// 筛选出「从未被标记删除」或「删除标记被重置为零值」的user_repository数据，即「有效数据」。
		Where("user_repository.deleted_at = ? OR user_repository.deleted_at IS NULL", time.Time{}.Format(common.DataTimeFormat)).
		Limit(int(size), int(offset)).
		Find(&uf)
	if err != nil {
		return nil, err
	}

	// 查询总数
	// TODO （可优化： 把总数存入 Redis）
	cnt, err = l.svcCtx.DBEngine.Table("user_repository").
		Where("parent_id = ? AND user_identity = ?", req.Id, userIdentity).
		Where("status != ? OR status IS NULL", common.StatusDeleted).
		Count(new(models.UserRepository))
	if err != nil {
		return nil, err
	}

	resp.List = uf
	resp.Count = cnt

	return
}
