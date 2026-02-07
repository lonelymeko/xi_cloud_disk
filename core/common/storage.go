package common

const (
	// StatusActive 表示文件处于可用状态。
	StatusActive = "active"
	// StatusDeleted 表示文件已被标记删除。
	StatusDeleted = "deleted"
	// StatusPurging 表示文件进入清理中状态。
	StatusPurging = "purging"
	// StatusPurged 表示文件已完成清理。
	StatusPurged = "purged"
)

const (
	// EventDelete 表示删除事件。
	EventDelete = "delete"
	// EventRestore 表示恢复事件。
	EventRestore = "restore"
	// EventPurge 表示清理事件。
	EventPurge = "purge"
)
