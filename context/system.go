package context

import (
	"fmt"
	"memci/llm"
	"memci/message"
	"sync"
)

// SegmentType Segment 类型
type SegmentType string

const (
	TypeSystemSegment SegmentType = "system"
	TypeCommonSegment SegmentType = "common"
)

// Segment Segment 接口
type Segment interface {
	Type() SegmentType
	GetMessageList() *message.MessageList
}

// SystemChapter 系统提示词 Chapter
type SystemChapter struct {
	*BaseChapter
	extraTools string // 额外的工具描述（如 ATTP 召回工具）
}

// NewSystemChapter 创建系统 Chapter
func NewSystemChapter() *SystemChapter {
	return &SystemChapter{
		BaseChapter: NewBaseChapter(TypeSystemChapter),
		extraTools:  "",
	}
}

// SetExtraTools 设置额外的工具描述
func (sc *SystemChapter) SetExtraTools(tools string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.extraTools = tools
}

// GetExtraTools 获取额外的工具描述
func (sc *SystemChapter) GetExtraTools() string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.extraTools
}

// AddSystemPage 添加系统提示词 Page
func (sc *SystemChapter) AddSystemPage(page *Page, index PageIndex) error {
	return sc.BaseChapter.AddPage(page, index)
}

// ToMessageList 转换为 MessageList（包含 extraTools）
func (sc *SystemChapter) ToMessageList() *message.MessageList {
	sc.mu.RLock()
	defer sc.mu.Unlock()

	msgList := message.NewMessageList()

	// 1. 添加所有系统 Page
	for _, index := range sc.ordered {
		page := sc.pages[index]
		pageMsgs := page.ToMessageList()
		for _, msg := range pageMsgs.Msgs {
			msgList.AddMessageContent(msg.Role, msg.Content)
		}
	}

	// 2. 如果有 extraTools，追加到最后一条系统消息
	if sc.extraTools != "" {
		// 获取最后一条消息，如果是系统消息，追加工具描述
		if len(msgList.Msgs) > 0 {
			lastMsg := msgList.Msgs[len(msgList.Msgs)-1]
			if lastMsg.Role == message.System {
				// 更新最后一条系统消息，追加工具描述
				lastMsg.Content = message.NewContentString(
					lastMsg.Content.String() + "\n\n" + sc.extraTools,
				)
			}
		}
	}

	return msgList
}

// SystemSegment 系统提示词 Segment
type SystemSegment struct {
	mu            sync.RWMutex
	systemChapter *SystemChapter
}

// NewSystemSegment 创建 System Segment
func NewSystemSegment(systemPrompt string) *SystemSegment {
	ss := &SystemSegment{
		systemChapter: NewSystemChapter(),
	}

	// 如果有系统提示词，创建一个 Page 存储
	if systemPrompt != "" {
		page := NewPage(0, "system", -1, "System prompt", nil)
		page.AddEntry(NewEntry(message.System, message.NewContentString(systemPrompt)))
		ss.systemChapter.AddSystemPage(page, 0)
	}

	return ss
}

// NewSystemSegmentWithEntries 从 EntryList 创建 System Segment
func NewSystemSegmentWithEntries(entries []*Entry) *SystemSegment {
	ss := &SystemSegment{
		systemChapter: NewSystemChapter(),
	}

	if len(entries) == 0 {
		return ss
	}

	// 为每个系统 Entry 创建一个 Page
	for i, entry := range entries {
		if entry.Role == message.System {
			page := NewPage(PageIndex(i), "system", -1, "", nil)
			page.AddEntry(entry)
			ss.systemChapter.AddSystemPage(page, PageIndex(i))
		}
	}

	return ss
}

// NewSystemSegmentWithMessages 从消息创建 System Segment（便利方法）
func NewSystemSegmentWithMessages(msgs []message.Message) *SystemSegment {
	entries := make([]*Entry, 0, len(msgs))
	for _, msg := range msgs {
		entries = append(entries, NewEntryFromMessage(msg))
	}
	return NewSystemSegmentWithEntries(entries)
}

// Type 返回 Segment 类型
func (ss *SystemSegment) Type() SegmentType {
	return TypeSystemSegment
}

// SetExtraTools 设置额外的工具描述
func (ss *SystemSegment) SetExtraTools(tools string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.systemChapter.SetExtraTools(tools)
}

// GetMessageList 获取 MessageList
func (ss *SystemSegment) GetMessageList() *message.MessageList {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.systemChapter.ToMessageList()
}

// CommonSegment 用户交互上下文 Segment
type CommonSegment struct {
	mu             sync.RWMutex
	activeChapter  *ActiveChapter
	archiveChapter *ArchiveChapter
	turnCounter    int // 对话轮次计数器
}

// NewCommonSegment 创建 Common Segment
func NewCommonSegment(
	activeSize int,
	archiveMaxToken int,
	storage PageStorage,
	compactModel *llm.CompactModel,
) *CommonSegment {
	return &CommonSegment{
		activeChapter:  NewActiveChapter(activeSize),
		archiveChapter: NewArchiveChapter(archiveMaxToken, storage, compactModel),
		turnCounter:    0,
	}
}

// Type 返回 Segment 类型
func (cs *CommonSegment) Type() SegmentType {
	return TypeCommonSegment
}

// AddPage 添加 Page 到 Common Segment
func (cs *CommonSegment) AddPage(page *Page, index PageIndex) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// 尝试添加到 ActiveChapter
	toArchive, err := cs.activeChapter.AddPage(page, index)
	if err != nil {
		return fmt.Errorf("failed to add page to active chapter: %w", err)
	}

	// 如果有需要归档的 pages，添加到 ArchiveChapter
	for i, archPage := range toArchive {
		// 使用旧的索引（从 ActiveChapter 中移除的最旧的）
		archIndex := PageIndex(index - PageIndex(len(toArchive)) + PageIndex(i))
		err := cs.archiveChapter.AddPage(archPage, archIndex)
		if err != nil {
			return fmt.Errorf("failed to archive page: %w", err)
		}
	}

	return nil
}

// RecallPage 从 Archive Chapter 召回 Page
func (cs *CommonSegment) RecallPage(index PageIndex) (*Page, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	return cs.archiveChapter.RecallPage(index)
}

// ProcessTurn 处理一轮对话（触发归档和卸载）
func (cs *CommonSegment) ProcessTurn() ([]PageIndex, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.turnCounter++
	cs.archiveChapter.AdvanceTurn()

	// 处理卸载逻辑
	unloaded, err := cs.archiveChapter.ProcessUnload()
	if err != nil {
		return nil, fmt.Errorf("failed to process unload: %w", err)
	}

	return unloaded, nil
}

// GetMessageList 获取完整的 MessageList（用于发送给 LLM）
func (cs *CommonSegment) GetMessageList() *message.MessageList {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	msgList := message.NewMessageList()

	// 1. 首先添加 ArchiveChapter 的 Contents Page
	archiveMsgs := cs.archiveChapter.ToMessageList()
	for _, msg := range archiveMsgs.Msgs {
		msgList.AddMessageContent(msg.Role, msg.Content)
	}

	// 2. 然后添加 ActiveChapter 的完整 Pages
	activeMsgs := cs.activeChapter.ToMessageList()
	for _, msg := range activeMsgs.Msgs {
		msgList.AddMessageContent(msg.Role, msg.Content)
	}

	return msgList
}

// GetCurrentTurn 获取当前对话轮次
func (cs *CommonSegment) GetCurrentTurn() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.turnCounter
}

// GetActiveChapter 获取 ActiveChapter
func (cs *CommonSegment) GetActiveChapter() *ActiveChapter {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.activeChapter
}

// GetArchiveChapter 获取 ArchiveChapter
func (cs *CommonSegment) GetArchiveChapter() *ArchiveChapter {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.archiveChapter
}

// ContextSystem 完整的上下文管理系统
type ContextSystem struct {
	mu            sync.RWMutex
	systemSegment *SystemSegment
	commonSegment *CommonSegment
}

// NewContextSystem 创建上下文系统
func NewContextSystem(
	sysMsgs []message.Message,
	activeSize int,
	archiveMaxToken int,
	storage PageStorage,
	compactModel *llm.CompactModel,
) *ContextSystem {
	return &ContextSystem{
		systemSegment: NewSystemSegmentWithMessages(sysMsgs),
		commonSegment: NewCommonSegment(activeSize, archiveMaxToken, storage, compactModel),
	}
}

// NewContextSystemFromEntries 从 Entry 创建上下文系统
func NewContextSystemFromEntries(
	sysEntries []*Entry,
	activeSize int,
	archiveMaxToken int,
	storage PageStorage,
	compactModel *llm.CompactModel,
) *ContextSystem {
	return &ContextSystem{
		systemSegment: NewSystemSegmentWithEntries(sysEntries),
		commonSegment: NewCommonSegment(activeSize, archiveMaxToken, storage, compactModel),
	}
}

// SetSystemPrompt 设置系统提示词
func (ctx *ContextSystem) SetSystemPrompt(prompt string) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	// 创建新的 Page 存储系统提示词
	page := NewPage(0, "system", -1, "System prompt", nil)
	page.AddEntry(NewEntry(message.System, message.NewContentString(prompt)))

	// 清空旧的系统 Chapter 并添加新的 Page
	systemChapter := ctx.systemSegment.systemChapter
	systemChapter.BaseChapter = NewBaseChapter(TypeSystemChapter)
	systemChapter.AddSystemPage(page, 0)
}

// SetExtraTools 设置额外的工具描述（如 ATTP 召回工具）
func (ctx *ContextSystem) SetExtraTools(tools string) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.systemSegment.SetExtraTools(tools)
}

// AddUserEntry 添加用户 Entry 并创建新 Page
func (ctx *ContextSystem) AddUserEntry(entry *Entry) (PageIndex, error) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	// 创建新 Page
	page := NewPage(0, "", -1, "", nil)
	page.AddEntry(entry)

	// 生成新的索引
	index := ctx.commonSegment.GetArchiveChapter().GetContentsPage().NextIndex()

	// 添加到 CommonSegment
	err := ctx.commonSegment.AddPage(page, index)
	if err != nil {
		return 0, fmt.Errorf("failed to add user entry: %w", err)
	}

	return index, nil
}

// AddUserMessage 添加用户消息并创建新 Page（便利方法）
func (ctx *ContextSystem) AddUserMessage(content string) (PageIndex, error) {
	return ctx.AddUserEntry(NewEntry(message.User, message.NewContentString(content)))
}

// AddUserMessageContent 添加用户消息内容（便利方法）
func (ctx *ContextSystem) AddUserMessageContent(content message.Content) (PageIndex, error) {
	entry := NewEntry(message.User, content)
	return ctx.AddUserEntry(entry)
}

// AddAssistantEntry 添加助手 Entry 到当前 Page
func (ctx *ContextSystem) AddAssistantEntry(entry *Entry) error {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	// 获取 ActiveChapter 中最新的 Page
	pages := ctx.commonSegment.GetActiveChapter().GetPages()
	if len(pages) == 0 {
		return fmt.Errorf("no active page found")
	}

	// 添加到最新的 Page
	latestPage := pages[len(pages)-1]
	latestPage.AddEntry(entry)

	return nil
}

// AddAssistantMessage 添加助手消息到当前 Page（便利方法）
func (ctx *ContextSystem) AddAssistantMessage(msg message.Message) error {
	return ctx.AddAssistantEntry(NewEntryFromMessage(msg))
}

// AddAssistantMessageContent 添加助手消息内容（便利方法）
func (ctx *ContextSystem) AddAssistantMessageContent(role string, content string) error {
	entry := NewEntry(role, message.NewContentString(content))
	return ctx.AddAssistantEntry(entry)
}

// AddToolEntry 添加工具 Entry 到当前 Page
func (ctx *ContextSystem) AddToolEntry(entry *Entry) error {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	// 获取 ActiveChapter 中最新的 Page
	pages := ctx.commonSegment.GetActiveChapter().GetPages()
	if len(pages) == 0 {
		return fmt.Errorf("no active page found")
	}

	// 添加到最新的 Page
	latestPage := pages[len(pages)-1]
	latestPage.AddEntry(entry)

	return nil
}

// AddToolMessage 添加工具消息到当前 Page（便利方法）
func (ctx *ContextSystem) AddToolMessage(content string) error {
	entry := NewEntry(message.Tool, message.NewContentString(content))
	return ctx.AddToolEntry(entry)
}

// ProcessTurn 处理一轮对话结束
func (ctx *ContextSystem) ProcessTurn() ([]PageIndex, error) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	return ctx.commonSegment.ProcessTurn()
}

// GetMessageList 获取完整的上下文 MessageList
func (ctx *ContextSystem) GetMessageList() *message.MessageList {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	// 合并 SystemSegment 和 CommonSegment 的消息
	msgList := message.NewMessageList()

	// 1. 系统消息
	sysMsgs := ctx.systemSegment.GetMessageList()
	for _, msg := range sysMsgs.Msgs {
		msgList.AddMessageContent(msg.Role, msg.Content)
	}

	// 2. 用户交互消息
	commonMsgs := ctx.commonSegment.GetMessageList()
	for _, msg := range commonMsgs.Msgs {
		msgList.AddMessageContent(msg.Role, msg.Content)
	}

	return msgList
}

// RecallPage 从 Archive Chapter 召回 Page
func (ctx *ContextSystem) RecallPage(index PageIndex) (*Page, error) {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	return ctx.commonSegment.RecallPage(index)
}

// GetCurrentTurn 获取当前对话轮次
func (ctx *ContextSystem) GetCurrentTurn() int {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.commonSegment.GetCurrentTurn()
}

// ListArchivedPages 列出 ArchiveChapter 中的所有 Page
func (ctx *ContextSystem) ListArchivedPages() []PageEntry {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.commonSegment.GetArchiveChapter().ListContentsEntries()
}

// GetSystemSegment 获取 SystemSegment
func (ctx *ContextSystem) GetSystemSegment() *SystemSegment {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.systemSegment
}

// GetCommonSegment 获取 CommonSegment
func (ctx *ContextSystem) GetCommonSegment() *CommonSegment {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.commonSegment
}

// GetActivePage 获取当前活跃的 Page
func (ctx *ContextSystem) GetActivePage() (*Page, error) {
	ctx.mu.RLock()
	defer ctx.mu.Unlock()

	pages := ctx.commonSegment.GetActiveChapter().GetPages()
	if len(pages) == 0 {
		return nil, fmt.Errorf("no active page found")
	}

	return pages[len(pages)-1], nil
}

// CreateNewPage 创建并返回一个新的空白 Page（用于高级用法）
func (ctx *ContextSystem) CreateNewPage() *Page {
	return NewPage(0, "", -1, "", nil)
}

// AddPageToActive 直接添加 Page 到 ActiveChapter（高级用法）
func (ctx *ContextSystem) AddPageToActive(page *Page, index PageIndex) error {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	err := ctx.commonSegment.AddPage(page, index)
	return err
}
