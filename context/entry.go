package context

import (
	"fmt"
	"memci/llm"
	"memci/message"
	"time"
)

// Entry 上下文系统最小单位，对应 MessageList 中的一个 MessageNode
// Entry 只持有节点引用，消息数据通过 Node 访问
type Entry struct {
	ID        string
	Node      *message.MessageNode // 指向 MessageList 中的节点
	Timestamp time.Time
	Metadata  map[string]string
}

// EntrySummary Entry 的摘要信息（用于 Contents Page）
type EntrySummary struct {
	EntryID string // 原始 Entry ID
	Summary string // 摘要内容
}

// NewEntry 创建新的 Entry（同时创建 MessageNode）
func NewEntry(role message.Role, content message.Content) *Entry {
	node := message.CreateNode(role, content)
	return &Entry{
		ID:        generateEntryID(),
		Node:      node,
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}
}

// NewEntryWithID 创建带有指定 ID 的 Entry（用于从存储恢复）
func NewEntryWithID(id string, role message.Role, content message.Content, timestamp time.Time) *Entry {
	node := message.CreateNode(role, content)
	return &Entry{
		ID:        id,
		Node:      node,
		Timestamp: timestamp,
		Metadata:  make(map[string]string),
	}
}

// NewEntryFromMessage 从 Message 创建 Entry
func NewEntryFromMessage(msg message.Message) *Entry {
	node := message.CreateNode(msg.Role, msg.Content)
	return &Entry{
		ID:        generateEntryID(),
		Node:      node,
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}
}

// NewEntryFromNode 从 MessageNode 创建 Entry
func NewEntryFromNode(node *message.MessageNode) *Entry {
	return &Entry{
		ID:        generateEntryID(),
		Node:      node,
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}
}

// Role 返回 Entry 的角色
func (e *Entry) Role() message.Role {
	return e.Node.GetMsg().Role
}

// Content 返回 Entry 的内容
func (e *Entry) Content() message.Content {
	return e.Node.GetMsg().Content
}

// ToMessage 将 Entry 转换为 Message
func (e *Entry) ToMessage() message.Message {
	return e.Node.GetMsg()
}

// String 返回 Entry 的字符串表示
func (e *Entry) String() string {
	return fmt.Sprintf("[%s] %s", e.Role(), e.Content().String())
}

// AddToolCall 添加工具调用
func (e *Entry) AddToolCall(toolCall message.ToolCall) {
	msg := e.Node.GetMsg()
	msg.ToolCalls = append(msg.ToolCalls, toolCall)
	e.Node.SetMsg(msg)
}

// SetMetadata 设置元数据
func (e *Entry) SetMetadata(key, value string) {
	e.Metadata[key] = value
}

// GetMetadata 获取元数据
func (e *Entry) GetMetadata(key string) (string, bool) {
	value, ok := e.Metadata[key]
	return value, ok
}

// Summarize 生成 Entry 的摘要（使用压缩模型）
func (e *Entry) Summarize(model *llm.CompactModel) (*EntrySummary, error) {
	if model == nil {
		// 无模型时返回简单摘要
		return &EntrySummary{
			EntryID: e.ID,
			Summary:  e.Content().String(),
		}, nil
	}

	// 使用压缩模型生成摘要
	// 构造临时消息列表
	msgList := message.NewMessageList().AddMessage(e.Role(), e.Content().String())

	result, err := model.Process(*msgList)
	if err != nil {
		return nil, fmt.Errorf("failed to summarize entry: %w", err)
	}

	return &EntrySummary{
		EntryID: e.ID,
		Summary:  result.Content.String(),
	}, nil
}

// entryIDCounter 用于生成唯一 ID
var entryIDCounter int64

// generateEntryID 生成唯一的 Entry ID
func generateEntryID() string {
	entryIDCounter++
	return fmt.Sprintf("entry-%d-%d", time.Now().Unix(), entryIDCounter)
}

// EntryList Entry 列表
type EntryList struct {
	Entries []*Entry
}

// NewEntryList 创建新的 EntryList
func NewEntryList() *EntryList {
	return &EntryList{
		Entries: make([]*Entry, 0),
	}
}

// AddEntry 添加 Entry
func (el *EntryList) AddEntry(entry *Entry) *EntryList {
	el.Entries = append(el.Entries, entry)
	return el
}

// AddEntryFromMessage 从 Message 添加 Entry
func (el *EntryList) AddEntryFromMessage(msg message.Message) *EntryList {
	entry := NewEntryFromMessage(msg)
	el.Entries = append(el.Entries, entry)
	return el
}

// Len 返回 Entry 数量
func (el *EntryList) Len() int {
	return len(el.Entries)
}

// Get 获取指定索引的 Entry
func (el *EntryList) Get(index int) (*Entry, bool) {
	if index < 0 || index >= len(el.Entries) {
		return nil, false
	}
	return el.Entries[index], true
}


// Clear 清空列表
func (el *EntryList) Clear() {
	el.Entries = make([]*Entry, 0)
}
