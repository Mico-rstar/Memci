# AgentContext 设计文档

## 概述

AgentContext 是 Agent 与 ContextSystem 交互的代理层，负责权限检查和业务逻辑处理。

## 核心职责

1. **权限检查**：集中管理所有操作的权限验证
2. **业务逻辑**：封装复杂的业务操作
3. **代理调用**：代理 Agent 请求到 ContextSystem 内部方法
4. **视图生成**：为 Agent 生成可操作的上下文视图

## 设计模式：代理模式

```
Agent → AgentContext (权限检查 + 业务逻辑) → ContextSystem (内部方法)
```

## AgentContext 结构体定义

```go
// AgentContext Agent的上下文代理
type AgentContext struct {
    // 被代理的ContextSystem
    system *ContextSystem

    // 元数据
    createdAt time.Time
    updatedAt time.Time
}

// NewAgentContext 创建新的AgentContext
func NewAgentContext(system *ContextSystem) *AgentContext
```

## 权限检查方法

```go
// checkPermission 统一的权限检查入口
func (ac *AgentContext) checkPermission(pageIndex PageIndex, operation string) error

// PermissionLevel 操作权限级别
type PermissionLevel int

const (
    // ReadLevel 只读操作（查询、获取信息）
    ReadLevel PermissionLevel = iota
    // WriteLevel 写操作（更新、移动、删除）
    WriteLevel
    // SystemLevel 系统级操作（创建Segment、管理权限）
    SystemLevel
)
```

**实现示例**：

```go
func (ac *AgentContext) checkPermission(pageIndex PageIndex, operation string) error {
    // 1. 查找Page所属的Segment（使用内部方法，返回指针）
    segment, err := ac.system.getSegmentByPageIndexInternal(pageIndex)
    if err != nil {
        return fmt.Errorf("page %s not found", pageIndex)
    }

    // 2. 特殊检查：禁止隐藏系统提示词段的root Page
    if operation == "hideDetails" && segment.GetType() == SystemSegment {
        if pageIndex == segment.GetRootIndex() {
            return fmt.Errorf("cannot hide system prompt root page %s: agent must remain constrained by system prompts", pageIndex)
        }
    }

    // 3. 根据操作类型确定所需权限级别
    requiredLevel := getRequiredLevel(operation)

    // 4. 检查Segment权限是否足够
    segmentPermission := segment.GetPermission()
    if !isPermissionSufficient(segmentPermission, requiredLevel) {
        return fmt.Errorf("operation '%s' on %s requires higher permission", operation, pageIndex)
    }

    return nil
}

func getRequiredLevel(operation string) PermissionLevel {
    switch operation {
    case "updatePage", "movePage", "removePage", "expandDetails", "hideDetails", "createPage":
        return WriteLevel
    case "getSegment", "listSegments", "getPage", "getChildren", "getParent", "getAncestors":
        return ReadLevel
    default:
        return SystemLevel
    }
}

func isPermissionSufficient(segmentPermission SegmentPermission, requiredLevel PermissionLevel) bool {
    switch segmentPermission {
    case SystemManaged:
        return true // 系统管理的Segment可以执行任何操作
    case ReadWrite:
        return requiredLevel <= WriteLevel
    case ReadOnly:
        return requiredLevel <= ReadLevel
    default:
        return false
    }
}
```

## 核心 API（Agent 可见方法）

### Segment 查询方法

```go
// GetSegment 获取Segment（只读，返回副本）
func (ac *AgentContext) GetSegment(id SegmentID) (Segment, error)

// ListSegments 列出所有Segment（只读，返回副本）
func (ac *AgentContext) ListSegments() ([]Segment, error)
```

### Page 操作方法

#### 状态变更方法

```go
// UpdatePage 更新Page信息（写权限）
func (ac *AgentContext) UpdatePage(pageIndex PageIndex, name, description string) error

// ExpandDetails 展开Page详情（写权限）
func (ac *AgentContext) ExpandDetails(pageIndex PageIndex) error

// HideDetails 隐藏Page详情（写权限）
func (ac *AgentContext) HideDetails(pageIndex PageIndex) error
```

**实现示例**：

```go
func (ac *AgentContext) UpdatePage(pageIndex PageIndex, name, description string) error {
    // 1. 权限检查
    if err := ac.checkPermission(pageIndex, "updatePage"); err != nil {
        return err
    }

    // 2. 调用ContextSystem内部方法
    return ac.system.updatePageInternal(pageIndex, name, description)
}

func (ac *AgentContext) ExpandDetails(pageIndex PageIndex) error {
    // 1. 权限检查
    if err := ac.checkPermission(pageIndex, "expandDetails"); err != nil {
        return err
    }

    // 2. 调用ContextSystem内部方法
    return ac.system.expandDetailsInternal(pageIndex)
}
```

#### 结构操作方法

```go
// MovePage 移动Page（写权限）
func (ac *AgentContext) MovePage(source, target PageIndex) error

// RemovePage 删除Page（写权限）
func (ac *AgentContext) RemovePage(pageIndex PageIndex) error

// CreateDetailPage 创建DetailPage（写权限）
func (ac *AgentContext) CreateDetailPage(name, description, detail string, parentIndex PageIndex) (PageIndex, error)

// CreateContentsPage 创建ContentsPage（写权限）
func (ac *AgentContext) CreateContentsPage(name, description string, children ...PageIndex) (PageIndex, error)
```

**实现示例**：

```go
func (ac *AgentContext) MovePage(source, target PageIndex) error {
    // 1. 权限检查（需要检查源和目标）
    if err := ac.checkPermission(source, "movePage"); err != nil {
        return err
    }
    if err := ac.checkPermission(target, "movePage"); err != nil {
        return err
    }

    // 2. 调用ContextSystem内部方法
    return ac.system.movePageInternal(source, target)
}

func (ac *AgentContext) CreateDetailPage(name, description, detail string, parentIndex PageIndex) (PageIndex, error) {
    // 1. 权限检查（父Page必须在可写Segment中）
    if err := ac.checkPermission(parentIndex, "createPage"); err != nil {
        return "", err
    }

    // 2. 获取父Page所属Segment（使用内部方法）
    segment, err := ac.system.getSegmentByPageIndexInternal(parentIndex)
    if err != nil {
        return "", err
    }

    // 3. 生成新PageIndex
    newPageIndex := ac.system.GenerateIndex(segment.GetID())

    // 4. 创建DetailPage
    page, err := NewDetailPage(name, description, detail, parentIndex)
    if err != nil {
        return "", err
    }
    page.SetIndex(newPageIndex)

    // 5. 添加到系统
    if err := ac.system.AddPage(page); err != nil {
        return "", err
    }

    // 6. 添加到父Page
    parent, err := ac.system.GetPage(parentIndex)
    if err != nil {
        return "", err
    }
    if parentPage, ok := parent.(*ContentsPage); ok {
        if err := parentPage.AddChild(newPageIndex); err != nil {
            return "", err
        }
    }

    return newPageIndex, nil
}
```

#### 查询方法

```go
// GetPage 获取Page（只读）
func (ac *AgentContext) GetPage(pageIndex PageIndex) (Page, error)

// GetChildren 获取子Page列表（只读）
func (ac *AgentContext) GetChildren(pageIndex PageIndex) ([]Page, error)

// GetParent 获取父Page（只读）
func (ac *AgentContext) GetParent(pageIndex PageIndex) (Page, error)

// GetAncestors 获取祖先Page列表（只读）
func (ac *AgentContext) GetAncestors(pageIndex PageIndex) ([]Page, error)

// FindPage 查找Page（只读）
func (ac *AgentContext) FindPage(query string) []Page
```

**实现示例**：

```go
func (ac *AgentContext) GetPage(pageIndex PageIndex) (Page, error) {
    // 1. 权限检查
    if err := ac.checkPermission(pageIndex, "getPage"); err != nil {
        return nil, err
    }

    // 2. 调用ContextSystem内部方法
    return ac.system.GetPage(pageIndex)
}

func (ac *AgentContext) GetChildren(pageIndex PageIndex) ([]Page, error) {
    // 1. 权限检查
    if err := ac.checkPermission(pageIndex, "getChildren"); err != nil {
        return nil, err
    }

    // 2. 调用ContextSystem内部方法
    return ac.system.GetChildren(pageIndex)
}
```

## 与其他组件的关系

### AgentContext vs ContextSystem

| 特性 | AgentContext | ContextSystem |
|------|-------------|---------------|
| 抽象层级 | 代理层 | 核心存储层 |
| 可见性 | Agent 可见 | Agent 不可见 |
| 权限检查 | 有 | 无 |
| 方法命名 | 公开方法 | 内部方法（Internal后缀） |
| 职责 | 权限 + 业务逻辑 | 状态管理 |

### AgentContext vs Segment

| 特性 | AgentContext | Segment |
|------|-------------|---------|
| 关系 | 使用 Segment | 被 AgentContext 使用 |
| 权限 | 检查者 | 被检查者 |
| 操作 | 代理操作 | 定义权限级别 |

## 操作权限矩阵

| 操作 | ReadOnly | ReadWrite | SystemManaged |
|------|----------|-----------|---------------|
| GetSegment | ✓ | ✓ | ✓ |
| ListSegments | ✓ | ✓ | ✓ |
| GetPage | ✓ | ✓ | ✓ |
| GetChildren | ✓ | ✓ | ✓ |
| GetParent | ✓ | ✓ | ✓ |
| GetAncestors | ✓ | ✓ | ✓ |
| FindPage | ✓ | ✓ | ✓ |
| UpdatePage | ✗ | ✓ | ✓ |
| ExpandDetails | ✓ | ✓ | ✓ |
| HideDetails | ⚠️ | ✓ | ✓ |
| MovePage | ✗ | ✓ | ✓ |
| RemovePage | ✗ | ✓ | ✓ |
| CreateDetailPage | ✗ | ✓ | ✓ |
| CreateContentsPage | ✗ | ✓ | ✓ |
| AddSegment | ✗ | ✗ | ✓ |
| RemoveSegment | ✗ | ✗ | ✓ |
| SetPermission | ✗ | ✗ | ✓ |

**重要约束**：
- **⚠️ HideDetails 特殊限制**：对于 `SystemSegment` 类型的 Segment，其 **root Page** 禁止执行 `HideDetails()` 操作
- 这确保 Agent 始终受系统提示词约束，不能通过隐藏来绕过
- 系统提示词 root Page 必须保持 `Expanded` 状态

## 设计要点

### 1. 统一权限检查入口

所有操作都通过 `checkPermission()` 方法进行权限检查，确保：
- 权限逻辑集中管理
- 易于维护和扩展
- 避免遗漏检查

### 2. 内部方法命名规范

ContextSystem 的内部方法统一使用 `Internal` 后缀：
- `updatePageInternal`
- `movePageInternal`
- `removePageInternal`
- `expandDetailsInternal`
- `hideDetailsInternal`

### 3. 错误处理

权限检查失败时返回明确的错误信息：
```go
if err := ac.checkPermission(pageIndex, "updatePage"); err != nil {
    return fmt.Errorf("permission denied: %w", err)
}
```

### 4. 系统提示词保护机制

为防止 Agent 绕过系统提示词约束，实施以下保护措施：

```go
// 1. 系统Segment root Page 默认必须为 Expanded
sysRoot, _ := NewContentsPage("System", "System prompts", "")
sysRoot.SetVisibility(Expanded)  // 强制展开

// 2. 权限检查中特殊处理
func (ac *AgentContext) checkPermission(pageIndex PageIndex, operation string) error {
    segment, _ := ac.system.getSegmentByPageIndexInternal(pageIndex)

    // 禁止隐藏系统提示词的 root Page
    if operation == "hideDetails" &&
       segment.GetType() == SystemSegment &&
       pageIndex == segment.GetRootIndex() {
        return fmt.Errorf("cannot hide system prompt root: agent must remain constrained")
    }

    // ... 其他权限检查
}
```

**保护的三个层次**：
1. **默认展开**：创建时强制设置 `Expanded` 状态
2. **禁止隐藏**：权限检查中拒绝 `hideDetails` 操作
3. **只读保护**：`ReadOnly` 权限禁止修改内容

## 典型使用流程

```go
// 1. 创建ContextSystem
contextSystem := NewContextSystem()

// 2. 创建并配置Segment
sysSeg := NewSegment("sys", "System", "System context", SystemSegment)
sysSeg.SetPermission(ReadOnly)
contextSystem.AddSegment(*sysSeg)  // 值传递，解引用

usrSeg := NewSegment("usr", "User", "User context", UserSegment)
usrSeg.SetPermission(ReadWrite)
contextSystem.AddSegment(*usrSeg)  // 值传递，解引用

// 3. 创建AgentContext代理
agentCtx := NewAgentContext(contextSystem)

// 4. Agent通过AgentContext操作

// 只读操作（对ReadOnly Segment也允许）
segments := agentCtx.ListSegments()

// 写操作（需要ReadWrite权限）
err := agentCtx.UpdatePage("usr-1", "新名称", "新描述")
if err != nil {
    // 权限不足或其他错误
    log.Println(err)
}

// 尝试修改系统Segment（会被拒绝）
err = agentCtx.UpdatePage("sys-1", "修改系统", "...")
// Error: permission denied: operation 'updatePage' on sys-1 requires higher permission

// 尝试隐藏系统提示词root Page（会被拒绝）
err = agentCtx.HideDetails("sys-0")
// Error: cannot hide system prompt root page sys-0: agent must remain constrained by system prompts
```

## 注意事项

1. **职责单一**：AgentContext 只负责权限检查和代理，不直接存储数据
2. **不可变系统**：ContextSystem 的引用在创建后不可变
3. **权限优先**：所有写操作必须先通过权限检查
4. **错误传播**：内部方法错误应直接传播给调用者
5. **并发安全**：AgentContext 本身不保证线程安全，如需并发应在外层加锁
6. **系统提示词保护**：
   - 系统 Segment 的 root Page 必须默认为 `Expanded` 状态
   - 禁止 Agent 对系统提示词 root Page 执行 `hideDetails()` 操作
   - 这是防止 Agent 绕过系统约束的关键安全机制
7. **Page 结构完整性**：
   - 除了 root page 外，所有 Page 都必须有父节点
   - `CreateDetailPage` 和 `CreateContentsPage` 都要求指定 `parentIndex`
   - ContextSystem 会验证父节点的存在性和类型
   - 这确保了树结构的完整性，防止孤儿节点
