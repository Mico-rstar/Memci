package context

import (
	"fmt"
	"memci/llm"
	"memci/message"
	"sync"
	"time"
)

// ChapterType Chapter 类型
type ChapterType string

const (
	TypeSystemChapter  ChapterType = "system"
	TypeArchiveChapter ChapterType = "archive"
	TypeActiveChapter  ChapterType = "active"
)

// Chapter Chapter 接口
type Chapter interface {
	Type() ChapterType
	AddPage(page *Page, index PageIndex) error
	GetPage(index PageIndex) (*Page, error)
	GetPages() []*Page
	ToMessageList() *message.MessageList
	Len() int
}

// BaseChapter Chapter 的基础实现
type BaseChapter struct {
	chapterType ChapterType
	pages       map[PageIndex]*Page
	ordered     []PageIndex // 保持插入顺序
	mu          sync.RWMutex
}

// NewBaseChapter 创建基础 Chapter
func NewBaseChapter(chapterType ChapterType) *BaseChapter {
	return &BaseChapter{
		chapterType: chapterType,
		pages:       make(map[PageIndex]*Page),
		ordered:     make([]PageIndex, 0),
	}
}

// Type 返回 Chapter 类型
func (bc *BaseChapter) Type() ChapterType {
	return bc.chapterType
}

// AddPage 添加 Page
func (bc *BaseChapter) AddPage(page *Page, index PageIndex) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if _, exists := bc.pages[index]; exists {
		return fmt.Errorf("page with index %d already exists", index)
	}

	bc.pages[index] = page
	bc.ordered = append(bc.ordered, index)
	return nil
}

// GetPage 获取指定索引的 Page
func (bc *BaseChapter) GetPage(index PageIndex) (*Page, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	page, ok := bc.pages[index]
	if !ok {
		return nil, fmt.Errorf("page with index %d not found", index)
	}

	return page, nil
}

// GetPages 获取所有 Pages（按顺序）
func (bc *BaseChapter) GetPages() []*Page {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	pages := make([]*Page, 0, len(bc.ordered))
	for _, index := range bc.ordered {
		pages = append(pages, bc.pages[index])
	}

	return pages
}

// RemovePage 移除指定索引的 Page
func (bc *BaseChapter) RemovePage(index PageIndex) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if _, ok := bc.pages[index]; !ok {
		return fmt.Errorf("page with index %d not found", index)
	}

	delete(bc.pages, index)

	// 从 ordered 中移除
	for i, idx := range bc.ordered {
		if idx == index {
			bc.ordered = append(bc.ordered[:i], bc.ordered[i+1:]...)
			break
		}
	}

	return nil
}

// ToMessageList 转换为 MessageList
func (bc *BaseChapter) ToMessageList() *message.MessageList {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	msgList := message.NewMessageList()

	for _, index := range bc.ordered {
		page := bc.pages[index]
		// 直接将 Page 的 MessageList 添加进来
		pageMsgs := page.ToMessageList()
		msgList.AddMessageList(pageMsgs)
	}

	return msgList
}

// Len 返回 Page 数量
func (bc *BaseChapter) Len() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return len(bc.pages)
}

// ActiveChapter 管理最近 n 个 page（直接在上下文窗口中）
type ActiveChapter struct {
	*BaseChapter
	maxSize int // 最大 page 数量
}

// NewActiveChapter 创建 Active Chapter
func NewActiveChapter(maxSize int) *ActiveChapter {
	return &ActiveChapter{
		BaseChapter: NewBaseChapter(TypeActiveChapter),
		maxSize:     maxSize,
	}
}

// AddPage 添加 Page（超出容量时返回需要归档的 pages）
func (ac *ActiveChapter) AddPage(page *Page, index PageIndex) ([]*Page, error) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if _, exists := ac.pages[index]; exists {
		return nil, fmt.Errorf("page with index %d already exists", index)
	}

	ac.pages[index] = page
	ac.ordered = append(ac.ordered, index)

	// 检查是否超出容量
	if len(ac.pages) > ac.maxSize {
		// 返回需要归档的 pages（最旧的）
		toArchive := make([]*Page, 0)
		numToArchive := len(ac.pages) - ac.maxSize

		for i := 0; i < numToArchive; i++ {
			idx := ac.ordered[i]
			toArchive = append(toArchive, ac.pages[idx])
		}

		// 从 ActiveChapter 中移除
		for i := 0; i < numToArchive; i++ {
			idx := ac.ordered[0]
			delete(ac.pages, idx)
			ac.ordered = ac.ordered[1:]
		}

		return toArchive, nil
	}

	return nil, nil
}

// ArchiveChapter 管理已卸载的 page（保留 Contents Page）
type ArchiveChapter struct {
	mu           sync.RWMutex
	contentsPage *ContentsPage
	storage      PageStorage
	compactModel *llm.CompactModel
	unloadConfig UnloadConfig
}

// NewArchiveChapter 创建 Archive Chapter
func NewArchiveChapter(maxToken int, storage PageStorage, compactModel *llm.CompactModel) *ArchiveChapter {
	return &ArchiveChapter{
		contentsPage: NewContentsPage(maxToken),
		storage:      storage,
		compactModel: compactModel,
		unloadConfig: DefaultUnloadConfig(),
	}
}

// SetUnloadConfig 设置卸载配置
func (ac *ArchiveChapter) SetUnloadConfig(config UnloadConfig) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.unloadConfig = config
}

// Type 返回 Chapter 类型
func (ac *ArchiveChapter) Type() ChapterType {
	return TypeArchiveChapter
}

// AddPage 添加 Page 到 Archive（立即卸载到存储，生成 Contents Page 条目）
func (ac *ArchiveChapter) AddPage(page *Page, index PageIndex) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	// 1. 生成 Page 摘要
	pageSummary, entrySummaries, err := ac.summarizePage(page)
	if err != nil {
		return fmt.Errorf("failed to summarize page: %w", err)
	}

	// 2. 卸载到存储
	err = ac.storage.Save(page, index)
	if err != nil {
		return fmt.Errorf("failed to save page to storage: %w", err)
	}

	// 3. 添加到 Contents Page
	entry := PageEntry{
		Index:          index,
		Summary:        pageSummary,
		EntrySummaries: entrySummaries,
		RecallCount:    0,
		LastRecallTurn: ac.contentsPage.GetCurrentTurn(),
		CreatedTime:    ac.getCreatedTime(page),
		LastRecallTime: ac.getCreatedTime(page),
	}

	err = ac.contentsPage.AddEntry(entry)
	if err != nil {
		return err
	}

	return nil
}

// summarizePage 生成 Page 和 Entry 的摘要
func (ac *ArchiveChapter) summarizePage(page *Page) (string, []EntrySummary, error) {
	// 生成 Page 级别摘要
	pageSummary := ac.summarizePageContent(page)

	// 生成 Entry 级别摘要
	entrySummaries := make([]EntrySummary, 0)

	for _, entry := range page.Entries {
		summary, err := entry.Summarize(ac.compactModel)
		if err != nil {
			// 如果摘要失败，使用简单摘要
			summary = &EntrySummary{
				EntryID: entry.ID,
				Summary:  truncateString(entry.Content().String(), 100),
			}
		}
		entrySummaries = append(entrySummaries, *summary)
	}

	return pageSummary, entrySummaries, nil
}

// summarizePageContent 生成 Page 内容的摘要
func (ac *ArchiveChapter) summarizePageContent(page *Page) string {
	// 直接使用 Page 的 Summarize 方法
	return page.Summarize()
}

// getCreatedTime 获取 Page 的创建时间（从第一个 Entry 的时间戳）
func (ac *ArchiveChapter) getCreatedTime(page *Page) time.Time {
	// 简单实现：返回当前时间
	// TODO: 可以从 Page 的元数据中获取实际创建时间
	return time.Now()
}

// RecallPage 从存储召回 Page
func (ac *ArchiveChapter) RecallPage(index PageIndex) (*Page, error) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	// 1. 从存储加载 Page
	page, err := ac.storage.Load(index)
	if err != nil {
		return nil, fmt.Errorf("failed to load page from storage: %w", err)
	}

	// 2. 更新召回统计
	currentTurn := ac.contentsPage.GetCurrentTurn()
	err = ac.contentsPage.UpdateRecall(index, currentTurn)
	if err != nil {
		// Contents Page 中可能没有这个条目（已经被卸载）
		// 这是可以接受的，因为 Page 还在存储中
	}

	return page, nil
}

// ProcessUnload 处理卸载逻辑（在每轮对话后调用）
func (ac *ArchiveChapter) ProcessUnload() ([]PageIndex, error) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	toUnload := ac.contentsPage.GetEntriesToUnload(ac.unloadConfig)

	// 从 Contents Page 移除，但保留在存储中
	for _, index := range toUnload {
		err := ac.contentsPage.RemoveEntry(index)
		if err != nil {
			return toUnload, fmt.Errorf("failed to remove entry %d: %w", index, err)
		}
	}

	return toUnload, nil
}

// GetPage 获取指定索引的 Page（从存储加载）
func (ac *ArchiveChapter) GetPage(index PageIndex) (*Page, error) {
	return ac.RecallPage(index)
}

// GetPages 获取所有 Pages（从 Contents Page）
func (ac *ArchiveChapter) GetPages() []*Page {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	// ArchiveChapter 不直接返回所有 Pages
	// 而是通过 Contents Page 提供摘要信息
	return nil
}

// ToMessageList 转换为 MessageList（包含 Contents Page）
func (ac *ArchiveChapter) ToMessageList() *message.MessageList {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	return ac.contentsPage.ToMessageList()
}

// Len 返回 Contents Page 中的条目数量
func (ac *ArchiveChapter) Len() int {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.contentsPage.Len()
}

// GetContentsPage 获取 Contents Page
func (ac *ArchiveChapter) GetContentsPage() *ContentsPage {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.contentsPage
}

// AdvanceTurn 前进到下一轮对话
func (ac *ArchiveChapter) AdvanceTurn() {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.contentsPage.AdvanceTurn()
}

// ListContentsEntries 列出 Contents Page 中的所有条目
func (ac *ArchiveChapter) ListContentsEntries() []PageEntry {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.contentsPage.ListEntries()
}

// truncateString 截断字符串到指定长度
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	// 尝试在空格处截断
	for i := maxLen - 10; i < maxLen && i < len(s); i++ {
		if s[i] == ' ' {
			return s[:i] + "..."
		}
	}

	return s[:maxLen-3] + "..."
}
