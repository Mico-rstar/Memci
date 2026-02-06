# Segment 设计文档

## 概述

Segment 是上下文管理系统的逻辑分段抽象，用于将整个上下文空间划分为多个独立的区域。

**核心特点**：
- 每个 Segment 都有一个 root ContentsPage
- 多个 Segment 的 root Page 并列显示
- Segment 对 Agent **不可见**（Agent 只看到 Page 树）
- 方便开发者逻辑分组（如：系统提示词段、用户交互段）

## Segment 结构体定义

```go
// SegmentID Segment的唯一标识
type SegmentID string

// SegmentType Segment的类型
type SegmentType int

const (
	// SystemSegment 系统级Segment（如系统提示词、安全规则）
	SystemSegment SegmentType = iota
	// UserSegment 用户交互Segment
	UserSegment
	// ToolSegment 工具调用Segment
	ToolSegment
	// CustomSegment 自定义Segment
	CustomSegment
)

// SegmentPermission Segment的权限控制
type SegmentPermission int

const (
	// ReadOnly 只读：Agent不能修改此Segment的任何Page
	ReadOnly SegmentPermission = iota
	// ReadWrite 读写：Agent可以修改此Segment的Page
	ReadWrite
	// SystemManaged 系统管理：只有系统代码可以修改，Agent完全不可操作
	SystemManaged
)

// Segment 上下文空间的逻辑分段
type Segment struct {
	// 基本信息
	id      SegmentID   // 唯一标识
	name    string      // Segment名称
	segType SegmentType // Segment类型
	description string  // Segment描述

	// 根Page
	rootIndex PageIndex // root ContentsPage的索引
	// 注意：Segment 不直接持有 Page 对象，通过索引引用

	// 配置
	maxCapacity int              // 最大Token容量（可选，用于上下文窗口管理）
	permission  SegmentPermission // 权限控制

	// 元数据
	createdAt time.Time
	updatedAt time.Time
	// metadata map[string]interface{} // 扩展元数据
}

// segmentJSON 用于JSON序列化的内部结构
type segmentJSON struct {
	ID          SegmentID         `json:"id"`
	Name        string            `json:"name"`
	SegmentType SegmentType       `json:"segmentType"`
	Description string            `json:"description"`
	RootIndex   string            `json:"rootIndex"`
	MaxCapacity int               `json:"maxCapacity"`
	Permission  SegmentPermission `json:"permission"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}
```

## 核心方法

```go
// NewSegment 创建新的Segment
func NewSegment(id SegmentID, name, description string, segType SegmentType) (*Segment, error)

// GetID 获取Segment ID
func (s *Segment) GetID() SegmentID

// GetName 获取Segment名称
func (s *Segment) GetName() string

// SetName 设置Segment名称
func (s *Segment) SetName(name string) error

// GetDescription 获取Segment描述
func (s *Segment) GetDescription() string

// SetDescription 设置Segment描述
func (s *Segment) SetDescription(description string) error

// GetType 获取Segment类型
func (s *Segment) GetType() SegmentType

// GetRootIndex 获取root Page索引
func (s *Segment) GetRootIndex() PageIndex

// SetRootIndex 设置root Page索引（通常由ContextSystem调用）
func (s *Segment) SetRootIndex(index PageIndex) error

// GetMaxCapacity 获取最大Token容量
func (s *Segment) GetMaxCapacity() int

// SetMaxCapacity 设置最大Token容量
func (s *Segment) SetMaxCapacity(capacity int) error

// GetPermission 获取权限级别
func (s *Segment) GetPermission() SegmentPermission

// SetPermission 设置权限级别
func (s *Segment) SetPermission(permission SegmentPermission) error

// IsReadOnly 检查是否为只读
func (s *Segment) IsReadOnly() bool

// CanModify 检查Agent是否可以修改此Segment的Page
func (s *Segment) CanModify() bool

// 序列化/反序列化
func (s *Segment) Marshal() ([]byte, error)
func (s *Segment) Unmarshal(data []byte) error
```

## 设计要点

### 1. Segment 与 Page 的关系

```
ContextSystem
│
├── Segment List
│   ├── Segment (系统提示词段)
│   │   └── root: ContentsPage [index: sys-0]
│   │       └─ [系统提示词、安全规则等子Page]
│   │
│   └── Segment (用户交互段)
│       └── root: ContentsPage [index: usr-0]
│           ├─ ContentsPage [index: usr-1]
│           └─ DetailPage [index: usr-2]
│
└── Page Registry (全局Page存储)
    ├── sys-0: ContentsPage
    ├── usr-0: ContentsPage
    ├── usr-1: ContentsPage
    └── usr-2: DetailPage
```

**关键点**：
- Segment 只存储 root Page 的索引
- 实际的 Page 对象由 ContextSystem 统一管理
- Segment 之间通过 root Page 并列显示

### 2. Agent 不可见性

**Agent 视角**：
```
║ ─────────────────────────────────────────────────────────────  ║
║ 📁 系统提示词段 (ContentsPage) [index: sys-0] [Expanded]         ║
║ ─────────────────────────────────────────────────────────────  ║
║ 📁 用户交互段 (ContentsPage) [index: usr-0] [Expanded]         ║
║ ─────────────────────────────────────────────────────────────  ║
```

Agent 看到的是多个 root Page，不知道 Segment 的存在。

**开发者视角**：
```go
// 开发者可以创建和配置Segment
systemSeg := NewSegment("sys", "系统提示词", "系统级上下文", SystemSegment)
systemSeg.SetPermission(ReadOnly)

userSeg := NewSegment("usr", "用户交互", "用户对话历史", UserSegment)
userSeg.SetPermission(ReadWrite)

// 添加到ContextSystem（值传递，所有权转移）
contextSystem.AddSegment(*systemSeg)  // ← 解引用，传递副本
contextSystem.AddSegment(*userSeg)

// ✅ 之后对原变量的修改不影响系统内的副本
systemSeg.SetPermission(ReadWrite)  // 不影响 ContextSystem 中的副本
```

**所有权原则**：
- `AddSegment` 接受值类型（`Segment` 而非 `*Segment`）
- 调用时使用 `*segment` 解引用，传递副本给 ContextSystem
- ContextSystem 内部存储指针，但指向的是独立的副本
- 开发者对原始变量的修改不会影响系统内的 Segment

### 3. Segment 的作用

| 作用 | 说明 | 示例 |
|------|------|------|
| **逻辑分组** | 将相关 Page 组织在一起 | 系统提示词、用户交互、工具调用 |
| **独立管理** | 每个 Segment 可独立配置容量 | 设置不同的 token 限制 |
| **显示控制** | 通过 ContextSystem 中的添加顺序控制显示 | 先添加的 Segment 显示在前 |
| **生命周期管理** | Segment 级别的归档和恢复 | 历史对话段可以整体归档 |

### 4. SegmentType 的使用

```go
// 预定义的Segment类型
const (
	SystemSegment SegmentType = iota  // 系统级（提示词、规则）
	UserSegment                       // 用户交互
	ToolSegment                       // 工具调用
	CustomSegment                     // 自定义
)

// 使用场景
systemSeg := NewSegment("sys", "System", "System prompts", SystemSegment)
userSeg := NewSegment("usr", "User", "User interactions", UserSegment)
toolSeg := NewSegment("tool", "Tools", "Tool calls history", ToolSegment)
customSeg := NewSegment("project", "Project", "Project specific context", CustomSegment)
```

### 5. 显示顺序的控制

Segment 的显示顺序由 ContextSystem 中的添加顺序决定：

```go
// ContextSystem 使用 slice 保持顺序
type ContextSystem struct {
    segments []*Segment  // 按添加顺序存储，显示时即按此顺序
}

// 创建Segment（不需要priority参数）
sysSeg := NewSegment("sys", "System", "...", SystemSegment)
userSeg := NewSegment("usr", "User", "...", UserSegment)

// 按顺序添加到ContextSystem（值传递）
contextSystem.AddSegment(*sysSeg)   // 先添加，显示在最前
contextSystem.AddSegment(*userSeg)  // 后添加，显示在后面

// Agent看到的视图（按添加顺序）
║ ─────────────────────────────────────────────────────────────  ║
║ 📁 System (ContentsPage) [index: sys-0]    ← 先添加           ║
║ ─────────────────────────────────────────────────────────────  ║
║ 📁 User (ContentsPage) [index: usr-0]      ← 后添加           ║
║ ─────────────────────────────────────────────────────────────  ║
```

**注意**：Segment 的显示顺序在**系统初始化时确定**，运行时不需要动态调整。开发者应该在创建 ContextSystem 时按正确的顺序添加 Segment。

### 6. MaxCapacity 的作用

用于上下文窗口管理：

```go
// 设置Segment的最大Token容量
userSeg.SetMaxCapacity(4000)  // 用户交互段最多4000 tokens

// ContextWindow在构建MessageList时会考虑这个限制
// 如果超出，会自动折叠子Page
```

### 7. 权限控制（安全机制）

Segment 提供了三级权限控制，用于保护系统级内容不被 Agent 误操作：

```go
// 权限级别
const (
	ReadOnly       // 只读：Agent只能查看，不能修改
	ReadWrite       // 读写：Agent可以完全操作
	SystemManaged   // 系统管理：Agent完全不可见不可操作
)
```

**代理模式的权限检查流程**：

```
Agent 操作请求
     │
     ▼
┌─────────────────────────────┐
│   AgentContext (代理)          │
│   ├─ checkPermission()       │  ← 统一的权限检查入口
│   └─ 检查通过？               │
│      │                       │
│      Yes                     No
│      │                       │
│      ▼                       ▼
│ 调用 ContextSystem      返回权限错误
│   内部方法执行
```

**优势**：
- 权限检查逻辑集中在 `AgentContext.checkPermission()`
- ContextSystem 方法只需关注业务逻辑
- 易于添加审计、日志等横切关注点

**典型权限配置**：

```go
// 系统提示词段：只读（保护系统提示词不被修改）
sysSeg := NewSegment("sys", "System", "System prompts", SystemSegment)
sysSeg.SetPermission(ReadOnly)

// 重要：系统提示词段的root Page必须默认为Expanded状态
sysRoot, _ := NewContentsPage("System", "System prompts and rules", "")
sysRoot.SetVisibility(Expanded)  // 强制展开，确保Agent始终受约束
sysRoot.SetIndex(PageIndex("sys-0"))
sysSeg.SetRootIndex(sysRoot.GetIndex())

// 用户交互段：读写（Agent可以自由操作）
userSeg := NewSegment("usr", "User", "User interactions", UserSegment)
userSeg.SetPermission(ReadWrite)

// 安全规则段：系统管理（Agent完全不可见）
securitySeg := NewSegment("security", "Security", "Security rules", SystemSegment)
securitySeg.SetPermission(SystemManaged)
```

**系统提示词段的特殊约束**：

| 约束 | 原因 |
|------|------|
| Root Page 默认 `Expanded` | 确保Agent始终能看到系统约束 |
| 禁止 `hideDetails()` | Agent不能通过隐藏来绕过系统提示词 |
| 禁止 `updatePage()` | 保护系统提示词内容不被修改 |

**受权限影响的操作（通过AgentContext代理）**：

| Agent 操作 | ReadOnly | ReadWrite | SystemManaged |
|-----------|----------|-----------|---------------|
| `ExpandDetails()` | ✅ 允许 | ✅ 允许 | ❌ 拒绝（不可见） |
| `HideDetails()` | ⚠️ 受限 | ✅ 允许 | ❌ 拒绝 |
| `UpdatePage()` | ❌ 拒绝 | ✅ 允许 | ❌ 拒绝 |
| `MovePage()` | ❌ 拒绝 | ✅ 允许 | ❌ 拒绝 |
| `RemovePage()` | ❌ 拒绝 | ✅ 允许 | ❌ 拒绝 |
| `createDetailPage()` | ❌ 拒绝 | ✅ 允许 | ❌ 拒绝 |
| `createContentsPage()` | ❌ 拒绝 | ✅ 允许 | ❌ 拒绝 |

**重要说明**：
- 对于 **SystemSegment 类型的 Segment**，其 **root Page** 禁止执行 `HideDetails()`，即使权限是 `ReadOnly`
- 这是防止 Agent 通过隐藏系统提示词来绕过系统约束
- root Page 默认必须是 `Expanded` 状态

## 典型使用场景

### 场景1：系统提示词 + 用户交互

```go
// 创建系统提示词Segment（只读，保护系统提示词）
sysSeg := NewSegment("sys", "System", "System-level context", SystemSegment)
sysSeg.SetPermission(ReadOnly)  // Agent只能查看，不能修改

// 系统提示词的root Page必须默认为Expanded
sysRoot, _ := NewContentsPage("System", "System prompts and rules", "")
sysRoot.SetVisibility(Expanded)  // 强制展开，确保Agent始终受约束
sysRoot.SetIndex(PageIndex("sys-0"))
sysSeg.SetRootIndex(sysRoot.GetIndex())

// 添加系统提示词子Page
systemPromptPage := NewDetailPage("System Prompt", "Main system prompt", "You are a helpful AI...", "sys-0")
systemPromptPage.SetIndex(PageIndex("sys-1"))
sysRoot.AddChild(systemPromptPage.GetIndex())

// 创建用户交互Segment（读写，Agent可以自由操作）
userSeg := NewSegment("usr", "User", "User conversation history", UserSegment)
userSeg.SetPermission(ReadWrite)  // Agent可以修改
userRoot, _ := NewContentsPage("User", "User interactions", "")
userRoot.SetIndex(PageIndex("usr-0"))
userSeg.SetRootIndex(userRoot.GetIndex())
```



## 与其他组件的关系

### Segment vs Page

| 特性 | Segment | Page |
|------|---------|------|
| 抽象层级 | 逻辑分组 | 页面容器 |
| 可见性 | Agent不可见 | Agent可见 |
| 包含内容 | root Page索引 | messages或子Page索引 |
| 主要用途 | 逻辑分组 | 存储内容 |

### Segment vs ContextSystem

```go
type ContextSystem struct {
    segments    []*Segment           // 按顺序存储，显示顺序即添加顺序
    segmentMap  map[SegmentID]*Segment  // 快速查找
    pages       map[PageIndex]Page  // 管理所有Page
    // ...
}

// ContextSystem 方法
func (cs *ContextSystem) AddSegment(segment Segment) error  // 值传递，获取所有权
func (cs *ContextSystem) RemoveSegment(id SegmentID) error
func (cs *ContextSystem) GetSegment(id SegmentID) (Segment, error)  // 返回副本
func (cs *ContextSystem) ListSegments() ([]Segment, error)  // 返回副本
func (cs *ContextSystem) UpdateSegment(id SegmentID, name, description string) error
func (cs *ContextSystem) SetSegmentPermission(id SegmentID, permission SegmentPermission) error
```

**注意**：Segment 顺序在初始化时通过添加顺序确定，运行时不提供动态调整方法。

**GetSegment 返回副本的设计**：
```go
// ❌ 错误：返回指针会破坏封装
func (cs *ContextSystem) GetSegment(id SegmentID) (*Segment, error)

// ✅ 正确：返回值类型副本
func (cs *ContextSystem) GetSegment(id SegmentID) (Segment, error)

// 使用示例
seg, _ := contextSystem.GetSegment("sys")
fmt.Println(seg.GetName())  // ✅ 可以读取
seg.SetPermission(ReadWrite)  // ❌ 修改副本，不影响系统

// 如需修改，应使用专门的方法
contextSystem.SetSegmentPermission("sys", ReadWrite)  // ✅ 正确
```

**为什么返回副本**：
- 保证 ContextSystem 对状态的完全控制
- 防止外部代码绕过权限检查
- Segment 结构体较小，复制开销可接受
- 符合封装原则：修改应通过专门的方法

### ContextSystem 中的权限检查（代理模式）

使用**代理模式**统一处理权限检查，将权限逻辑集中管理：

**所有权转移**：
```go
// AddSegment 使用值传递（而非指针传递）
func (cs *ContextSystem) AddSegment(segment Segment) error

// 开发者调用时需解引用
seg := NewSegment("sys", "System", "...", SystemSegment)
contextSystem.AddSegment(*seg)  // ← 值传递，系统获得独立副本

// ✅ 之后对 seg 的修改不影响 ContextSystem
seg.SetPermission(ReadWrite)  // 不影响系统内的副本
```

**为什么使用值传递**：
- 防止悬垂引用：开发者删除原变量不影响系统
- 避免外部修改：保证 ContextSystem 内部状态的一致性
- 明确所有权：值传递表示所有权的转移（类似 Rust 的 Move 语义）
- 性能考虑：Segment 结构体较小，复制的开销可接受

**双套 API 设计**：
```go
// 公开 API：返回副本，供外部使用
func (cs *ContextSystem) GetSegment(id SegmentID) (Segment, error)
func (cs *ContextSystem) GetSegmentByPageIndex(pageIndex PageIndex) (Segment, error)

// 内部 API：返回指针，供系统内部和 AgentContext 使用
func (cs *ContextSystem) getSegmentByPageIndexInternal(pageIndex PageIndex) (*Segment, error)
```

**代理模式的权限检查**：

```go
// ContextSystem 内部实现（不包含权限检查）
type ContextSystem struct {
    segments    []*Segment
    segmentMap  map[SegmentID]*Segment
    pages       map[PageIndex]Page
}

// 内部方法：不进行权限检查
func (cs *ContextSystem) updatePageInternal(pageIndex PageIndex, name, description string) error
func (cs *ContextSystem) movePageInternal(source, target PageIndex) error
func (cs *ContextSystem) removePageInternal(pageIndex PageIndex) error
// ... 其他内部方法

// 辅助方法：根据 pageIndex 查找所属的 Segment（内部方法）
func (cs *ContextSystem) getSegmentByPageIndexInternal(pageIndex PageIndex) (*Segment, error) {
    for _, seg := range cs.segments {
        if seg == nil {
            continue
        }
        prefix := string(seg.GetID()) + "-"
        if strings.HasPrefix(string(pageIndex), prefix) {
            return seg, nil
        }
    }
    return nil, fmt.Errorf("no segment found for page %s", pageIndex)
}
```

**AgentContext 代理（统一权限检查入口）**：

```go
// AgentContext Agent的上下文代理，负责统一权限检查
type AgentContext struct {
    contextSystem *ContextSystem  // 被代理的ContextSystem
}

// NewAgentContext 创建Agent上下文代理
func NewAgentContext(cs *ContextSystem) *AgentContext {
    return &AgentContext{contextSystem: cs}
}

// 权限检查方法
func (ac *AgentContext) checkPermission(pageIndex PageIndex, requireWrite bool) error {
    // 1. 找到所属Segment（使用内部方法）
    segment, err := ac.contextSystem.getSegmentByPageIndexInternal(pageIndex)
    if err != nil {
        return fmt.Errorf("segment for page %s not found", pageIndex)
    }

    // 2. SystemManaged权限：完全不可见
    if segment.GetPermission() == SystemManaged {
        return fmt.Errorf("access denied: page %s is in system-managed segment", pageIndex)
    }

    // 3. 写操作权限检查
    if requireWrite && !segment.CanModify() {
        return fmt.Errorf("access denied: page %s is in read-only segment %s",
            pageIndex, segment.GetID())
    }

    return nil
}

    return nil
}

// Agent操作接口（带权限检查）
func (ac *AgentContext) UpdatePage(pageIndex PageIndex, name, description string) error {
    // 权限检查
    if err := ac.checkPermission(pageIndex, true); err != nil {
        return err
    }
    // 调用内部方法
    return ac.contextSystem.updatePageInternal(pageIndex, name, description)
}

func (ac *AgentContext) MovePage(source, target PageIndex) error {
    if err := ac.checkPermission(source, true); err != nil {
        return err
    }
    if err := ac.checkPermission(target, true); err != nil {
        return err
    }
    return ac.contextSystem.movePageInternal(source, target)
}

func (ac *AgentContext) RemovePage(pageIndex PageIndex) error {
    if err := ac.checkPermission(pageIndex, true); err != nil {
        return err
    }
    return ac.contextSystem.removePageInternal(pageIndex)
}

func (ac *AgentContext) CreateDetailPage(name, description, detail string, parentIndex PageIndex) error {
    if err := ac.checkPermission(parentIndex, true); err != nil {
        return err
    }
    return ac.contextSystem.createDetailPageInternal(name, description, detail, parentIndex)
}

func (ac *AgentContext) ExpandDetails(pageIndex PageIndex) error {
    // 只读操作，检查可见性即可
    if err := ac.checkPermission(pageIndex, false); err != nil {
        return err
    }
    return ac.contextSystem.expandDetailsInternal(pageIndex)
}
```

**使用方式**：

```go
// 创建系统
contextSystem := NewContextSystem()
sysSeg := NewSegment("sys", "System", "...", SystemSegment)
sysSeg.SetPermission(ReadOnly)
contextSystem.AddSegment(*sysSeg)  // 值传递，解引用

// 创建Agent代理（Agent通过代理操作）
agentCtx := NewAgentContext(contextSystem)

// Agent尝试修改系统提示词（会被代理拦截）
err := agentCtx.UpdatePage("sys-1", "Modified", "...")
// 返回：access denied: page sys-1 is in read-only segment sys

// Agent查看内容（只读操作，允许）
err := agentCtx.ExpandDetails("sys-1")
// 成功执行
```

**代理模式的优势**：

| 优势 | 说明 |
|------|------|
| **集中管理** | 所有权限检查逻辑在 AgentContext 中 |
| **职责分离** | ContextSystem 只负责状态管理，不关心权限 |
| **易于维护** | 权限规则变更只需修改 AgentContext |
| **防止遗漏** | 无法绕过代理直接操作内部方法 |
| **可扩展性** | 可以轻松添加审计、日志等横切关注点 |

## 序列化格式

Segment 支持序列化，用于持久化：

```json
{
  "id": "usr",
  "name": "User",
  "segmentType": 1,
  "description": "User conversation history",
  "rootIndex": "usr-0",
  "maxCapacity": 4000,
  "permission": 1,
  "createdAt": "2025-02-05T10:00:00Z",
  "updatedAt": "2025-02-05T10:00:00Z"
}
```

**权限值说明**：
- `0` = `ReadOnly`：只读
- `1` = `ReadWrite`：读写
- `2` = `SystemManaged`：系统管理

## 索引命名规范

为避免冲突，建议采用以下索引命名规范：

```
{segment-id}-{index}

示例：
- sys-0, sys-1, sys-2      (系统Segment的Page)
- usr-0, usr-1, usr-2      (用户Segment的Page)
- tool-0, tool-1, tool-2    (工具Segment的Page)
- project-a-0, project-a-1 (项目A的Page)
```

## 注意事项

1. **Segment 是开发者工具**：Agent 不感知 Segment，只看到 Page 树
2. **root Page 必须是 ContentsPage**：因为 Segment 本质上是一个容器
3. **索引唯一性**：不同 Segment 的 Page 索引不能冲突（建议使用前缀）
4. **显示顺序**：通过 ContextSystem 中的添加顺序控制，先添加的 Segment 显示在前
5. **容量控制**：通过 maxCapacity 控制每个 Segment 的 token 使用
6. **权限控制**：系统提示词段应设置为 `ReadOnly` 或 `SystemManaged`，防止 Agent 误修改系统级内容
7. **默认权限**：创建 Segment 时应根据类型自动设置默认权限
   - `SystemSegment`: 默认 `ReadOnly`
   - `UserSegment`: 默认 `ReadWrite`
   - `ToolSegment`: 默认 `ReadWrite`
   - `CustomSegment`: 默认 `ReadWrite`
8. **所有权管理**：
   - `AddSegment` 使用**值传递**（`Segment` 类型，非 `*Segment`）
   - 调用时需解引用：`contextSystem.AddSegment(*segment)`
   - ContextSystem 存储独立副本，开发者对原变量的修改不影响系统
   - 这是 Go 中实现所有权转移的标准方式
9. **Page 结构完整性约束**：
   - **核心规则**：除了 root page 外，所有 Page 都必须有父节点
   - root page 是每个 Segment 的根节点，是唯一允许没有父节点的 Page
   - ContextSystem 在 `AddPage()` 时会验证：
     - 如果 Page 没有父节点，必须是某个 Segment 的 root
     - 如果 Page 有父节点，父节点必须存在且是 ContentsPage
   - 这个约束确保没有孤儿节点，保持树结构完整
