# ContextWindow 设计文档

## 概述

ContextWindow 负责将 Page Tree 渲染为 MessageList，这是最终发送给模型的内容。

## 核心职责

1. **遍历 Page Tree**：按 Segment 顺序遍历 Active 的 Page
2. **根据状态渲染**：根据 Page 的可见性决定渲染策略
3. **生成 MessageList**：将渲染结果组装成 MessageList

## ContextWindow 结构体

```go
// ContextWindow 上下文窗口
type ContextWindow struct {
    system *ContextSystem  // 持有 ContextSystem 的引用
}

// NewContextWindow 创建新的上下文窗口
func NewContextWindow(system *ContextSystem) *ContextWindow
```

## 核心 API

```go
// GenerateMessageList 生成发送给模型的 MessageList
func (cw *ContextWindow) GenerateMessageList() (*MessageList, error)

// RenderPage 渲染单个 Page 为消息内容
func (cw *ContextWindow) RenderPage(page Page) string
```

## 渲染规则

### Page 状态与渲染策略

| Page 状态 | Visibility | 渲染结果 | 说明 |
|-----------|-----------|----------|------|
| Active | Expanded | 完整内容 | DetailPage 显示 detail，ContentsPage 显示所有子节点 |
| Active | Hidden | 仅摘要 | DetailPage 显示 description，ContentsPage 不展开子节点 |
| HotArchived | - | 不渲染 | 不在上下文窗口内 |
| ColdArchived | - | 不渲染 | 已卸载出系统 |

### 渲染示例

**Page Tree (逻辑结构)**：
```
📁 sys-0 (ContentsPage) [Expanded]
├─ 📄 sys-1 (DetailPage) [Expanded]
│   └─ detail: "你是一个AI助手..."
📁 usr-0 (ContentsPage) [Expanded]
├─ 📁 usr-1 (ContentsPage) [Hidden]
│   └─ description: "关于Go的讨论"
└─ 📄 usr-2 (DetailPage) [Expanded]
    └─ detail: "如何实现goroutine？"
```

**MessageList (渲染结果)**：
```go
MessageList {
    Head: &MessageNode {
        Content: "你是一个AI助手...",
    },
    Tail: &MessageNode {
        Content: "关于Go的讨论\n\n如何实现goroutine？...",
    },
}
```

## 实现示例

```go
func (cw *ContextWindow) GenerateMessageList() (*MessageList, error) {
    messageList := message.NewMessageList()

    // 1. 遍历所有 Segment（按添加顺序）
    segments := cw.system.ListSegments()
    for _, segment := range segments {
        // 2. 获取 Segment 的 root Page
        rootIndex := segment.GetRootIndex()
        if rootIndex == "" {
            continue
        }

        rootPage, err := cw.system.GetPage(rootIndex)
        if err != nil {
            continue
        }

        // 3. 递归渲染 Page Tree
        if rootPage.GetLifecycle() == Active {
            content := cw.renderPageRecursive(rootPage)
            if content != "" {
                messageList.Append(content)
            }
        }
    }

    return messageList, nil
}

// renderPageRecursive 递归渲染 Page 及其子节点
func (cw *ContextWindow) renderPageRecursive(page Page) string {
    // 非状态检查
    if page.GetLifecycle() != Active {
        return ""
    }

    var builder strings.Builder

    switch p := page.(type) {
    case *ContentsPage:
        // ContentsPage 渲染逻辑
        cw.renderContentsPage(p, &builder, 0)

    case *DetailPage:
        // DetailPage 渲染逻辑
        cw.renderDetailPage(p, &builder)
    }

    return builder.String()
}

// renderContentsPage 渲染 ContentsPage
func (cw *ContextWindow) renderContentsPage(page *ContentsPage, builder *strings.Builder, depth int) {
    // 添加目录标题
    if depth > 0 {
        builder.WriteString(fmt.Sprintf("## %s\n", page.GetDescription()))
    }

    // 根据 Visibility 决定是否展开子节点
    if page.GetVisibility() == Expanded {
        for _, childIndex := range page.GetChildren() {
            child, err := cw.system.GetPage(childIndex)
            if err != nil {
                continue
            }
            content := cw.renderPageRecursive(child)
            builder.WriteString(content)
        }
    } else {
        // Hidden 状态：只显示摘要，不展开子节点
        builder.WriteString(fmt.Sprintf("[%d个子页面]\n", len(page.GetChildren())))
    }
}

// renderDetailPage 渲染 DetailPage
func (cw *ContextWindow) renderDetailPage(page *DetailPage, builder *strings.Builder) {
    if page.GetVisibility() == Expanded {
        // Expanded: 显示完整 detail
        builder.WriteString(page.GetDetail())
    } else {
        // Hidden: 只显示 description
        builder.WriteString(page.GetDescription())
    }
}
```

## 渲染格式

### 格式化规则

```go
// 根节点（Segment root）不添加标题
// 子节点按层级添加标题
depth 0: (无标题)
depth 1: ## 标题
depth 2: ### 标题
depth 3: #### 标题
```

**示例**：

```
你是一个AI助手...              // sys-0 Expanded (root，无标题)

## Go语言讨论                  // usr-1 Expanded (depth 1)
### goroutine原理             // usr-1-1 Expanded (depth 2)
...详细内容...

## Python问题                  // usr-2 Expanded (depth 1)
...详细内容...
```

## Token 计算

ContextWindow 负责计算当前 MessageList 的 token 数量：

```go
// EstimateTokens 估算当前 MessageList 的 token 数量
func (cw *ContextWindow) EstimateTokens(messageList *MessageList) int

// GetTokenCount 获取当前 token 统计
func (cw *ContextWindow) GetTokenCount() int
```

## 上下文窗口管理

当 token 数量接近上限时，ContextWindow 应该：

1. **自动折叠**：将 Expanded 的 Page 改为 Hidden
2. **优先级策略**：
   - 保留系统提示词（sys Segment）始终 Expanded
   - 最近的内容优先保持 Expanded
   - 历史内容优先折叠
3. **通知机制**：告知 Agent 哪些 Page 被折叠

```go
// AutoCollapse 自动折叠以适应 token 限制
func (cw *ContextWindow) AutoCollapse(maxTokens int) ([]PageIndex, error)

// 返回被折叠的 Page 索引列表
```

## 设计要点

### 1. 渲染 vs 存储

- **ContextSystem**: 存储 Page Tree（逻辑结构）
- **ContextWindow**: 渲染为 MessageList（实际发送内容）
- **分离关注点**: 存储关注组织，渲染关注格式化

### 2. 增量渲染

**问题**: 每次都重新渲染整个树效率低

**解决方案**: 缓存 + 增量更新
```go
type ContextWindow struct {
    system    *ContextSystem
    cache     map[PageIndex]string  // 缓存渲染结果
    cacheDirty bool                   // 脏标记
}

func (cw *ContextWindow) GenerateMessageList() (*MessageList, error) {
    if !cw.cacheDirty {
        return cw.getCachedMessageList()
    }
    // 重新渲染...
}
```

### 3. Token 优化策略

```go
// 根据容量自动调整可见性
func (cw *ContextWindow) AdjustVisibility() {
    for _, segment := range cw.system.ListSegments() {
        if segment.GetMaxCapacity() > 0 {
            currentTokens := cw.calculateSegmentTokens(segment)
            if currentTokens > segment.GetMaxCapacity() {
                cw.collapseLeastImportantPages(segment)
            }
        }
    }
}
```

### 4. 格式化灵活性

```go
// Renderer 渲染器接口（支持不同格式）
type Renderer interface {
    RenderDetailPage(page *DetailPage) string
    RenderContentsPage(page *ContentsPage) string
}

// MarkdownRenderer Markdown 格式渲染器
type MarkdownRenderer struct{}

// PlainTextRenderer 纯文本格式渲染器
type PlainTextRenderer struct{}

// ContextWindow 支持自定义渲染器
type ContextWindow struct {
    system   *ContextSystem
    renderer Renderer
}
```

## 与其他组件的关系

### ContextWindow vs ContextSystem

| 特性 | ContextWindow | ContextSystem |
|------|---------------|---------------|
| 职责 | 渲染 MessageList | 存储 Page Tree |
| 数据结构 | MessageList | Page + Segment |
| 输出 | 发送给模型 | 给 Agent 操作 |

### ContextWindow vs AgentContext

| 特性 | ContextWindow | AgentContext |
|------|---------------|--------------|
| 职责 | 渲染 | 权限控制 |
| 调用时机 | 发送消息前 | Agent 操作时 |
| 依赖 | 只读 ContextSystem | 代理 ContextSystem |

## 使用流程

```go
// 1. 创建 ContextSystem
contextSystem := NewContextSystem()
sysSeg := NewSegment("sys", "System", "...", SystemSegment)
sysSeg.SetPermission(ReadOnly)
contextSystem.AddSegment(*sysSeg)

// 2. 创建 ContextWindow
contextWindow := NewContextWindow(contextSystem)

// 3. 添加 Page
sysRoot := NewContentsPage("System", "系统提示词", "")
sysRoot.SetVisibility(Expanded)
sysRoot.SetIndex("sys-0")
sysSeg.SetRootIndex(sysRoot.GetIndex())
contextSystem.AddPage(sysRoot)

// 4. 生成 MessageList
messageList, err := contextWindow.GenerateMessageList()
if err != nil {
    return err
}

// 5. 发送给模型
response, err := llm.Send(messageList)
```

## 注意事项

1. **只读操作**: ContextWindow 不修改 Page Tree，只读取
2. **性能优化**: 大量 Page 时考虑增量渲染
3. **Token 限制**: 自动折叠以适应模型上下文窗口
4. **格式统一**: 使用一致的渲染格式（如 Markdown）
5. **错误处理**: 单个 Page 渲染失败不应影响整体
