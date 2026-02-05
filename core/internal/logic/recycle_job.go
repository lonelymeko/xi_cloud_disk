package logic

import (
	"context"
	"time"

	"cloud_disk/core/common"
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/models"
	"cloud_disk/core/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

func StartRecycleJob(ctx context.Context, svcCtx *svc.ServiceContext) {
	interval := utils.RecycleScanInterval()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				purgeExpired(ctx, svcCtx)
			}
		}
	}()
}

func purgeExpired(ctx context.Context, svcCtx *svc.ServiceContext) {
	now := time.Now().Format(common.DataTimeFormat)
	var expired []models.UserRepository
	err := svcCtx.DBEngine.Table("user_repository").
		Where("status = ? AND expire_at != '' AND expire_at <= ?", common.StatusDeleted, now).
		Find(&expired)
	if err != nil {
		logx.Errorf("purge list failed: %v", err)
		return
	}
	if len(expired) == 0 {
		return
	}
	repoSet := map[string]struct{}{}
	for _, item := range expired {
		if item.RepositoryIdentity != "" {
			repoSet[item.RepositoryIdentity] = struct{}{}
		}
	}
	for repoID := range repoSet {
		cnt, err := svcCtx.DBEngine.Table("user_repository").
			Where("repository_identity = ? AND (status != ? OR status IS NULL)", repoID, common.StatusDeleted).
			Count(new(models.UserRepository))
		if err != nil {
			logx.Errorf("purge count failed: %v", err)
			continue
		}
		if cnt > 0 {
			continue
		}
		repo := new(models.RepositoryPool)
		has, err := svcCtx.DBEngine.Where("identity = ?", repoID).Get(repo)
		if err != nil || !has {
			continue
		}
		objectKey := repo.ObjectKey
		if objectKey == "" {
			objectKey = utils.ObjectKeyFromPath(repo.Path)
		}
		_, _ = svcCtx.DBEngine.Table("repository_pool").
			Where("identity = ?", repoID).
			Update(map[string]any{"status": common.StatusPurging})
		if objectKey != "" {
			if err := utils.DeleteOSSObject(ctx, objectKey); err != nil {
				logx.Errorf("oss delete failed: %v", err)
				continue
			}
		}
		_, _ = svcCtx.DBEngine.Table("repository_pool").
			Where("identity = ?", repoID).
			Update(map[string]any{"status": common.StatusPurged, "deleted_at": now})
		_, _ = svcCtx.DBEngine.Table("user_repository").
			Where("repository_identity = ? AND status = ? AND expire_at != '' AND expire_at <= ?", repoID, common.StatusDeleted, now).
			Update(map[string]any{"status": common.StatusPurged})
		_, _ = svcCtx.DBEngine.Insert(&models.FileEventLog{
			Identity:           utils.UUID(),
			RepositoryIdentity: repoID,
			EventType:          common.EventPurge,
		})
	}
}
