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
		Content: NewContentString(content),
	})
	return m
}

func (m *MessageList) AddCachedMessage(role Role, content string) *MessageList {
	m.Msgs = append(m.Msgs, Message{
		Role:    role,
		Content: NewCachedContentString(content),
	})
	return m
}

func (m *MessageList) ClearMessages() {
	m.Msgs = make([]Message, 0)
}

func (m *MessageList) Len() int {
	return len(m.Msgs)
}
