package models

// RepositoryPool 对应 repository_pool 表（文件存储池表）
type RepositoryPool struct {
	Id        int
	Identity  string
	Hash      string
	Name      string
	Ext       string
	Size      int64
	Path      string
	CreatedAt string `xorm:"created"`
	UpdatedAt string `xorm:"updated"`
	DeletedAt string `xorm:"deleted"`
}

// TableName 指定数据表名
func (table RepositoryPool) TableName() string {
	return "repository_pool"
}
