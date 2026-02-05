package models

// FileEventLog 对应 file_event_log 表（文件事件日志表）。
type FileEventLog struct {
	Id                 int
	Identity           string
	RepositoryIdentity string
	UserIdentity       string
	EventType          string
	CreatedAt          string `xorm:"created"`
}

// TableName 指定数据表名。
func (table FileEventLog) TableName() string {
	return "file_event_log"
}
