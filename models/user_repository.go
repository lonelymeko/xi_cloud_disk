package models

// 对应 user_repository 表（用户文件关联表）
type UserRepository struct {
	Id                 int
	Identity           string
	UserIdentity       string
	ParentId           int
	RepositoryIdentity string
	Ext                string
	Name               string
	CreatedAt          string
	UpdatedAt          string
	DeletedAt          string
}

// 指定数据表名
func (table UserRepository) TableName() string {
	return "user_repository"
}
