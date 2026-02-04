package models

type UserBasic struct {
	Id        int
	Identity  string
	Name      string
	Password  string
	Email     string
	CreatedAt string `xorm:"created"`
	UpdatedAt string `xorm:"updated"`
	DeletedAt string `xorm:"deleted"`
}

func (table UserBasic) TableName() string {
	return "user_basic"
}
