package models

// UserBasic 对应 user_basic 表（用户基础信息表）。
type UserBasic struct {
	Id        int
	Identity  string
	Name      string
	Password  string
	Email     string
	Role      string
	CreatedAt string `xorm:"created"`
	UpdatedAt string `xorm:"updated"`
	DeletedAt string `xorm:"deleted"`
}

// TableName 指定数据表名。
func (table UserBasic) TableName() string {
	return "user_basic"
}
