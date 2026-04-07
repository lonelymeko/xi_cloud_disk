package models

// AgentChatSession 持久化用户 Agent 会话元数据。
type AgentChatSession struct {
	Id                 int64  `xorm:"'id' pk autoincr"`
	SessionID          string `xorm:"'session_id' varchar(64) notnull unique(session_user) index"`
	UserIdentity       string `xorm:"'user_identity' varchar(64) notnull unique(session_user) index"`
	Title              string `xorm:"'title' varchar(255) notnull default('New Chat')"`
	PendingInterruptID string `xorm:"'pending_interrupt_id' varchar(128) null"`
	CreatedAt          string `xorm:"'created_at' created"`
	UpdatedAt          string `xorm:"'updated_at' updated"`
}

func (table AgentChatSession) TableName() string {
	return "agent_chat_session"
}
