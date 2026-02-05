package utils

import (
	"cloud_disk/core/models"
	"fmt"

	"xorm.io/xorm"
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
	return nil
}

func TablesHealthy(engine *xorm.Engine) error {
	names := []string{
		new(models.UserBasic).TableName(),
		new(models.RepositoryPool).TableName(),
		new(models.UserRepository).TableName(),
		new(models.ShareBasic).TableName(),
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
	return nil
}
