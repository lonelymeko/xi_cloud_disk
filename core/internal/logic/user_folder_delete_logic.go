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

// func (l *UserFolderDeleteLogic) UserFolderDelete(req *types.UserFolderDeleteRequest) (resp *types.UserFolderDeleteResponse, err error) {
// 	// 获取用户身份
// 	userIdentity, ok := l.ctx.Value("user_identity").(string)
// 	if !ok {
// 		return nil, errors.New("用户身份验证失败")
// 	}

// 	go l.deleteUserFolder(userIdentity, req.Identity)

// 	return
// }

// func (l *UserFolderDeleteLogic) deleteUserFolder(userIdentity string, parentId string) {
// 	// 如果是文件夹，查询出所有子文件夹
// 	// 循环查询出所有的子 id（根据父 id 查）
// 	var datas []models.UserRepository
// 	// 收集所有的子 id，最后批量删除
// 	var ids []string
// 	data := models.UserRepository{}
// 	for {
// 		// 先查询是不是文件夹，不是就直接删除
// 		l.svcCtx.DBEngine.Where("user_identity = ? AND identity = ?", userIdentity, parentId).Find(&data)
// 		if data.Ext != "" || strings.HasPrefix(data.Ext, ".") {
// 			l.svcCtx.DBEngine.Where("identity = ?", parentId).Delete(&models.UserRepository{})
// 			return
// 		}

// 		l.svcCtx.DBEngine.Where("user_identity = ? AND parent_id = ?", userIdentity, parentId).Find(&datas)
// 		for _, item := range datas {
// 			ids = append(ids, item.Identity)
// 			parentId = item.Identity
// 			l.deleteUserFolder(userIdentity, parentId)
// 		}
// 		if len(datas) == 0 {
// 			break
// 		}

// 	}
// 	// 批量删除
// 	_, err := l.svcCtx.DBEngine.In("identity", ids).Delete(datas)
// 	if err != nil {
// 		fmt.Errorf("failed to delete user folder: %w", err)
// 	}

// }
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

	// 批量软删除
	_, err = l.svcCtx.DBEngine.
		Table("user_repository").
		In("identity", idsToDelete).
		Where("user_identity = ?", userIdentity).
		Delete(&[]models.UserRepository{})

	if err != nil {
		return nil, err
	}

	resp = &types.UserFolderDeleteResponse{}
	return resp, nil
}
