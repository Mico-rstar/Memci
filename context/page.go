package context

import (
	"memci/llm"
	"memci/message"
	"strings"
)

// Page 存储一组相关的 Entry（如一次对话）
type Page struct {
	Entries      []*Entry
	Index        PageIndex
	Name         string
	MaxToken     int              // -1 表示不限制最大 token 数
	Description  string
	CompactModel *llm.CompactModel
}

// NewPage 创建新的 Page
func NewPage(index PageIndex, name string, maxToken int, description string, compactModel *llm.CompactModel) *Page {
	return &Page{
		Entries:      make([]*Entry, 0),
		Index:        index,
		Name:         name,
		MaxToken:     maxToken,
		Description:  description,
		CompactModel: compactModel,
	}
}

// AddEntry 添加 Entry 到 Page
func (p *Page) AddEntry(entry *Entry) {
	p.Entries = append(p.Entries, entry)
}

// AddEntryFromRoleAndContent 从角色和内容创建并添加 Entry（便利方法）
func (p *Page) AddEntryFromRoleAndContent(role message.Role, content message.Content) {
	entry := NewEntry(role, content)
	p.AddEntry(entry)
}

// Clear 清空所有 Entry
func (p *Page) Clear() {
	p.Entries = make([]*Entry, 0)
}

// GetEntries 获取所有 Entry
func (p *Page) GetEntries() []*Entry {
	return p.Entries
}

// Len 返回 Entry 数量
func (p *Page) Len() int {
	return len(p.Entries)
}

// CompactToOneMessage 使用 CompactModel 将所有消息压缩为一条 Entry
func (p *Page) CompactToOneMessage() error {
	if p.Len() == 0 {
		return nil
	}

	if p.CompactModel == nil {
		return nil
	}

	// 将 Entries 的 MessageNode 连接成 MessageList 进行压缩
	msgList := p.ToMessageList()

	result, err := p.CompactModel.Process(*msgList)
	if err != nil {
		return err
	}

	// 清空原 Entry 列表
	p.Clear()

	// 添加压缩后的 Entry
	p.AddEntry(NewEntryFromMessage(result))

	return nil
}

// MergeToOneMessage 将所有 Entry 合并为一条 Entry（简单拼接）
func (p *Page) MergeToOneMessage() {
	if p.Len() <= 1 {
		return
	}

	// 保存第一条 Entry 的 role
	firstRole := p.Entries[0].Role()

	// 收集所有 Entry 的内容
	var contents []string
	for _, entry := range p.Entries {
		contents = append(contents, entry.Content().String())
	}

	// 合并内容
	mergedContent := strings.Join(contents, "\n\n")

	// 清空原 Entry 列表
	p.Clear()

	// 添加合并后的 Entry
	p.AddEntry(NewEntry(firstRole, message.NewContentString(mergedContent)))
}

// Summarize 生成 Page 的摘要
func (p *Page) Summarize() string {
	if p.Description != "" {
		return p.Description
	}

	// 简单摘要：收集所有 Entry 的内容预览
	var parts []string
	for _, entry := range p.Entries {
		preview := entry.Content().String()
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		parts = append(parts, entry.Role()+": "+preview)
	}

	return strings.Join(parts, " | ")
}

// ToMessageList 将 Page 转换为 MessageList（边界转换方法）
func (p *Page) ToMessageList() *message.MessageList {
	msgList := message.NewMessageList()
	for _, entry := range p.Entries {
		msgList.AddNode(entry.Node)
	}
	return msgList
}

// EstimateTokens 估算 Page 的 token 数
func (p *Page) EstimateTokens() int {
	totalChars := 0
	for _, entry := range p.Entries {
		totalChars += len(entry.Content().String())
	}
	// 粗略估算：token 数 ≈ 字符数 / 4
	return totalChars / 4
}
