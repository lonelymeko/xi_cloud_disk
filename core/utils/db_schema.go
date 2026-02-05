package utils

import (
	"cloud_disk/core/models"
	"fmt"

	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

func EnsureSchema(engine *xorm.Engine) error {
	if err := engine.Sync2(new(models.UserBasic)); err != nil {
		return fmt.Errorf("sync user_basic: %w", err)
	}
	if err := engine.Sync2(new(models.RepositoryPool)); err != nil {
		return fmt.Errorf("sync repository_pool: %w", err)
	}
	if err := engine.Sync2(new(models.UserRepository)); err != nil {
		return fmt.Errorf("sync user_repository: %w", err)
	}
	if err := engine.Sync2(new(models.ShareBasic)); err != nil {
		return fmt.Errorf("sync share_basic: %w", err)
	}
	if err := engine.Sync2(new(models.FileEventLog)); err != nil {
		return fmt.Errorf("sync file_event_log: %w", err)
	}
	return nil
}

func TablesHealthy(engine *xorm.Engine) error {
	names := []string{
		new(models.UserBasic).TableName(),
		new(models.RepositoryPool).TableName(),
		new(models.UserRepository).TableName(),
		new(models.ShareBasic).TableName(),
		new(models.FileEventLog).TableName(),
	}
	for _, n := range names {
		ok, err := engine.IsTableExist(n)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("table %s missing", n)
		}
	}
	metas, err := engine.DBMetas()
	if err != nil {
		return err
	}
	metaMap := map[string]*schemas.Table{}
	for _, m := range metas {
		metaMap[m.Name] = m
	}
	requiredCols := map[string][]string{
		new(models.RepositoryPool).TableName(): {"identity", "hash", "object_key", "status", "expire_at"},
		new(models.UserRepository).TableName(): {"identity", "user_identity", "repository_identity", "status", "expire_at", "parent_id"},
		new(models.ShareBasic).TableName():     {"identity", "repository_identity", "expired_time"},
		new(models.FileEventLog).TableName():   {"identity", "repository_identity", "user_identity", "event_type"},
	}
	for table, cols := range requiredCols {
		meta, ok := metaMap[table]
		if !ok {
			return fmt.Errorf("table %s meta missing", table)
		}
		for _, col := range cols {
			if meta.GetColumn(col) == nil {
				return fmt.Errorf("table %s missing column %s", table, col)
			}
		}
	}
	return nil
}
