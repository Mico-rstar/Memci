package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
			if segment.GetType() == SystemSegment {
				messageList.Append(message.System, wrappedContent)
			} else {
				messageList.Append(message.User, wrappedContent)
			}
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
		// DetailPage: ### [索引] 名称: 描述 [标记]
		builder.WriteString(fmt.Sprintf("%s [%s] %s", heading, pageIndex, pageName))
		if pageDesc != "" {
			builder.WriteString(fmt.Sprintf(": %s", pageDesc))
		}

		// 如果Expanded，显示detail内容
		if visibility == Expanded && p.GetDetail() != "" {
			// [Hide] 标记在外围，detail 内容用代码块包裹避免内部 markdown 语法冲突
			builder.WriteString("\n")
			builder.WriteString(fmt.Sprintf("[Hide]\n~~~\n%s\n~~~\n", p.GetDetail()))
		} else if visibility == Hidden && p.GetDetail() != "" {
			// Hidden 状态但有 detail 内容，显示 [Expand] 提示
			builder.WriteString(" ([Expand]...)")
		}
		builder.WriteString("\n")

	case *ContentsPage:
		// ContentsPage: ### [索引] 名称: 描述 [标记]
		builder.WriteString(fmt.Sprintf("%s [%s] %s", heading, pageIndex, pageName))
		if pageDesc != "" {
			builder.WriteString(fmt.Sprintf(": %s", pageDesc))
		}

		// 如果Expanded，递归渲染子节点
		if visibility == Expanded {
			builder.WriteString("\n")
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
			builder.WriteString("\n")
		} else {
			builder.WriteString("\n")
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
	// 跳过 SystemSegment，避免折叠系统提示词导致 agent 行为失控
	for _, segment := range segments {
		// 跳过系统段，系统提示词不应该被折叠
		if segment.GetType() == SystemSegment {
			continue
		}

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

// ExportToFile 将当前ContextWindow导出到文件
// 生成人类可读的markdown格式，包含所有Page的结构和内容
func (cw *ContextWindow) ExportToFile(outputDir string, turn int) (string, error) {
	// 确保输出目录存在
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// 生成文件名：context_snapshot_turn_<turn>_<timestamp>.md
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("context_snapshot_turn_%d_%s.md", turn, timestamp)
	filepath := filepath.Join(outputDir, filename)

	// 生成内容
	content := cw.generateHumanReadableContent(turn, timestamp)

	// 写入文件
	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write snapshot file: %w", err)
	}

	return filepath, nil
}

// generateHumanReadableContent 生成人类可读的ContextWindow内容
func (cw *ContextWindow) generateHumanReadableContent(turn int, timestamp string) string {
	var builder strings.Builder

	// 头部信息
	builder.WriteString("# ContextWindow Snapshot\n\n")
	builder.WriteString(fmt.Sprintf("**Turn**: %d\n", turn))
	builder.WriteString(fmt.Sprintf("**Timestamp**: %s\n\n", timestamp))
	builder.WriteString("---\n\n")

	// 获取所有Segment
	segments, err := cw.system.ListSegments()
	if err != nil {
		builder.WriteString(fmt.Sprintf("*Error listing segments: %v*\n\n", err))
		return builder.String()
	}

	// 渲染每个Segment
	for _, segment := range segments {
		rootIndex := segment.GetRootIndex()
		if rootIndex == "" {
			continue
		}

		// Segment信息
		builder.WriteString(fmt.Sprintf("## Segment: %s\n\n", segment.GetID()))
		builder.WriteString(fmt.Sprintf("**Type**: %s\n", segment.GetType()))
		builder.WriteString(fmt.Sprintf("**Permission**: %s\n\n", segment.GetPermission()))

		// 递归渲染页面树
		rootPage, err := cw.system.GetPage(rootIndex)
		if err != nil {
			builder.WriteString(fmt.Sprintf("*Error getting root page: %v*\n\n", err))
			continue
		}

		content := cw.renderPageForHuman(rootPage, 0)
		if content != "" {
			builder.WriteString(content)
		}

		builder.WriteString("\n")
	}

	// Token估算
	tokens, err := cw.EstimateTokens()
	if err == nil {
		builder.WriteString("---\n\n")
		builder.WriteString(fmt.Sprintf("**Estimated Tokens**: %d\n", tokens))
	}

	return builder.String()
}

// renderPageForHuman 为人类可读格式渲染页面（不包含markdown代码块包裹）
func (cw *ContextWindow) renderPageForHuman(page Page, depth int) string {
	var builder strings.Builder

	visibility := page.GetVisibility()
	lifecycle := page.GetLifecycle()

	// 只渲染Active状态的页面
	if lifecycle != Active {
		return ""
	}

	// 生成markdown标题级别
	headingLevel := depth + 3
	heading := strings.Repeat("#", headingLevel)

	// 格式：### [索引] 名称: 描述
	pageIndex := page.GetIndex()
	pageName := page.GetName()
	pageDesc := page.GetDescription()

	switch p := page.(type) {
	case *DetailPage:
		builder.WriteString(fmt.Sprintf("%s [%s] %s", heading, pageIndex, pageName))
		if pageDesc != "" {
			builder.WriteString(fmt.Sprintf(": %s", pageDesc))
		}

		// 状态标记
		if visibility == Expanded {
			builder.WriteString(" **[Expanded]**")
		} else {
			builder.WriteString(" **[Hidden]**")
		}

		// 如果Expanded，显示detail内容
		if visibility == Expanded && p.GetDetail() != "" {
			builder.WriteString("\n\n")
			builder.WriteString("```\n")
			builder.WriteString(p.GetDetail())
			builder.WriteString("\n```")
		}
		builder.WriteString("\n\n")

	case *ContentsPage:
		builder.WriteString(fmt.Sprintf("%s [%s] %s", heading, pageIndex, pageName))
		if pageDesc != "" {
			builder.WriteString(fmt.Sprintf(": %s", pageDesc))
		}

		children := p.GetChildren()
		if len(children) > 0 {
			builder.WriteString(fmt.Sprintf(" **(%d children)**", len(children)))
		}

		// 状态标记
		if visibility == Expanded {
			builder.WriteString(" **[Expanded]**")
		} else {
			builder.WriteString(" **[Hidden]**")
		}

		builder.WriteString("\n\n")

		// 如果Expanded，递归渲染子节点
		if visibility == Expanded {
			for _, childIndex := range children {
				childPage, err := cw.system.GetPage(childIndex)
				if err != nil {
					builder.WriteString(fmt.Sprintf("*Error: cannot get child %s*\n\n", childIndex))
					continue
				}

				childContent := cw.renderPageForHuman(childPage, depth+1)
				if childContent != "" {
					builder.WriteString(childContent)
				}
			}
		}
	}

	return builder.String()
}
