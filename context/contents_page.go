package context

import (
	"fmt"
	"memci/message"
	"strings"
	"sync"
	"time"
)

// PageIndex Page 的唯一索引
type PageIndex int

// PageEntry Page 中的一个条目（用于归档的 Entry 元数据）
type PageEntry struct {
	Index          PageIndex      // Page 索引
	Summary        string         // Page 整体摘要
	EntrySummaries []EntrySummary // 每个 Entry 的摘要
	RecallCount    int            // 被召回次数
	LastRecallTurn int            // 最后召回的轮次
	CreatedTime    time.Time      // 创建时间
	LastRecallTime time.Time      // 最后召回时间
}

// UnloadConfig 卸载策略配置
type UnloadConfig struct {
	MinRecallTurns   int     // m 轮对话后召回次数为 0 则卸载
	StaleRecallTurns int     // p 轮对话后召回次数不变则卸载
	MaxTokenRatio    float64 // 达到最大 token 的比例时按 LRU 卸载
}

// DefaultUnloadConfig 返回默认的卸载配置
func DefaultUnloadConfig() UnloadConfig {
	return UnloadConfig{
		MinRecallTurns:   10,
		StaleRecallTurns: 20,
		MaxTokenRatio:    0.9,
	}
}

// ContentsPage 元数据页，提供被归档 Page 的目录和索引
// 不同于 Page（存储实际内容），ContentsPage 只存储元数据用于生成目录
type ContentsPage struct {
	mu          sync.RWMutex
	MaxToken    int
	Entries     []PageEntry   // PageEntry 是元数据，不是 Entry
	CurrentTurn int
	nextIndex   PageIndex
}

// NewContentsPage 创建新的元数据页
func NewContentsPage(maxToken int) *ContentsPage {
	return &ContentsPage{
		MaxToken:  maxToken,
		Entries:   make([]PageEntry, 0),
		nextIndex: 1, // 从 1 开始
	}
}

// AddEntry 添加条目到 Contents Page
func (cp *ContentsPage) AddEntry(entry PageEntry) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	// 检查是否已存在
	for _, e := range cp.Entries {
		if e.Index == entry.Index {
			return fmt.Errorf("entry with index %d already exists", entry.Index)
		}
	}

	cp.Entries = append(cp.Entries, entry)
	return nil
}

// RemoveEntry 从 Contents Page 移除条目
func (cp *ContentsPage) RemoveEntry(index PageIndex) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	for i, entry := range cp.Entries {
		if entry.Index == index {
			cp.Entries = append(cp.Entries[:i], cp.Entries[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("entry with index %d not found", index)
}

// GetEntry 获取指定索引的条目
func (cp *ContentsPage) GetEntry(index PageIndex) (PageEntry, bool) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	for _, entry := range cp.Entries {
		if entry.Index == index {
			return entry, true
		}
	}

	return PageEntry{}, false
}

// UpdateRecall 更新召回统计
func (cp *ContentsPage) UpdateRecall(index PageIndex, currentTurn int) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	for i := range cp.Entries {
		if cp.Entries[i].Index == index {
			cp.Entries[i].RecallCount++
			cp.Entries[i].LastRecallTurn = currentTurn
			cp.Entries[i].LastRecallTime = time.Now()
			return nil
		}
	}

	return fmt.Errorf("entry with index %d not found", index)
}

// ShouldUnload 检查单个条目是否应该卸载
func (cp *ContentsPage) ShouldUnload(entry PageEntry, config UnloadConfig) bool {
	turnsSinceLastRecall := cp.CurrentTurn - entry.LastRecallTurn

	// 条件1: m 轮对话后召回次数为 0
	if entry.RecallCount == 0 && turnsSinceLastRecall >= config.MinRecallTurns {
		return true
	}

	// 条件3: p 轮对话后召回次数没变过（假设初始创建时 LastRecallTurn = CreatedTurn）
	if turnsSinceLastRecall >= config.StaleRecallTurns {
		return true
	}

	return false
}

// GetEntriesToUnload 根据配置获取需要卸载的条目列表
func (cp *ContentsPage) GetEntriesToUnload(config UnloadConfig) []PageIndex {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	// 首先检查是否超过 MaxToken
	if !cp.isOverMaxToken(config.MaxTokenRatio) {
		// 没有超过 token 限制，只检查召回条件
		return cp.getEntriesByRecallCondition(config)
	}

	// 超过 token 限制，按 LRU 卸载
	return cp.getEntriesByLRU()
}

// isOverMaxToken 检查是否超过最大 token 限制
func (cp *ContentsPage) isOverMaxToken(ratio float64) bool {
	estimatedTokens := cp.estimateTokens()
	return estimatedTokens > int(float64(cp.MaxToken)*ratio)
}

// estimateTokens 估算当前 Contents Page 的 token 数
func (cp *ContentsPage) estimateTokens() int {
	totalChars := 0
	for _, entry := range cp.Entries {
		totalChars += len(entry.Summary)
		for _, es := range entry.EntrySummaries {
			totalChars += len(es.Summary)
		}
	}
	// 粗略估算：token 数 ≈ 字符数 / 4
	return totalChars / 4
}

// getEntriesByRecallCondition 根据召回条件获取需要卸载的条目
func (cp *ContentsPage) getEntriesByRecallCondition(config UnloadConfig) []PageIndex {
	toUnload := make([]PageIndex, 0)

	for _, entry := range cp.Entries {
		if cp.ShouldUnload(entry, config) {
			toUnload = append(toUnload, entry.Index)
		}
	}

	return toUnload
}

// getEntriesByLRU 按 LRU 策略获取需要卸载的条目
func (cp *ContentsPage) getEntriesByLRU() []PageIndex {
	// 按最后召回时间排序，最久未使用的优先
	entries := make([]PageEntry, len(cp.Entries))
	copy(entries, cp.Entries)

	// 简单冒泡排序（按最后召回时间）
	for i := 0; i < len(entries)-1; i++ {
		for j := 0; j < len(entries)-1-i; j++ {
			if entries[j].LastRecallTime.After(entries[j+1].LastRecallTime) {
				entries[j], entries[j+1] = entries[j+1], entries[j]
			}
		}
	}

	// 计算需要卸载的数量
	// 持续删除直到低于 MaxToken 的 80%
	toUnload := make([]PageIndex, 0)
	tempEntries := make([]PageEntry, len(cp.Entries))
	copy(tempEntries, cp.Entries)

	for _, entry := range entries {
		// 移除这个条目
		for i, e := range tempEntries {
			if e.Index == entry.Index {
				tempEntries = append(tempEntries[:i], tempEntries[i+1:]...)
				break
			}
		}
		toUnload = append(toUnload, entry.Index)

		// 重新估算 token
		totalChars := 0
		for _, e := range tempEntries {
			totalChars += len(e.Summary)
			for _, es := range e.EntrySummaries {
				totalChars += len(es.Summary)
			}
		}
		if totalChars/4 < int(float64(cp.MaxToken)*0.8) {
			break
		}
	}

	return toUnload
}

// AdvanceTurn 前进到下一轮对话
func (cp *ContentsPage) AdvanceTurn() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.CurrentTurn++
}

// GetCurrentTurn 获取当前轮次
func (cp *ContentsPage) GetCurrentTurn() int {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return cp.CurrentTurn
}

// NextIndex 生成下一个可用的索引
func (cp *ContentsPage) NextIndex() PageIndex {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	idx := cp.nextIndex
	cp.nextIndex++
	return idx
}

// Len 返回条目数量
func (cp *ContentsPage) Len() int {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return len(cp.Entries)
}

// ToMessageList 转换为 MessageList（用于放入上下文窗口）
func (cp *ContentsPage) ToMessageList() *message.MessageList {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	msgList := message.NewMessageList()

	if len(cp.Entries) == 0 {
		return msgList
	}

	// 构建 Contents Page 的文本表示
	var builder strings.Builder
	builder.WriteString("# 历史对话目录\n\n")

	for _, entry := range cp.Entries {
		builder.WriteString(fmt.Sprintf("## [Page #%d]\n", entry.Index))
		builder.WriteString(fmt.Sprintf("**摘要**: %s\n", entry.Summary))
		if len(entry.EntrySummaries) > 0 {
			builder.WriteString("**详细内容**:\n")
			for _, es := range entry.EntrySummaries {
				builder.WriteString(fmt.Sprintf("  - %s: %s\n", es.EntryID, es.Summary))
			}
		}
	}

	builder.WriteString("\n你可以使用 `recall_page(page_index)` 工具来召回任何页面到上下文窗口中。\n")

	msgList.AddMessage(message.System, builder.String())

	return msgList
}

// ListEntries 列出所有条目的信息
func (cp *ContentsPage) ListEntries() []PageEntry {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	entries := make([]PageEntry, len(cp.Entries))
	copy(entries, cp.Entries)
	return entries
}

// Clear 清空所有条目
func (cp *ContentsPage) Clear() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.Entries = make([]PageEntry, 0)
}
