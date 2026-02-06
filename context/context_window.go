package context

import (
	"fmt"
	"strings"

	"memci/message"
)

// ContextWindow 页面树渲染层
type ContextWindow struct {
	system *ContextSystem
}

// NewContextWindow 创建新的ContextWindow
func NewContextWindow(system *ContextSystem) *ContextWindow {
	return &ContextWindow{
		system: system,
	}
}

// GenerateMessageList 生成发送给模型的MessageList
func (cw *ContextWindow) GenerateMessageList() (*message.MessageList, error) {
	// 获取所有Segment
	segments, err := cw.system.ListSegments()
	if err != nil {
		return nil, err
	}

	// 按顺序遍历每个Segment的root page
	messageList := message.NewMessageList()
	for _, segment := range segments {
		rootIndex := segment.GetRootIndex()
		if rootIndex == "" {
			continue
		}

		// 递归渲染从root开始的页面树
		rootPage, err := cw.system.GetPage(rootIndex)
		if err != nil {
			return nil, fmt.Errorf("failed to get root page %s: %w", rootIndex, err)
		}

		content := cw.renderPageRecursive(rootPage, 0)
		if content != "" {
			// 在最外层包裹 ```markdown ... ``` 提醒Agent这是markdown格式
			wrappedContent := fmt.Sprintf("```markdown\n%s\n```", content)
			// 为每个Segment创建一个消息节点
			messageList.Append(message.System, wrappedContent)
		}
	}

	return messageList, nil
}

// renderPageRecursive 递归渲染页面树
// depth: 当前层级深度，用于markdown标题级别（1表示根层级）
func (cw *ContextWindow) renderPageRecursive(page Page, depth int) string {
	var builder strings.Builder

	visibility := page.GetVisibility()
	lifecycle := page.GetLifecycle()

	// 只渲染Active状态的页面
	if lifecycle != Active {
		return ""
	}

	// 生成markdown标题级别（depth + 1个#）
	headingLevel := depth + 1
	heading := strings.Repeat("#", headingLevel)

	// 格式：### [索引] 名称: 描述
	pageIndex := page.GetIndex()
	pageName := page.GetName()
	pageDesc := page.GetDescription()

	switch p := page.(type) {
	case *DetailPage:
		// DetailPage: ### [索引] 名称: 描述
		builder.WriteString(fmt.Sprintf("%s [%s] %s", heading, pageIndex, pageName))
		if pageDesc != "" {
			builder.WriteString(fmt.Sprintf(": %s", pageDesc))
		}
		builder.WriteString("\n")

		// 如果Expanded，显示detail内容
		if visibility == Expanded && p.GetDetail() != "" {
			// [Hide] 标记在外围，detail 内容用代码块包裹避免内部 markdown 语法冲突
			builder.WriteString(fmt.Sprintf("[Hide]\n~~~\n%s\n~~~\n", p.GetDetail()))
		} else if visibility == Hidden && p.GetDetail() != "" {
			// Hidden 状态但有 detail 内容，显示 [Expand] 提示
			builder.WriteString(" ([Expand]...)")
		}

	case *ContentsPage:
		// ContentsPage: ### [索引] 名称: 描述
		builder.WriteString(fmt.Sprintf("%s [%s] %s", heading, pageIndex, pageName))
		if pageDesc != "" {
			builder.WriteString(fmt.Sprintf(": %s", pageDesc))
		}
		builder.WriteString("\n")

		// 如果Expanded，递归渲染子节点
		if visibility == Expanded {
			children := p.GetChildren()
			if len(children) > 0 {
				for _, childIndex := range children {
					childPage, err := cw.system.GetPage(childIndex)
					if err != nil {
						// 子页面不存在，跳过
						continue
					}

					childContent := cw.renderPageRecursive(childPage, depth+1)
					if childContent != "" {
						builder.WriteString(childContent)
					}
				}
			}
		} else if visibility != Expanded && len(p.GetChildren()) > 0 {
			// Hidden 状态但有子页面，显示 [Expand] 提示
			builder.WriteString(fmt.Sprintf(" (%d [Expand]...)", len(p.GetChildren())))
		}
	}

	return builder.String()
}

// EstimateTokens 估算当前MessageList的token数量
// 这是一个简化的估算，实际应用中应该使用更精确的tokenizer
func (cw *ContextWindow) EstimateTokens() (int, error) {
	messageList, err := cw.GenerateMessageList()
	if err != nil {
		return 0, err
	}

	// 简单估算：中文字符约1.5 tokens，英文单词约0.25 token
	// 这里使用粗略估算：每个字符约0.3 token
	totalChars := 0
	nodes := messageList.GetNode()
	for nodes != nil {
		totalChars += len(nodes.GetMsg().Content.String())
		nodes = nodes.Next()
	}

	// 粗略估算：每3个字符约1个token
	return int(float64(totalChars) / 3.0), nil
}

// AutoCollapse 自动折叠以适应token限制
func (cw *ContextWindow) AutoCollapse(maxTokens int) ([]PageIndex, error) {
	// 这是一个简化的实现，实际应用中需要更智能的折叠策略
	// 策略：优先折叠最早的DetailPage

	var collapsedPages []PageIndex
	currentTokens, err := cw.EstimateTokens()
	if err != nil {
		return nil, err
	}

	if currentTokens <= maxTokens {
		return collapsedPages, nil
	}

	// 获取所有Segment
	segments, err := cw.system.ListSegments()
	if err != nil {
		return nil, err
	}

	// 遍历所有页面，优先折叠DetailPage
	for _, segment := range segments {
		rootIndex := segment.GetRootIndex()
		if rootIndex == "" {
			continue
		}

		pagesToCollapse := cw.findPagesToCollapse(rootIndex)
		for _, pageIndex := range pagesToCollapse {
			collapsedPages = append(collapsedPages, pageIndex)

			// 折叠该页面
			if err := cw.HideDetails(pageIndex); err != nil {
				return nil, fmt.Errorf("failed to collapse page %s: %w", pageIndex, err)
			}

			// 重新估算token
			currentTokens, _ = cw.EstimateTokens()
			if currentTokens <= maxTokens {
				return collapsedPages, nil
			}
		}
	}

	return collapsedPages, nil
}

// findPagesToCollapse 查找可以折叠的页面（DFS遍历）
func (cw *ContextWindow) findPagesToCollapse(rootIndex PageIndex) []PageIndex {
	var pagesToCollapse []PageIndex

	var dfs func(pageIndex PageIndex)
	dfs = func(pageIndex PageIndex) {
		page, err := cw.system.GetPage(pageIndex)
		if err != nil {
			return
		}

		// 只处理Expanded状态的页面
		if page.GetVisibility() != Expanded {
			return
		}

		switch p := page.(type) {
		case *DetailPage:
			// DetailPage可以折叠
			pagesToCollapse = append(pagesToCollapse, pageIndex)

		case *ContentsPage:
			// ContentsPage：先处理子节点，再考虑自己
			children := p.GetChildren()
			for _, childIndex := range children {
				dfs(childIndex)
			}
			// 如果折叠所有子节点后仍超出限制，也可以折叠ContentsPage本身
			// 但这里简化处理，只折叠DetailPage
		}
	}

	dfs(rootIndex)
	return pagesToCollapse
}

// HideDetails 隐藏Page详情（代理到ContextSystem内部方法）
func (cw *ContextWindow) HideDetails(pageIndex PageIndex) error {
	return cw.system.hideDetailsInternal(pageIndex)
}
