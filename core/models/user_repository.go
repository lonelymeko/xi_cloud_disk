package models

// 对应 user_repository 表（用户文件关联表）
type UserRepository struct {
	Id                 int
	Identity           string
	UserIdentity       string
	ParentId           int64
	RepositoryIdentity string
	Ext                string
	Name               string
	CreatedAt          string `xorm:"created"`
	UpdatedAt          string `xorm:"updated"`
	DeletedAt          string `xorm:"deleted"`
}

// 指定数据表名
func (table UserRepository) TableName() string {
	return "user_repository"
}
