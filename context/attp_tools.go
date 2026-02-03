package context

import (
	"fmt"
	"strings"
)

// ATTPTools ATTP 协议召回工具
type ATTPTools struct {
	contextSystem *ContextSystem
}

// NewATTPTools 创建 ATTP 工具
func NewATTPTools(ctxSystem *ContextSystem) *ATTPTools {
	return &ATTPTools{
		contextSystem: ctxSystem,
	}
}

// RecallPageParams 召回 Page 参数
type RecallPageParams struct {
	PageIndex PageIndex `json:"page_index"`
}

// ListPagesParams 列出可用 Pages 参数
type ListPagesParams struct {
	Limit int `json:"limit,omitempty"` // 限制返回数量
}

// PageInfo Page 信息
type PageInfo struct {
	Index         PageIndex `json:"index"`
	Summary       string    `json:"summary"`
	RecallCount   int       `json:"recall_count"`
	LastRecallTurn int      `json:"last_recall_turn"`
}

// RecallPageResult 召回 Page 的结果
type RecallPageResult struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	PageSummary string `json:"page_summary,omitempty"`
}

// RecallPage 召回指定 Page
func (at *ATTPTools) RecallPage(params RecallPageParams) (RecallPageResult, error) {
	page, err := at.contextSystem.RecallPage(params.PageIndex)
	if err != nil {
		return RecallPageResult{
			Success: false,
			Message: fmt.Sprintf("无法召回 Page #%d: %v", params.PageIndex, err),
		}, nil
	}

	// 生成 Page 摘要
	summary := generatePageSummary(page)

	return RecallPageResult{
		Success:     true,
		Message:     fmt.Sprintf("成功召回 Page #%d", params.PageIndex),
		PageSummary: summary,
	}, nil
}

// ListPages 列出可用的 Pages（从 Contents Page）
func (at *ATTPTools) ListPages(params ListPagesParams) ([]PageInfo, error) {
	entries := at.contextSystem.ListArchivedPages()

	// 应用限制
	limit := params.Limit
	if limit <= 0 || limit > len(entries) {
		limit = len(entries)
	}

	result := make([]PageInfo, 0, limit)
	currentTurn := at.contextSystem.GetCurrentTurn()

	for i := 0; i < limit && i < len(entries); i++ {
		entry := entries[i]
		result = append(result, PageInfo{
			Index:         entry.Index,
			Summary:       entry.Summary,
			RecallCount:   entry.RecallCount,
			LastRecallTurn: currentTurn - entry.LastRecallTurn,
		})
	}

	return result, nil
}

// generatePageSummary 生成 Page 的摘要（用于召回后显示）
func generatePageSummary(page *Page) string {
	entries := page.Entries
	if len(entries) == 0 {
		return "(空 Page)"
	}

	var summary string
	if len(entries) > 0 {
		var parts []string
		for _, entry := range entries {
			parts = append(parts, fmt.Sprintf("[%s] %s", entry.Role(), truncateForSummary(entry.Content().String(), 100)))
		}
		summary = strings.Join(parts, "\n")
	}

	return summary
}

// truncateForSummary 截断字符串用于摘要显示
func truncateForSummary(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}

	return string(runes[:maxLen-3]) + "..."
}

// GetToolDefinitions 获取工具定义（用于系统提示词）
// 返回符合 ATTP 协议的 Python 函数签名定义
func (at *ATTPTools) GetToolDefinitions() string {
	return `
# 上下文召回工具

def recall_page(page_index: int) -> dict:
    """
    召回指定索引的历史对话页面到上下文窗口中。

    Args:
        page_index: 页面索引，从 list_pages() 工具获取

    Returns:
        dict: 包含以下字段的字典：
            - success: bool - 是否召回成功
            - message: str - 结果消息
            - page_summary: str - 页面内容摘要（仅成功时）

    Example:
        >>> result = recall_page(42)
        >>> if result['success']:
        ...     print(result['page_summary'])
        ... else:
        ...     print(f"召回失败: {result['message']}")
    """
    pass

def list_pages(limit: int = 10) -> list[dict]:
    """
    列出 Archive 中可用的历史对话页面。

    Args:
        limit: 返回结果的最大数量，默认为 10

    Returns:
        list[dict]: 页面信息列表，每个页面包含：
            - index: int - 页面索引
            - summary: str - 页面摘要
            - recall_count: int - 被召回次数
            - last_recall_turn: int - 最后召回的轮次

    Example:
        >>> pages = list_pages(limit=5)
        >>> for page in pages:
        ...     print(f"Page #{page['index']}: {page['summary']}")
        ...     print(f"  召回次数: {page['recall_count']}")
    """
    pass
`
}

// ExecuteRecallPage 执行召回 Page（供 ATTP 协议引擎调用）
func (at *ATTPTools) ExecuteRecallPage(pageIndex int) (map[string]interface{}, error) {
	result, err := at.RecallPage(RecallPageParams{PageIndex: PageIndex(pageIndex)})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success":      result.Success,
		"message":      result.Message,
		"page_summary": result.PageSummary,
	}, nil
}

// ExecuteListPages 执行列出 Pages（供 ATTP 协议引擎调用）
func (at *ATTPTools) ExecuteListPages(limit int) (map[string]interface{}, error) {
	pages, err := at.ListPages(ListPagesParams{Limit: limit})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"pages": pages,
	}, nil
}

// GetRecallStats 获取召回统计信息
func (at *ATTPTools) GetRecallStats() map[string]interface{} {
	entries := at.contextSystem.ListArchivedPages()

	totalRecalls := 0
	for _, entry := range entries {
		totalRecalls += entry.RecallCount
	}

	return map[string]interface{}{
		"total_archived_pages": len(entries),
		"total_recalls":        totalRecalls,
		"current_turn":         at.contextSystem.GetCurrentTurn(),
	}
}
