package context

import (
	"fmt"
	"memci/message"
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

	// 获取系统段指针并生成索引
	sysSegPtr, err := cm.system.getSegmentInternal("sys")
	if err != nil {
		return fmt.Errorf("failed to get sys segment: %w", err)
	}
	sysRootIndex := sysSegPtr.GenerateIndex()

	// 创建系统提示词 root page
	sysRoot, _ := NewContentsPage("System", "System prompts", "")
	sysRoot.SetVisibility(Expanded)
	sysRoot.SetIndex(sysRootIndex)
	if err := cm.system.SetSegmentRootIndex("sys", sysRoot.GetIndex()); err != nil {
		return err
	}
	if err := cm.system.AddPage(sysRoot); err != nil {
		return err
	}

	// 创建用户交互段
	usrSeg := NewSegment("usr", "User", "User interactions", UserSegment)
	usrSeg.SetPermission(ReadWrite)
	if err := cm.system.AddSegment(*usrSeg); err != nil {
		return err
	}

	// 获取用户段指针并生成索引
	usrSegPtr, err := cm.system.getSegmentInternal("usr")
	if err != nil {
		return fmt.Errorf("failed to get usr segment: %w", err)
	}
	usrRootIndex := usrSegPtr.GenerateIndex()

	// 创建用户 root page
	usrRoot, _ := NewContentsPage("User", "User interactions", "")
	usrRoot.SetVisibility(Expanded)
	usrRoot.SetIndex(usrRootIndex)
	if err := cm.system.SetSegmentRootIndex("usr", usrRoot.GetIndex()); err != nil {
		return err
	}
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
func (cm *ContextManager) GenerateMessageList() (*message.MessageList, error) {
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
