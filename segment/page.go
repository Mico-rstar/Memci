package segment

import (
	"memci/llm"
	"memci/message"
	"strings"
)

// page是对一组逻辑上连续的message的抽象
type Page struct {
	message.MessageList
	Name        string
	MaxToken    int     // -1 when not limit max token numbers
	Description string
	CompactModel *llm.CompactModel
}

func NewPage(name string, maxToken int, description string, compactModel *llm.CompactModel) *Page {
	return &Page{
		Name:         name,
		MaxToken:     maxToken,
		Description:  description,
		CompactModel: compactModel,
	}
}

func (p *Page) AddMessage(role message.Role, msg message.Message) {
	p.MessageList.AddMessage(role, msg.Content)
}

func (p *Page) ClearMessages() {
	p.MessageList.ClearMessages()
}

// CompactToOneMessage 使用 CompactModel 将所有消息压缩为一条消息
func (p *Page) CompactToOneMessage() error {
	if p.MessageList.Len() == 0 {
		return nil
	}

	if p.CompactModel == nil {
		return nil
	}

	// 调用压缩模型
	result := p.CompactModel.Process(p.MessageList)

	// 清空原消息列表
	p.ClearMessages()

	// 添加压缩后的消息
	p.AddMessage(result.Role, result)

	return nil
}

// MergeToOneMessage 将所有消息合并为一条消息（简单拼接）
func (p *Page) MergeToOneMessage() {
	if p.MessageList.Len() <= 1 {
		return
	}

	// 保存第一条消息的 role
	firstRole := p.MessageList.Msgs[0].Role

	// 收集所有消息内容
	var contents []string
	for _, msg := range p.MessageList.Msgs {
		contents = append(contents, msg.Content)
	}

	// 合并内容
	mergedContent := strings.Join(contents, "\n\n")

	// 清空原消息列表
	p.ClearMessages()

	// 添加合并后的消息
	p.MessageList.AddMessage(firstRole, mergedContent)
}