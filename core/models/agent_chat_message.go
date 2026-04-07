package models

// AgentChatMessage 持久化用户 Agent 会话消息。
type AgentChatMessage struct {
	Id           int64  `xorm:"'id' pk autoincr"`
	SessionID    string `xorm:"'session_id' varchar(64) notnull index"`
	UserIdentity string `xorm:"'user_identity' varchar(64) notnull index"`
	Role         string `xorm:"'role' varchar(32) notnull"`
	Content      string `xorm:"'content' text"`
	MessageJSON  string `xorm:"'message_json' mediumtext notnull"`
	CreatedAt    string `xorm:"'created_at' created"`
}

func (table AgentChatMessage) TableName() string {
	return "agent_chat_message"
}
