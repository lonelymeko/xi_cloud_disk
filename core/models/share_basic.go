package models

// ShareBasic 对应 share_basic 表（文件分享表）
type ShareBasic struct {
	Id                 int
	Identity           string
	UserIdentity       string
	RepositoryIdentity string
	ExpiredTime        int
	CreatedAt          string `xorm:"created"`
	UpdatedAt          string `xorm:"updated"`
	DeletedAt          string `xorm:"deleted"`
}

// TableName 指定数据表名
func (table ShareBasic) TableName() string {
	return "share_basic"
}
