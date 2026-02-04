package context

import (
	"encoding/json"
	"memci/llm"
	"memci/message"
	"strings"
)

// Page 存储一组相关的 Entry（如一次对话）
// 在视图模式中，Page 作为 Chapter.MessageList 中一段连续节点的视图
type Page struct {
	chapter     *BaseChapter        // 所属 Chapter
	head        *message.MessageNode // 视图的首节点
	tail        *message.MessageNode // 视图的尾节点
	Index       PageIndex
	Name        string
	MaxToken    int  // -1 表示不限制最大 token 数
	Description string
	CompactModel *llm.CompactModel
}

// NewPage 创建新的 Page
func NewPage(index PageIndex, name string, maxToken int, description string, compactModel *llm.CompactModel) *Page {
	return &Page{
		Index:        index,
		Name:         name,
		MaxToken:     maxToken,
		Description:  description,
		CompactModel: compactModel,
		head:         nil,
		tail:         nil,
	}
}

// AddEntry 添加 Entry 到 Page（在添加到 Chapter 前）
// Entry 的 Node 会被添加到 Page 的内部链表中
// 当 Page 被添加到 Chapter 时，所有 Node 会被连接到 Chapter.MessageList
func (p *Page) AddEntry(entry *Entry) {
	node := entry.Node

	// 如果是第一个 Entry，设置 head
	if p.head == nil {
		p.head = node
		p.tail = node
		return
	}

	// 否则添加到 tail 后面，并更新 tail
	p.tail.SetNext(node)
	node.SetPrev(p.tail)
	p.tail = node
}

// Detach 从 MessageList 中分离此 Page 的所有节点，并保持前后 Page 的连接
func (p *Page) Detach() {
	if p.chapter == nil || p.head == nil || p.tail == nil {
		return
	}

	// 调用 Chapter 的 RemovePageRange 方法
	p.chapter.RemovePageRange(p)
}

// Head 返回 Page 的首节点
func (p *Page) Head() *message.MessageNode {
	return p.head
}

// Tail 返回 Page 的尾节点
func (p *Page) Tail() *message.MessageNode {
	return p.tail
}

// GetEntries 获取所有 Entry（懒加载：从 head 遍历到 tail）
func (p *Page) GetEntries() []*Entry {
	if p.head == nil {
		return []*Entry{}
	}

	var entries []*Entry
	for node := p.head; node != nil; node = node.GetNext() {
		entries = append(entries, &Entry{
			Node: node,
		})
		if node == p.tail {
			break
		}
	}
	return entries
}

// Len 返回 Entry 数量（计算链表长度）
func (p *Page) Len() int {
	if p.head == nil {
		return 0
	}

	count := 0
	for node := p.head; node != nil; node = node.GetNext() {
		count++
		if node == p.tail {
			break
		}
	}
	return count
}

// Summarize 生成 Page 的摘要
func (p *Page) Summarize() string {
	if p.Description != "" {
		return p.Description
	}

	// 遍历节点生成摘要
	var parts []string
	for node := p.head; node != nil; node = node.GetNext() {
		msg := node.GetMsg()
		preview := msg.Content.String()
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		parts = append(parts, msg.Role+": "+preview)
		if node == p.tail {
			break
		}
	}

	return strings.Join(parts, " | ")
}

// ToMessageList 将 Page 转换为 MessageList（视图方法）
func (p *Page) ToMessageList() *message.MessageList {
	if p.head == nil {
		return message.NewMessageList()
	}

	msgList := message.NewMessageList()
	for node := p.head; node != nil; node = node.GetNext() {
		msgList.AddNodeWithoutReset(node)
		if node == p.tail {
			break
		}
	}
	return msgList
}

// EstimateTokens 估算 Page 的 token 数
func (p *Page) EstimateTokens() int {
	if p.head == nil {
		return 0
	}

	totalChars := 0
	for node := p.head; node != nil; node = node.GetNext() {
		totalChars += len(node.GetMsg().Content.String())
		if node == p.tail {
			break
		}
	}
	// 粗略估算：token 数 ≈ 字符数 / 4
	return totalChars / 4
}

// jsonPage 用于 JSON 序列化的临时结构
type jsonPage struct {
	Entries      []*Entry
	Index        PageIndex
	Name         string
	MaxToken     int
	Description  string
	CompactModel *llm.CompactModel
}

// MarshalJSON 实现 json.Marshaler 接口
// 序列化时将 head/tail 转换为 Entries 列表
func (p *Page) MarshalJSON() ([]byte, error) {
	// 创建临时结构用于序列化
	jp := jsonPage{
		Entries:      p.GetEntries(),
		Index:        p.Index,
		Name:         p.Name,
		MaxToken:     p.MaxToken,
		Description:  p.Description,
		CompactModel: p.CompactModel,
	}
	return json.Marshal(jp)
}

// UnmarshalJSON 实现 json.Unmarshaler 接口
// 反序列化时从 Entries 重建 head/tail 和链表连接
func (p *Page) UnmarshalJSON(data []byte) error {
	var jp jsonPage
	if err := json.Unmarshal(data, &jp); err != nil {
		return err
	}

	// 复制基本字段
	p.Index = jp.Index
	p.Name = jp.Name
	p.MaxToken = jp.MaxToken
	p.Description = jp.Description
	p.CompactModel = jp.CompactModel

	// 从 Entries 重建 head/tail 和链表连接
	if len(jp.Entries) > 0 {
		p.head = jp.Entries[0].Node
		p.tail = jp.Entries[len(jp.Entries)-1].Node

		// 重建 Node 的链表连接
		for i := 0; i < len(jp.Entries); i++ {
			currentNode := jp.Entries[i].Node

			// 设置 next 指针
			if i < len(jp.Entries)-1 {
				currentNode.SetNext(jp.Entries[i+1].Node)
			} else {
				currentNode.SetNext(nil)
			}

			// 设置 prev 指针
			if i > 0 {
				currentNode.SetPrev(jp.Entries[i-1].Node)
			} else {
				currentNode.SetPrev(nil)
			}
		}
	}

	return nil
}
