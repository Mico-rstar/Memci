# ContextManager 设计文档

## 概述

ContextManager 是上下文管理系统的统一入口，负责协调 AgentContext（写操作）和 ContextWindow（读操作），简化使用流程。

## 核心职责

1. **统一管理**：统一管理 ContextSystem、AgentContext、ContextWindow
2. **简化 API**：提供单一的入口点，隐藏组件间的协作细节
3. **生命周期管理**：管理所有组件的创建和销毁
4. **操作协调**：协调 Agent 操作和消息生成的时序

## ContextManager 结构体

```go
// ContextManager 上下文管理器
type ContextManager struct {
    system  *ContextSystem   // 核心存储系统
    agent   *AgentContext    // Agent 代理层
    window  *ContextWindow   // 渲染层
    mu      sync.RWMutex     // 并发控制
}

// NewContextManager 创建新的上下文管理器
func NewContextManager() *ContextManager
```

## 核心 API

### 初始化方法

```go
// Initialize 初始化上下文管理器
func (cm *ContextManager) Initialize() error

// SetupSegment 创建并配置 Segment
func (cm *ContextManager) SetupSegment(
    id SegmentID,
    name string,
    description string,
    segType SegmentType,
    permission SegmentPermission,
) error
```

### Agent 操作方法（代理到 AgentContext）

```go
// UpdatePage 更新 Page 信息
func (cm *ContextManager) UpdatePage(pageIndex PageIndex, name, description string) error

// ExpandDetails 展开 Page 详情
func (cm *ContextManager) ExpandDetails(pageIndex PageIndex) error

// HideDetails 隐藏 Page 详情
func (cm *ContextManager) HideDetails(pageIndex PageIndex) error

// MovePage 移动 Page
func (cm *ContextManager) MovePage(source, target PageIndex) error

// RemovePage 删除 Page
func (cm *ContextManager) RemovePage(pageIndex PageIndex) error

// CreateDetailPage 创建 DetailPage
func (cm *ContextManager) CreateDetailPage(
    name, description, detail string,
    parentIndex PageIndex,
) (PageIndex, error)

// CreateContentsPage 创建 ContentsPage
func (cm *ContextManager) CreateContentsPage(
    name, description string,
    parentIndex PageIndex,
    children ...PageIndex,
) (PageIndex, error)
```

### 渲染方法（代理到 ContextWindow）

```go
// GenerateMessageList 生成发送给模型的 MessageList
func (cm *ContextManager) GenerateMessageList() (*MessageList, error)

// EstimateTokens 估算当前 MessageList 的 token 数量
func (cm *ContextManager) EstimateTokens() (int, error)

// AutoCollapse 自动折叠以适应 token 限制
func (cm *ContextManager) AutoCollapse(maxTokens int) ([]PageIndex, error)
```

### 查询方法

```go
// GetPage 获取 Page
func (cm *ContextManager) GetPage(pageIndex PageIndex) (Page, error)

// GetChildren 获取子 Page
func (cm *ContextManager) GetChildren(pageIndex PageIndex) ([]Page, error)

// GetSegment 获取 Segment
func (cm *ContextManager) GetSegment(id SegmentID) (Segment, error)

// ListSegments 列出所有 Segment
func (cm *ContextManager) ListSegments() ([]Segment, error)
```

## 实现示例

```go
package context

import (
    "sync"
)

// ContextManager 上下文管理器
type ContextManager struct {
    system *ContextSystem
    agent  *AgentContext
    window *ContextWindow
    mu     sync.RWMutex
}

// NewContextManager 创建新的上下文管理器
func NewContextManager() *ContextManager {
    system := NewContextSystem()
    agent := NewAgentContext(system)
    window := NewContextWindow(system)

    return &ContextManager{
        system: system,
        agent:  agent,
        window: window,
    }
}

// Initialize 初始化上下文管理器
func (cm *ContextManager) Initialize() error {
    cm.mu.Lock()
    defer cm.mu.Unlock()

    // 创建默认的系统提示词段
    sysSeg := NewSegment("sys", "System", "System prompts", SystemSegment)
    sysSeg.SetPermission(ReadOnly)
    if err := cm.system.AddSegment(*sysSeg); err != nil {
        return err
    }

    // 创建系统提示词 root page
    sysRoot := NewContentsPage("System", "System prompts", "")
    sysRoot.SetVisibility(Expanded)
    sysRoot.SetIndex(PageIndex("sys-0"))
    sysSeg.SetRootIndex(sysRoot.GetIndex())

    if err := cm.system.AddPage(sysRoot); err != nil {
        return err
    }

    // 创建用户交互段
    usrSeg := NewSegment("usr", "User", "User interactions", UserSegment)
    usrSeg.SetPermission(ReadWrite)
    if err := cm.system.AddSegment(*usrSeg); err != nil {
        return err
    }

    // 创建用户 root page
    usrRoot := NewContentsPage("User", "User interactions", "")
    usrRoot.SetVisibility(Expanded)
    usrRoot.SetIndex(PageIndex("usr-0"))
    usrSeg.SetRootIndex(usrRoot.GetIndex())

    if err := cm.system.AddPage(usrRoot); err != nil {
        return err
    }

    return nil
}

// SetupSegment 创建并配置 Segment
func (cm *ContextManager) SetupSegment(
    id SegmentID,
    name string,
    description string,
    segType SegmentType,
    permission SegmentPermission,
) error {
    cm.mu.Lock()
    defer cm.mu.Unlock()

    seg := NewSegment(id, name, description, segType)
    seg.SetPermission(permission)

    return cm.system.AddSegment(*seg)
}

// ============ Agent 操作方法 ============

// UpdatePage 更新 Page 信息
func (cm *ContextManager) UpdatePage(pageIndex PageIndex, name, description string) error {
    cm.mu.Lock()
    defer cm.mu.Unlock()

    return cm.agent.UpdatePage(pageIndex, name, description)
}

// ExpandDetails 展开 Page 详情
func (cm *ContextManager) ExpandDetails(pageIndex PageIndex) error {
    cm.mu.Lock()
    defer cm.mu.Unlock()

    return cm.agent.ExpandDetails(pageIndex)
}

// HideDetails 隐藏 Page 详情
func (cm *ContextManager) HideDetails(pageIndex PageIndex) error {
    cm.mu.Lock()
    defer cm.mu.Unlock()

    return cm.agent.HideDetails(pageIndex)
}

// MovePage 移动 Page
func (cm *ContextManager) MovePage(source, target PageIndex) error {
    cm.mu.Lock()
    defer cm.mu.Unlock()

    return cm.agent.MovePage(source, target)
}

// RemovePage 删除 Page
func (cm *ContextManager) RemovePage(pageIndex PageIndex) error {
    cm.mu.Lock()
    defer cm.mu.Unlock()

    return cm.agent.RemovePage(pageIndex)
}

// CreateDetailPage 创建 DetailPage
func (cm *ContextManager) CreateDetailPage(
    name, description, detail string,
    parentIndex PageIndex,
) (PageIndex, error) {
    cm.mu.Lock()
    defer cm.mu.Unlock()

    return cm.agent.CreateDetailPage(name, description, detail, parentIndex)
}

// CreateContentsPage 创建 ContentsPage
func (cm *ContextManager) CreateContentsPage(
    name, description string,
    parentIndex PageIndex,
    children ...PageIndex,
) (PageIndex, error) {
    cm.mu.Lock()
    defer cm.mu.Unlock()

    return cm.agent.CreateContentsPage(name, description, parentIndex, children...)
}

// ============ 渲染方法 ============

// GenerateMessageList 生成发送给模型的 MessageList
func (cm *ContextManager) GenerateMessageList() (*MessageList, error) {
    cm.mu.RLock()
    defer cm.mu.RUnlock()

    return cm.window.GenerateMessageList()
}

// EstimateTokens 估算当前 MessageList 的 token 数量
func (cm *ContextManager) EstimateTokens() (int, error) {
    cm.mu.RLock()
    defer cm.mu.RUnlock()

    return cm.window.EstimateTokens()
}

// AutoCollapse 自动折叠以适应 token 限制
func (cm *ContextManager) AutoCollapse(maxTokens int) ([]PageIndex, error) {
    cm.mu.Lock()
    defer cm.mu.Unlock()

    return cm.window.AutoCollapse(maxTokens)
}

// ============ 查询方法 ============

// GetPage 获取 Page
func (cm *ContextManager) GetPage(pageIndex PageIndex) (Page, error) {
    cm.mu.RLock()
    defer cm.mu.RUnlock()

    return cm.agent.GetPage(pageIndex)
}

// GetChildren 获取子 Page
func (cm *ContextManager) GetChildren(pageIndex PageIndex) ([]Page, error) {
    cm.mu.RLock()
    defer cm.mu.RUnlock()

    return cm.agent.GetChildren(pageIndex)
}

// GetSegment 获取 Segment
func (cm *ContextManager) GetSegment(id SegmentID) (Segment, error) {
    cm.mu.RLock()
    defer cm.mu.RUnlock()

    return cm.agent.GetSegment(id)
}

// ListSegments 列出所有 Segment
func (cm *ContextManager) ListSegments() ([]Segment, error) {
    cm.mu.RLock()
    defer cm.mu.RUnlock()

    return cm.agent.ListSegments()
}
```

## 设计要点

### 1. 统一入口

ContextManager 作为唯一的入口点，简化了使用：

```go
// 之前：需要分别创建和管理三个组件
contextSystem := NewContextSystem()
agentCtx := NewAgentContext(contextSystem)
contextWindow := NewContextWindow(contextSystem)

// 现在：只需要一个组件
contextMgr := NewContextManager()
contextMgr.Initialize()
```

### 2. 读写分离

内部仍然保持读写分离：
- 写操作 → AgentContext（带权限检查）
- 读操作 → ContextWindow（只读渲染）

但对外暴露统一的 API。

### 3. 线程安全

ContextManager 提供内置的线程安全保障：
- 使用 `sync.RWMutex` 保护所有操作
- 读操作使用 `RLock()`
- 写操作使用 `Lock()`

### 4. 组件协作

ContextManager 协调三个组件的协作：

```
ContextManager
    │
    ├── ContextSystem ← 核心存储
    ├── AgentContext  ← 写操作代理
    └── ContextWindow ← 读操作渲染
```

## 典型使用流程

### 1. 基本使用

```go
// 1. 创建 ContextManager
contextMgr := NewContextManager()

// 2. 初始化（创建默认 Segment）
if err := contextMgr.Initialize(); err != nil {
    log.Fatal(err)
}

// 3. Agent 操作
newPageIndex, err := contextMgr.CreateDetailPage(
    "我的问题",
    "关于Go的问题",
    "如何实现协程？",
    "usr-0",  // 添加到用户段的 root
)
if err != nil {
    log.Fatal(err)
}

// 4. 展开 Page
err = contextMgr.ExpandDetails(newPageIndex)
if err != nil {
    log.Fatal(err)
}

// 5. 生成消息列表
messageList, err := contextMgr.GenerateMessageList()
if err != nil {
    log.Fatal(err)
}

// 6. 发送给模型
response, err := llm.Send(messageList)
```

### 2. 自定义 Segment

```go
contextMgr := NewContextManager()

// 创建自定义 Segment
err := contextMgr.SetupSegment(
    "project-a",
    "Project A",
    "Project A related context",
    CustomSegment,
    ReadWrite,
)
if err != nil {
    log.Fatal(err)
}

// 添加 Page 到自定义 Segment
// ...（需要先获取该 Segment 的 root index）
```

### 3. Token 管理

```go
contextMgr := NewContextManager()
contextMgr.Initialize()

// 执行一些操作...
contextMgr.CreateDetailPage(...)
contextMgr.CreateDetailPage(...)

// 检查 token 数量
tokens, err := contextMgr.EstimateTokens()
if err != nil {
    log.Fatal(err)
}

// 如果超出限制，自动折叠
if tokens > 8000 {
    collapsedPages, err := contextMgr.AutoCollapse(8000)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Collapsed pages: %v", collapsedPages)
}
```

## 与其他组件的关系

### ContextManager vs AgentContext

| 特性 | ContextManager | AgentContext |
|------|----------------|--------------|
| 抽象层级 | 统一管理层 | 代理层 |
| 职责 | 协调所有组件 | 权限检查 |
| 线程安全 | 有（内置锁） | 无 |
| 使用场景 | 应用入口 | 内部组件 |

### ContextManager vs ContextWindow

| 特性 | ContextManager | ContextWindow |
|------|----------------|---------------|
| 抽象层级 | 统一管理层 | 渲染层 |
| 职责 | 协调读写操作 | 渲染 MessageList |
| 操作类型 | 读写 | 只读 |
| 使用场景 | 应用入口 | 内部组件 |

## 架构图

```
应用层
    │
    ▼
ContextManager (统一入口)
    │
    ├── 写操作 → AgentContext → ContextSystem
    │                  ↑
    │                  │
    └── 读操作 → ContextWindow ┘
```

## 注意事项

1. **单例模式**：ContextManager 通常应该是单例，整个应用共享一个实例
2. **生命周期**：ContextManager 的生命周期与应用相同
3. **初始化顺序**：必须先调用 `Initialize()` 才能执行其他操作
4. **线程安全**：所有方法都是线程安全的，可以在 goroutine 中并发调用
5. **错误处理**：写操作失败应该立即处理，避免状态不一致
6. **性能考虑**：频繁调用 `GenerateMessageList()` 可能有性能开销，考虑缓存

## 优势总结

| 优势 | 说明 |
|------|------|
| **简化使用** | 单一入口，隐藏组件复杂性 |
| **统一管理** | 统一管理所有组件的生命周期 |
| **线程安全** | 内置并发控制，外部无需加锁 |
| **易于测试** | 统一的接口便于 mock 和测试 |
| **灵活扩展** | 可以轻松添加新的协调逻辑 |
