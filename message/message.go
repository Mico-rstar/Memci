package message

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Role = string

const (
	User      Role = "user"
	System    Role = "system"
	Assistant Role = "assistant"
	Developer Role = "developer"
	Tool      Role = "tool"
)

// ToolCall represents a tool call in a message
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall represents a function call
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// CacheControl represents cache control settings for content caching
type CacheControl struct {
	Type string `json:"type"` // "ephemeral" is currently the only supported type
}

// ContentPart represents a part of multimodal content
type ContentPart struct {
	Type         string        `json:"type"`
	Text         string        `json:"text,omitempty"`
	ImageURL     *ImageURL     `json:"image_url,omitempty"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// ImageURL represents an image URL
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "auto", "low", "high"
}

// Content represents message content that can be either a string or an array of ContentPart
type Content struct {
	textValue  string
	Parts      []ContentPart
	isString   bool
}

// NewContentString creates a Content from a string
func NewContentString(s string) Content {
	return Content{textValue: s, isString: true}
}

func NewCachedContentString(s string) Content {
	return Content{
		textValue: s,
		Parts: []ContentPart{{
			Type:         "text",
			Text:         s,
			CacheControl: NewEphemeralCacheControl(),
		}},
		isString: true,
	}
}

// NewContentParts creates a Content from ContentPart array
func NewContentParts(parts []ContentPart) Content {
	return Content{Parts: parts, isString: false}
}

// NewEphemeralCacheControl creates a cache control for ephemeral caching
func NewEphemeralCacheControl() *CacheControl {
	return &CacheControl{Type: "ephemeral"}
}

// NewTextContentPart creates a text content part
func NewTextContentPart(text string) ContentPart {
	return ContentPart{
		Type: "text",
		Text: text,
	}
}

// NewCachedTextContentPart creates a text content part with cache control
func NewCachedTextContentPart(text string) ContentPart {
	return ContentPart{
		Type:         "text",
		Text:         text,
		CacheControl: NewEphemeralCacheControl(),
	}
}

// NewImageContentPart creates an image content part
func NewImageContentPart(url string, detail string) ContentPart {
	return ContentPart{
		Type: "image_url",
		ImageURL: &ImageURL{
			URL:    url,
			Detail: detail,
		},
	}
}

// MarshalJSON implements json.Marshaler
func (c Content) MarshalJSON() ([]byte, error) {
	if c.isString {
		return json.Marshal(c.textValue)
	}
	return json.Marshal(c.Parts)
}

// UnmarshalJSON implements json.Unmarshaler
func (c *Content) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		c.textValue = s
		c.isString = true
		return nil
	}

	// Try to unmarshal as array
	var parts []ContentPart
	if err := json.Unmarshal(data, &parts); err == nil {
		c.Parts = parts
		c.isString = false
		return nil
	}

	return fmt.Errorf("content must be string or array of content parts")
}

// IsString returns true if content is a string
func (c Content) IsString() bool {
	return c.isString
}

// GetString returns the string content (valid only if IsString() is true)
func (c Content) GetString() string {
	return c.textValue
}

// Parts returns the content parts (valid only if IsString() is false)
func (c Content) GetParts() []ContentPart {
	return c.Parts
}

// String returns a string representation of the content
// For string content, returns the string directly
// For array content, concatenates all text parts
func (c Content) String() string {
	if c.isString {
		return c.textValue
	}
	var builder strings.Builder
	for _, part := range c.Parts {
		if part.Text != "" {
			builder.WriteString(part.Text)
		}
	}
	return builder.String()
}

// Message represents a chat completion message
type Message struct {
	Role      string     `json:"role"`
	Content   Content    `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// MessageNode represents a node in the linked list
type MessageNode struct {
	msg  Message
	next *MessageNode
	prev *MessageNode
}

// GetMsg returns the message in this node
func (n *MessageNode) GetMsg() Message {
	return n.msg
}

// SetMsg sets the message in this node
func (n *MessageNode) SetMsg(msg Message) {
	n.msg = msg
}

// MessageList is a doubly linked list of messages
type MessageList struct {
	head *MessageNode
	tail *MessageNode
	len  int
}

func NewMessageList() *MessageList {
	return &MessageList{
		head: nil,
		tail: nil,
		len:  0,
	}
}

func (m *MessageList) AddMessages(msgs ...Message) *MessageList {
	for _, msg := range msgs {
		m.AddMessageContent(msg.Role, msg.Content)
	}
	return m
}

func (m *MessageList) AddMessage(role Role, content string) *MessageList {
	return m.AddMessageContent(role, NewContentString(content))
}

func (m *MessageList) AddCachedMessage(role Role, content string) *MessageList {
	return m.AddMessageContent(role, NewCachedContentString(content))
}

// AddMessageContent adds a message with Content type
func (m *MessageList) AddMessageContent(role Role, content Content) *MessageList {
	node := &MessageNode{
		msg: Message{
			Role:    role,
			Content: content,
		},
		next: nil,
		prev: nil,
	}

	if m.tail == nil {
		m.head = node
		m.tail = node
	} else {
		m.tail.next = node
		node.prev = m.tail
		m.tail = node
	}
	m.len++
	return m
}

// AddMessageList adds all messages from another MessageList
func (m *MessageList) AddMessageList(other *MessageList) *MessageList {
	for node := other.head; node != nil; node = node.next {
		m.AddMessageContent(node.msg.Role, node.msg.Content)
	}
	return m
}

// AddNode adds an existing MessageNode to the list
func (m *MessageList) AddNode(node *MessageNode) *MessageList {
	node.next = nil
	node.prev = nil

	if m.tail == nil {
		m.head = node
		m.tail = node
	} else {
		m.tail.next = node
		node.prev = m.tail
		m.tail = node
	}
	m.len++
	return m
}

// AddNodeWithoutReset adds an existing MessageNode to the list without resetting next/prev
// This is used for nodes that are already part of a linked list structure
func (m *MessageList) AddNodeWithoutReset(node *MessageNode) *MessageList {
	if m.tail == nil {
		m.head = node
		m.tail = node
	} else {
		m.tail.next = node
		node.prev = m.tail
		m.tail = node
	}
	m.len++
	return m
}

// RemoveNode removes a specific node from the list (O(1) operation)
func (m *MessageList) RemoveNode(node *MessageNode) {
	if node == nil {
		return
	}

	// Update previous node
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		// Node is head
		m.head = node.next
	}

	// Update next node
	if node.next != nil {
		node.next.prev = node.prev
	} else {
		// Node is tail
		m.tail = node.prev
	}

	m.len--
}

// SetHead sets the head of the MessageList (for internal use after node removal)
func (m *MessageList) SetHead(head *MessageNode) {
	m.head = head
	if head == nil {
		m.tail = nil
		m.len = 0
	}
}

// GetHead returns the head node of the MessageList
func (m *MessageList) GetHead() *MessageNode {
	return m.head
}

// GetTail returns the tail node of the MessageList
func (m *MessageList) GetTail() *MessageNode {
	return m.tail
}

// CreateNode creates a new MessageNode (without adding to list)
func CreateNode(role Role, content Content) *MessageNode {
	return &MessageNode{
		msg: Message{
			Role:    role,
			Content: content,
		},
		next: nil,
		prev: nil,
	}
}

// GetNode returns the head node of the list
func (m *MessageList) GetNode() *MessageNode {
	return m.head
}

// GetNext returns the next node
func (n *MessageNode) GetNext() *MessageNode {
	return n.next
}

// GetPrev returns the previous node
func (n *MessageNode) GetPrev() *MessageNode {
	return n.prev
}

// SetNext sets the next node (for rebuilding linked lists)
func (n *MessageNode) SetNext(next *MessageNode) {
	n.next = next
}

// SetPrev sets the previous node (for rebuilding linked lists)
func (n *MessageNode) SetPrev(prev *MessageNode) {
	n.prev = prev
}

// MarshalJSON implements json.Marshaler
// 序列化时只保存 msg 数据，不保存链表指针 (next/prev)
func (n *MessageNode) MarshalJSON() ([]byte, error) {
	return json.Marshal(n.msg)
}

// UnmarshalJSON implements json.Unmarshaler
// 反序列化时创建新节点，next/prev 为 nil
func (n *MessageNode) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &n.msg)
}

func (m *MessageList) ClearMessages() {
	m.head = nil
	m.tail = nil
	m.len = 0
}

func (m *MessageList) Len() int {
	return m.len
}

// ForEach 遍历每个 Message
func (m *MessageList) ForEach(fn func(msg Message)) {
	for node := m.head; node != nil; node = node.next {
		fn(node.msg)
	}
}

// ForEachNode 遍历每个 MessageNode
func (m *MessageList) ForEachNode(fn func(node *MessageNode)) {
	for node := m.head; node != nil; node = node.next {
		fn(node)
	}
}

// Range 遍历每个 Message，返回 false 时停止遍历
func (m *MessageList) Range(fn func(msg Message) bool) {
	for node := m.head; node != nil; node = node.next {
		if !fn(node.msg) {
			break
		}
	}
}

// RangeNode 遍历每个 MessageNode，返回 false 时停止遍历
func (m *MessageList) RangeNode(fn func(node *MessageNode) bool) {
	for node := m.head; node != nil; node = node.next {
		if !fn(node) {
			break
		}
	}
}

// ToSlice 将 MessageList 转换为 Message 切片
func (m *MessageList) ToSlice() []Message {
	result := make([]Message, 0, m.len)
	for node := m.head; node != nil; node = node.next {
		result = append(result, node.msg)
	}
	return result
}

// Msgs returns all messages as a slice (for backward compatibility)
// func (m *MessageList) Msgs() []Message {
// 	result := make([]Message, 0, m.len)
// 	for node := m.head; node != nil; node = node.next {
// 		result = append(result, node.msg)
// 	}
// 	return result
// }

// MarshalJSON implements json.Marshaler - 直接序列化链表，避免创建切片副本
func (m *MessageList) MarshalJSON() ([]byte, error) {
	buf := []byte{'['}

	for node := m.head; node != nil; node = node.next {
		if node != m.head {
			buf = append(buf, ',')
		}
		msgJSON, err := json.Marshal(node.msg)
		if err != nil {
			return nil, err
		}
		buf = append(buf, msgJSON...)
	}

	buf = append(buf, ']')
	return buf, nil
}

// UnmarshalJSON implements json.Unmarshaler
func (m *MessageList) UnmarshalJSON(data []byte) error {
	var msgs []Message
	if err := json.Unmarshal(data, &msgs); err != nil {
		return err
	}

	m.ClearMessages()
	for _, msg := range msgs {
		m.AddMessageContent(msg.Role, msg.Content)
	}
	return nil
}
