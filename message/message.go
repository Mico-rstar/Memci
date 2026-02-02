package message

import "github.com/sashabaranov/go-openai"

type Message = openai.ChatCompletionMessage
type Role = string

const (
	User      Role = "user"
	System    Role = "system"
	Assistant Role = "assistant"
	Developer Role = "developer"
)

type MessageList struct {
	Msgs []Message
}

func NewMessageList() *MessageList {
	return &MessageList{
		Msgs: make([]Message, 0),
	}
}

func (m *MessageList) AddMessages(msgs ...Message) *MessageList {
	m.Msgs = append(m.Msgs, msgs...)
	return m
}

func (m *MessageList) AddMessage(role Role, content string) *MessageList {
	m.Msgs = append(m.Msgs, Message{
		Role:    role,
		Content: content,
	})
	return m
}

func (m *MessageList) ClearMessages() {
	m.Msgs = make([]Message, 0)
}

func (m *MessageList) Len() int {
	return len(m.Msgs)
}
