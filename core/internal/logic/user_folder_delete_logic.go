// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"
	"errors"

	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"cloud_disk/core/models"

	"github.com/zeromicro/go-zero/core/logx"
)

type UserFolderDeleteLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUserFolderDeleteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserFolderDeleteLogic {
	return &UserFolderDeleteLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UserFolderDeleteLogic) UserFolderDelete(req *types.UserFolderDeleteRequest) (resp *types.UserFolderDeleteResponse, err error) {
	userIdentity, ok := l.ctx.Value("user_identity").(string)
	if !ok {
		return nil, errors.New("用户身份验证失败")
	}

	// 使用递归 CTE 一次性查询所有子项
	sql := `
        WITH RECURSIVE folder_tree AS (
            -- 初始查询：目标文件夹
            SELECT id, identity, parent_id, ext
            FROM user_repository
            WHERE identity = ? AND user_identity = ? AND deleted_at IS NULL
            
            UNION ALL
            
            -- 递归查询：所有子项
            SELECT ur.id, ur.identity, ur.parent_id, ur.ext
            FROM user_repository ur
            INNER JOIN folder_tree ft ON ur.parent_id = ft.id
            WHERE ur.user_identity = ? AND ur.deleted_at IS NULL
        )
        SELECT identity FROM folder_tree
    `

	var idsToDelete []string
	err = l.svcCtx.DBEngine.SQL(sql, req.Identity, userIdentity, userIdentity).Find(&idsToDelete)
	if err != nil {
		return nil, err
	}

	if len(idsToDelete) == 0 {
		return nil, errors.New("文件或文件夹不存在")
	}

	// 批量软删除（使用硬删除，xorm 的 Delete 方法）
	affected, err := l.svcCtx.DBEngine.
		Table("user_repository").
		In("identity", idsToDelete).
		Where("user_identity = ?", userIdentity).
		Delete(&models.UserRepository{})

	if err != nil {
		logx.Errorf("删除失败: %v", err)
		return nil, err
	}

	logx.Infof("成功删除 %d 个项目", affected)

	resp = &types.UserFolderDeleteResponse{}
	return resp, nil
}
