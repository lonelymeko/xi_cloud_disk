package models

type FileEventLog struct {
	Id                 int
	Identity           string
	RepositoryIdentity string
	UserIdentity       string
	EventType          string
	CreatedAt          string `xorm:"created"`
}

func (table FileEventLog) TableName() string {
	return "file_event_log"
}
