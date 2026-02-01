package message

import "github.com/sashabaranov/go-openai"

type Message = openai.ChatCompletionMessage
type Role = string

const (
	User Role = "user"
	System Role = "system"
	Assistant Role = "assistant"
)

type MessageList struct {
	Msgs	[]Message
}

func NewMessageList() *MessageList {
	return &MessageList{
		Msgs: make([]Message, 0),
	}
}

func(m *MessageList) AddMessage(role Role, content string) {
	m.Msgs = append(m.Msgs, Message{
		Role:    role,
		Content: content,
	})
}

func(m *MessageList) AddToolCall(toolCallID string) {
	// TODO
	// m.Msgs = append(m.Msgs, Message{
	// 	Role:    string(Assistant),
	// 	Content: "",
	// 	ToolCallID: toolCallID,
	// })
}


