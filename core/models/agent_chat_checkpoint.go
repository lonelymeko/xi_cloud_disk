package models

// AgentChatCheckpoint 持久化 Agent checkpoint，用于中断/恢复。
type AgentChatCheckpoint struct {
	Id            int64  `xorm:"'id' pk autoincr"`
	CheckpointKey string `xorm:"'checkpoint_key' varchar(255) notnull unique index"`
	SessionID     string `xorm:"'session_id' varchar(64) notnull index"`
	UserIdentity  string `xorm:"'user_identity' varchar(64) notnull index"`
	Payload       []byte `xorm:"'payload' mediumblob notnull"`
	UpdatedAt     string `xorm:"'updated_at' updated"`
}

func (table AgentChatCheckpoint) TableName() string {
	return "agent_chat_checkpoint"
}
