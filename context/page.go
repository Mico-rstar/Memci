package context

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// PageLifecycle Page的生命周期状态
type PageLifecycle int

const (
	// Active Page在上下文窗口内，对Agent可见
	Active PageLifecycle = iota
	// HotArchived Page不在上下文窗口内，但祖先Page是Active的，可通过展开恢复
	HotArchived
	// ColdArchived Page已脱离上下文系统，需通过外部记忆系统重新注入
	ColdArchived
)

// String 返回生命周期的字符串表示
func (l PageLifecycle) String() string {
	switch l {
	case Active:
		return "Active"
	case HotArchived:
		return "HotArchived"
	case ColdArchived:
		return "ColdArchived"
	default:
		return "Unknown"
	}
}

// PageVisibility Page的可见性状态（仅当Lifecycle为Active时有效）
type PageVisibility int

const (
	// Expanded Page处于展开状态
	// 对于ContentsPage：子节点可见
	// 对于DetailPage：detail内容可见
	Expanded PageVisibility = iota
	// Hidden Page处于隐藏状态
	// 对于ContentsPage：自身可见，但子节点不可见
	// 对于DetailPage：description可见，但detail不可见
	Hidden
)

// String 返回可见性的字符串表示
func (v PageVisibility) String() string {
	switch v {
	case Expanded:
		return "Expanded"
	case Hidden:
		return "Hidden"
	default:
		return "Unknown"
	}
}

// PageIndex Page的唯一索引标识
// 格式：对于单Segment为整数如 "0", "1"
//       对于多Segment使用前缀如 "sys-0", "usr-0"
type PageIndex string

// Page 页面接口，所有页面类型必须实现此接口
type Page interface {
	// 基本信息
	GetIndex() PageIndex           // 获取Page的唯一索引
	GetName() string               // 获取Page名称
	GetDescription() string        // 获取Page描述
	GetLifecycle() PageLifecycle   // 获取生命周期状态
	GetVisibility() PageVisibility // 获取可见性状态

	// 状态变更
	SetVisibility(visibility PageVisibility) error // 设置可见性（Expanded/Hidden）
	SetLifecycle(lifecycle PageLifecycle) error    // 设置生命周期状态

	// 父子关系
	GetParent() PageIndex               // 获取父Page的索引，根Page返回空字符串
	SetParent(parentIndex PageIndex) error // 设置父Page

	// 更新字段
	SetDescription(description string) error
	SetName(name string) error

	// 序列化/反序列化（用于PageStorage）
	Marshal() ([]byte, error)  // 序列化为字节
	Unmarshal(data []byte) error // 从字节反序列化
}

// DetailPage 叶子节点页面，存储原始的交互消息
type DetailPage struct {
	// 基本信息
	index       PageIndex      // 唯一索引
	name        string         // 页面名称
	description string         // 页面描述（摘要）
	lifecycle   PageLifecycle // 生命周期状态
	visibility  PageVisibility // 可见性状态

	// 层级关系
	parent PageIndex // 父Page索引，空字符串表示根

	// 核心内容
	detail string // 原始消息内容（合并后的完整对话）

	// 元数据
	createdAt   time.Time // 创建时间
	updatedAt   time.Time // 更新时间
}

// detailPageJSON 用于JSON序列化的内部结构
type detailPageJSON struct {
	Index        string         `json:"index"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Lifecycle    PageLifecycle  `json:"lifecycle"`
	Visibility   PageVisibility `json:"visibility"`
	Parent       string         `json:"parent"`
	Detail       string         `json:"detail"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

// NewDetailPage 创建新的DetailPage
func NewDetailPage(name, description, detail string, parentIndex PageIndex) (*DetailPage, error) {
	if name == "" {
		return nil, errors.New("name cannot be empty")
	}
	now := time.Now()
	return &DetailPage{
		name:        name,
		description: description,
		detail:      detail,
		parent:      parentIndex,
		lifecycle:   Active,
		visibility:  Hidden, // 默认只展示description
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// GetIndex 获取Page索引
func (p *DetailPage) GetIndex() PageIndex {
	return p.index
}

// GetName 获取Page名称
func (p *DetailPage) GetName() string {
	return p.name
}

// GetDescription 获取Page描述
func (p *DetailPage) GetDescription() string {
	return p.description
}

// GetDetail 获取原始消息内容
func (p *DetailPage) GetDetail() string {
	return p.detail
}

// SetDetail 更新原始消息内容
func (p *DetailPage) SetDetail(detail string) error {
	p.detail = detail
	p.updatedAt = time.Now()
	return nil
}

// SetVisibility 设置可见性
func (p *DetailPage) SetVisibility(visibility PageVisibility) error {
	p.visibility = visibility
	p.updatedAt = time.Now()
	return nil
}

// SetLifecycle 设置生命周期
func (p *DetailPage) SetLifecycle(lifecycle PageLifecycle) error {
	p.lifecycle = lifecycle
	p.updatedAt = time.Now()
	return nil
}

// GetParent 获取父Page索引
func (p *DetailPage) GetParent() PageIndex {
	return p.parent
}

// SetParent 设置父Page
func (p *DetailPage) SetParent(parentIndex PageIndex) error {
	p.parent = parentIndex
	p.updatedAt = time.Now()
	return nil
}

// GetLifecycle 获取生命周期状态
func (p *DetailPage) GetLifecycle() PageLifecycle {
	return p.lifecycle
}

// GetVisibility 获取可见性状态
func (p *DetailPage) GetVisibility() PageVisibility {
	return p.visibility
}

// SetDescription 设置描述
func (p *DetailPage) SetDescription(description string) error {
	p.description = description
	p.updatedAt = time.Now()
	return nil
}

// SetName 设置名称
func (p *DetailPage) SetName(name string) error {
	if name == "" {
		return errors.New("name cannot be empty")
	}
	p.name = name
	p.updatedAt = time.Now()
	return nil
}

// SetIndex 设置索引（通常由ContextSystem调用）
func (p *DetailPage) SetIndex(index PageIndex) {
	p.index = index
}

// Marshal 序列化
func (p *DetailPage) Marshal() ([]byte, error) {
	data := detailPageJSON{
		Index:        string(p.index),
		Name:         p.name,
		Description:  p.description,
		Lifecycle:    p.lifecycle,
		Visibility:   p.visibility,
		Parent:       string(p.parent),
		Detail:       p.detail,
		CreatedAt:    p.createdAt,
		UpdatedAt:    p.updatedAt,
	}
	return json.Marshal(data)
}

// Unmarshal 反序列化
func (p *DetailPage) Unmarshal(data []byte) error {
	var jsonData detailPageJSON
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return err
	}
	p.index = PageIndex(jsonData.Index)
	p.name = jsonData.Name
	p.description = jsonData.Description
	p.lifecycle = jsonData.Lifecycle
	p.visibility = jsonData.Visibility
	p.parent = PageIndex(jsonData.Parent)
	p.detail = jsonData.Detail
	p.createdAt = jsonData.CreatedAt
	p.updatedAt = jsonData.UpdatedAt
	return nil
}

// ContentsPage 目录节点页面，存储子页面的摘要索引
type ContentsPage struct {
	// 基本信息
	index       PageIndex      // 唯一索引
	name        string         // 页面名称
	description string         // 页面描述（目录摘要）
	lifecycle   PageLifecycle // 生命周期状态
	visibility  PageVisibility // 可见性状态

	// 层级关系
	parent PageIndex // 父Page索引，空字符串表示根

	// 子页面管理
	children []PageIndex // 子Page索引列表（有序）

	// 元数据
	createdAt time.Time // 创建时间
	updatedAt time.Time // 更新时间
}

// contentsPageJSON 用于JSON序列化的内部结构
type contentsPageJSON struct {
	Index       string         `json:"index"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Lifecycle   PageLifecycle  `json:"lifecycle"`
	Visibility  PageVisibility `json:"visibility"`
	Parent      string         `json:"parent"`
	Children    []string       `json:"children"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

// NewContentsPage 创建新的ContentsPage
func NewContentsPage(name, description string, parentIndex PageIndex) (*ContentsPage, error) {
	if name == "" {
		return nil, errors.New("name cannot be empty")
	}
	now := time.Now()
	return &ContentsPage{
		name:        name,
		description: description,
		parent:      parentIndex,
		lifecycle:   Active,
		visibility:  Hidden, // 默认只展示description
		children:    make([]PageIndex, 0),
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// GetIndex 获取Page索引
func (p *ContentsPage) GetIndex() PageIndex {
	return p.index
}

// GetName 获取Page名称
func (p *ContentsPage) GetName() string {
	return p.name
}

// GetDescription 获取Page描述
func (p *ContentsPage) GetDescription() string {
	return p.description
}

// SetVisibility 设置可见性
func (p *ContentsPage) SetVisibility(visibility PageVisibility) error {
	p.visibility = visibility
	p.updatedAt = time.Now()
	return nil
}

// SetLifecycle 设置生命周期
func (p *ContentsPage) SetLifecycle(lifecycle PageLifecycle) error {
	p.lifecycle = lifecycle
	p.updatedAt = time.Now()
	return nil
}

// GetParent 获取父Page索引
func (p *ContentsPage) GetParent() PageIndex {
	return p.parent
}

// SetParent 设置父Page
func (p *ContentsPage) SetParent(parentIndex PageIndex) error {
	p.parent = parentIndex
	p.updatedAt = time.Now()
	return nil
}

// GetLifecycle 获取生命周期状态
func (p *ContentsPage) GetLifecycle() PageLifecycle {
	return p.lifecycle
}

// GetVisibility 获取可见性状态
func (p *ContentsPage) GetVisibility() PageVisibility {
	return p.visibility
}

// SetDescription 设置描述
func (p *ContentsPage) SetDescription(description string) error {
	p.description = description
	p.updatedAt = time.Now()
	return nil
}

// SetName 设置名称
func (p *ContentsPage) SetName(name string) error {
	if name == "" {
		return errors.New("name cannot be empty")
	}
	p.name = name
	p.updatedAt = time.Now()
	return nil
}

// SetIndex 设置索引（通常由ContextSystem调用）
func (p *ContentsPage) SetIndex(index PageIndex) {
	p.index = index
}

// Marshal 序列化
func (p *ContentsPage) Marshal() ([]byte, error) {
	children := make([]string, len(p.children))
	for i, child := range p.children {
		children[i] = string(child)
	}
	data := contentsPageJSON{
		Index:       string(p.index),
		Name:        p.name,
		Description: p.description,
		Lifecycle:   p.lifecycle,
		Visibility:  p.visibility,
		Parent:      string(p.parent),
		Children:    children,
		CreatedAt:   p.createdAt,
		UpdatedAt:   p.updatedAt,
	}
	return json.Marshal(data)
}

// Unmarshal 反序列化
func (p *ContentsPage) Unmarshal(data []byte) error {
	var jsonData contentsPageJSON
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return err
	}
	p.index = PageIndex(jsonData.Index)
	p.name = jsonData.Name
	p.description = jsonData.Description
	p.lifecycle = jsonData.Lifecycle
	p.visibility = jsonData.Visibility
	p.parent = PageIndex(jsonData.Parent)
	p.children = make([]PageIndex, len(jsonData.Children))
	for i, child := range jsonData.Children {
		p.children[i] = PageIndex(child)
	}
	p.createdAt = jsonData.CreatedAt
	p.updatedAt = jsonData.UpdatedAt
	return nil
}

// AddChild 添加子页面
func (p *ContentsPage) AddChild(childIndex PageIndex) error {
	// 检查是否已存在
	for _, child := range p.children {
		if child == childIndex {
			return fmt.Errorf("child %s already exists", childIndex)
		}
	}
	p.children = append(p.children, childIndex)
	p.updatedAt = time.Now()
	return nil
}

// RemoveChild 移除子页面
func (p *ContentsPage) RemoveChild(childIndex PageIndex) error {
	for i, child := range p.children {
		if child == childIndex {
			p.children = append(p.children[:i], p.children[i+1:]...)
			p.updatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("child %s not found", childIndex)
}

// GetChildren 获取所有子页面索引
func (p *ContentsPage) GetChildren() []PageIndex {
	return p.children
}

// HasChild 检查是否包含指定子页面
func (p *ContentsPage) HasChild(childIndex PageIndex) bool {
	for _, child := range p.children {
		if child == childIndex {
			return true
		}
	}
	return false
}

// ChildCount 获取子页面数量
func (p *ContentsPage) ChildCount() int {
	return len(p.children)
}
