# Page 设计文档

## 概述

Page 是上下文管理系统的核心抽象，类比文件系统中的"文件"概念（广义文件，包含文件和目录）。

- **Page接口**：统一的页面抽象
- **DetailPage**：叶子节点，存储原始消息内容
- **ContentsPage**：目录节点，存储子页面的摘要索引

## Page 接口定义

```go
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
    SetVisibility(visibility PageVisibility) error  // 设置可见性（Expanded/Hidden）
    SetLifecycle(lifecycle PageLifecycle) error     // 设置生命周期状态

    // 父子关系
    GetParent() PageIndex          // 获取父Page的索引，根Page返回空字符串
    SetParent(parentIndex PageIndex) error // 设置父Page

    // 序列化/反序列化（用于Storage）
    Marshal() ([]byte, error)      // 序列化为字节
    Unmarshal(data []byte) error   // 从字节反序列化
}
```

## DetailPage 结构体

```go
// DetailPage 叶子节点页面，存储原始的交互消息
type DetailPage struct {
    // 基本信息
    index       PageIndex       // 唯一索引
    name        string          // 页面名称
    description string          // 页面描述（摘要）
    lifecycle   PageLifecycle   // 生命周期状态
    visibility  PageVisibility  // 可见性状态

    // 层级关系
    parent      PageIndex       // 父Page索引，空字符串表示根

    // 核心内容
    detail      string          // 原始消息内容（合并后的完整对话）
    // detail字段存储：用户消息 + 多轮assistant/tool消息的完整内容

    // 元数据
    createdAt  time.Time        // 创建时间
    updatedAt  time.Time        // 更新时间
    messageCount int            // 包含的消息数量
}

// DetailPage 核心方法

// NewDetailPage 创建新的DetailPage
func NewDetailPage(name, description, detail string, parentIndex PageIndex) (*DetailPage, error)

// GetIndex 获取Page索引
func (p *DetailPage) GetIndex() PageIndex

// GetName 获取Page名称
func (p *DetailPage) GetName() string

// GetDescription 获取Page描述
func (p *DetailPage) GetDescription() string

// GetDetail 获取原始消息内容
func (p *DetailPage) GetDetail() string

// SetDetail 更新原始消息内容
func (p *DetailPage) SetDetail(detail string) error

// SetVisibility 设置可见性
func (p *DetailPage) SetVisibility(visibility PageVisibility) error

// SetLifecycle 设置生命周期
func (p *DetailPage) SetLifecycle(lifecycle PageLifecycle) error

// GetParent 获取父Page索引
func (p *DetailPage) GetParent() PageIndex

// SetParent 设置父Page
func (p *DetailPage) SetParent(parentIndex PageIndex) error

// GetLifecycle 获取生命周期状态
func (p *DetailPage) GetLifecycle() PageLifecycle

// GetVisibility 获取可见性状态
func (p *DetailPage) GetVisibility() PageVisibility

// Marshal 序列化
func (p *DetailPage) Marshal() ([]byte, error)

// Unmarshal 反序列化
func (p *DetailPage) Unmarshal(data []byte) error

// SetDescription 设置描述
func (p *DetailPage) SetDescription(description string) error

// SetName 设置名称
func (p *DetailPage) SetName(name string) error
```

## ContentsPage 结构体

```go
// ContentsPage 目录节点页面，存储子页面的摘要索引
type ContentsPage struct {
    // 基本信息
    index       PageIndex       // 唯一索引
    name        string          // 页面名称
    description string          // 页面描述（目录摘要）
    lifecycle   PageLifecycle   // 生命周期状态
    visibility  PageVisibility  // 可见性状态

    // 层级关系
    parent      PageIndex       // 父Page索引，空字符串表示根

    // 子页面管理
    children    []PageIndex     // 子Page索引列表（有序）
    // 注意：这里存储的是索引，不是Page对象
    // 实际的Page对象由ContextSystem统一管理

    // 元数据
    createdAt  time.Time        // 创建时间
    updatedAt  time.Time        // 更新时间
}

// ContentsPage 核心方法

// NewContentsPage 创建新的ContentsPage
func NewContentsPage(name, description string, parentIndex PageIndex) (*ContentsPage, error)

// GetIndex 获取Page索引
func (p *ContentsPage) GetIndex() PageIndex

// GetName 获取Page名称
func (p *ContentsPage) GetName() string

// GetDescription 获取Page描述
func (p *ContentsPage) GetDescription() string

// SetVisibility 设置可见性
func (p *ContentsPage) SetVisibility(visibility PageVisibility) error

// SetLifecycle 设置生命周期
func (p *ContentsPage) SetLifecycle(lifecycle PageLifecycle) error

// GetParent 获取父Page索引
func (p *ContentsPage) GetParent() PageIndex

// SetParent 设置父Page
func (p *ContentsPage) SetParent(parentIndex PageIndex) error

// GetLifecycle 获取生命周期状态
func (p *ContentsPage) GetLifecycle() PageLifecycle

// GetVisibility 获取可见性状态
func (p *ContentsPage) GetVisibility() PageVisibility

// Marshal 序列化
func (p *ContentsPage) Marshal() ([]byte, error)

// Unmarshal 反序列化
func (p *ContentsPage) Unmarshal(data []byte) error

// SetDescription 更新描述
func (p *ContentsPage) SetDescription(description string) error

// SetName 更新名称
func (p *ContentsPage) SetName(name string) error

// === 子页面管理方法 ===

// AddChild 添加子页面
func (p *ContentsPage) AddChild(childIndex PageIndex) error

// RemoveChild 移除子页面
func (p *ContentsPage) RemoveChild(childIndex PageIndex) error

// GetChildren 获取所有子页面索引
func (p *ContentsPage) GetChildren() []PageIndex

// HasChild 检查是否包含指定子页面
func (p *ContentsPage) HasChild(childIndex PageIndex) bool

// ChildCount 获取子页面数量
func (p *ContentsPage) ChildCount() int
```

## 设计要点

### 1. 索引（PageIndex）的重要性
- **唯一标识**：每个Page都有唯一的PageIndex
- **跨类型引用**：ContentsPage通过索引引用子Page，而非直接持有对象
- **Agent操作**：Agent通过index进行所有操作（expandDetails, movePage等）

### 2. 生命周期与可见性分离
- **Lifecycle**：决定Page是否在上下文窗口内（Active/HotArchived/ColdArchived）
- **Visibility**：仅对Active Page有效，决定展示粒度（Expanded/Hidden）

### 3. ContentPage与DetailPage的区别
| 特性 | ContentsPage | DetailPage |
|------|-------------|-----------|
| 角色 | 目录节点 | 叶子节点 |
| 子节点 | 可以有子Page | 无子Page |
| 内容存储 | 子Page的索引列表 | 原始消息内容（detail字段） |
| Expanded状态 | 显示子Page索引 | 显示detail内容 |
| Hidden状态 | 隐藏子Page索引 | 隐藏detail内容 |

### 4. 懒加载设计
- ContentsPage不直接存储子Page对象，只存储索引
- 实际的Page对象由ContextSystem统一管理
- Agent通过index请求ContextSystem获取Page详情

### 5. 序列化支持
- 所有Page都支持Marshal/Unmarshal
- 用于Storage的持久化和恢复
- 支持Active ↔ Archived的转换

## 使用示例

```go
// 创建DetailPage
detailPage := NewDetailPage(
    "goroutine原理",
    "解释goroutine的调度机制",
    "User: Go的goroutine是如何调度的？\nAssistant: ...",
    "root",
)

// 创建ContentsPage
contentsPage := NewContentsPage(
    "Go语言讨论",
    "关于Go语言的专题讨论",
    "root",
)

// 添加子页面到ContentsPage
contentsPage.AddChild(detailPage.GetIndex())

// 设置可见性
detailPage.SetVisibility(Expanded)
contentsPage.SetVisibility(Expanded)

// 序列化（用于持久化）
data, _ := detailPage.Marshal()

// 反序列化（从持久化恢复）
restoredPage := &DetailPage{}
restoredPage.Unmarshal(data)
```

## 与ContextSystem的交互

Page本身不直接处理：
- ✗ Page树遍历
- ✗ 上下文窗口计算
- ✗ MessageList生成

这些职责由ContextSystem承担，Page专注于：
- ✓ 自身状态管理
- ✓ 数据存储
- ✓ 序列化/反序列化
- ✓ 父子关系维护
