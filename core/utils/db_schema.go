package utils

import (
	"cloud_disk/core/models"
	"fmt"

	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

// EnsureSchema 同步数据库表结构。
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
	if err := ensureAutoIncrement(engine); err != nil {
		return err
	}
	return nil
}

// TablesHealthy 校验表存在性与结构完整性。
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
	if err := ensureColumnTypes(metaMap); err != nil {
		return err
	}
	if err := ensureIndexes(metaMap); err != nil {
		return err
	}
	return nil
}

func ensureAutoIncrement(engine *xorm.Engine) error {
	metas, err := engine.DBMetas()
	if err != nil {
		return err
	}
	metaMap := map[string]*schemas.Table{}
	for _, m := range metas {
		metaMap[m.Name] = m
	}
	return ensureTableAutoIncrement(engine, metaMap, new(models.UserRepository).TableName(), "id")
}

func ensureTableAutoIncrement(engine *xorm.Engine, metaMap map[string]*schemas.Table, tableName, column string) error {
	meta := metaMap[tableName]
	if meta == nil {
		return fmt.Errorf("table %s meta missing", tableName)
	}
	col := meta.GetColumn(column)
	if col == nil {
		return fmt.Errorf("table %s missing column %s", tableName, column)
	}
	if col.IsAutoIncrement && hasPrimaryKey(meta, column) {
		return nil
	}
	if engine.Dialect().URI().DBType != schemas.MYSQL {
		return fmt.Errorf("table %s column %s autoincrement missing", tableName, column)
	}
	_, err := engine.Exec(fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s BIGINT NOT NULL AUTO_INCREMENT", tableName, column))
	if err != nil {
		return fmt.Errorf("table %s modify autoincrement failed: %w", tableName, err)
	}
	if !hasPrimaryKey(meta, column) {
		_, err = engine.Exec(fmt.Sprintf("ALTER TABLE %s ADD PRIMARY KEY (%s)", tableName, column))
		if err != nil {
			return fmt.Errorf("table %s add primary key failed: %w", tableName, err)
		}
	}
	return nil
}

func hasPrimaryKey(table *schemas.Table, column string) bool {
	if table == nil || len(table.PrimaryKeys) == 0 {
		return false
	}
	for _, key := range table.PrimaryKeys {
		if key == column {
			return true
		}
	}
	return false
}

// ensureColumnTypes 校验关键字段类型。
func ensureColumnTypes(metaMap map[string]*schemas.Table) error {
	repo := metaMap[new(models.RepositoryPool).TableName()]
	if repo == nil {
		return fmt.Errorf("table %s meta missing", new(models.RepositoryPool).TableName())
	}
	if err := checkColumnType(repo, "expire_at", []string{"datetime", "timestamp"}); err != nil {
		return err
	}
	if err := checkColumnType(repo, "status", []string{"varchar", "char", "text"}); err != nil {
		return err
	}

	userRepo := metaMap[new(models.UserRepository).TableName()]
	if userRepo == nil {
		return fmt.Errorf("table %s meta missing", new(models.UserRepository).TableName())
	}
	if err := checkColumnType(userRepo, "expire_at", []string{"datetime", "timestamp"}); err != nil {
		return err
	}
	if err := checkColumnType(userRepo, "status", []string{"varchar", "char", "text"}); err != nil {
		return err
	}
	return nil
}

// ensureIndexes 校验关键索引存在。
func ensureIndexes(metaMap map[string]*schemas.Table) error {
	userRepo := metaMap[new(models.UserRepository).TableName()]
	if userRepo == nil {
		return fmt.Errorf("table %s meta missing", new(models.UserRepository).TableName())
	}
	if !indexHasColumns(userRepo, []string{"user_identity", "parent_id", "status"}) {
		return fmt.Errorf("table %s missing index on user_identity,parent_id,status", userRepo.Name)
	}

	repo := metaMap[new(models.RepositoryPool).TableName()]
	if repo == nil {
		return fmt.Errorf("table %s meta missing", new(models.RepositoryPool).TableName())
	}
	if !indexHasColumns(repo, []string{"hash", "status"}) {
		return fmt.Errorf("table %s missing index on hash,status", repo.Name)
	}
	return nil
}

// checkColumnType 校验字段类型是否在允许列表中。
func checkColumnType(table *schemas.Table, column string, allowed []string) error {
	col := table.GetColumn(column)
	if col == nil {
		return fmt.Errorf("table %s missing column %s", table.Name, column)
	}
	colType := col.SQLType.Name
	for _, allow := range allowed {
		if colType == allow {
			return nil
		}
	}
	return fmt.Errorf("table %s column %s type %s not allowed", table.Name, column, colType)
}

// indexHasColumns 判断是否存在匹配字段集合的索引。
func indexHasColumns(table *schemas.Table, cols []string) bool {
	if table == nil || len(table.Indexes) == 0 {
		return false
	}
	for _, idx := range table.Indexes {
		if equalColumns(idx.Cols, cols) {
			return true
		}
	}
	return false
}

// equalColumns 判断两个字段集合是否等价。
func equalColumns(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	set := map[string]int{}
	for _, v := range left {
		set[v]++
	}
	for _, v := range right {
		if set[v] == 0 {
			return false
		}
		set[v]--
	}
	for _, v := range set {
		if v != 0 {
			return false
		}
	}
	return true
}
