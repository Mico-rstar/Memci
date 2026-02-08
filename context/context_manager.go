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
	if _, err := cm.createSegmentWithRootPage(
		"sys", "System", "System prompts",
		SystemSegment, ReadOnly,
	); err != nil {
		return err
	}

	// 创建用户认知段
	if _, err := cm.createSegmentWithRootPage(
		"usr", "User", "对用户的认知，例如用户姓名、兴趣、偏好等",
		UserSegment, ReadWrite,
	); err != nil {
		return err
	}

	// 创建自我认知段
	if _, err := cm.createSegmentWithRootPage(
		"self", "Self", "对自我的认知，例如自己的名字、性格、兴趣等",
		UserSegment, ReadWrite,
	); err != nil {
		return err
	}

	// 创建经验教训
	if _, err := cm.createSegmentWithRootPage(
		"teach", "Teach", "总结经验教训",
		UserSegment, ReadWrite,
	); err != nil {
		return err
	}

	// 创建话题段
	if _, err := cm.createSegmentWithRootPage(
		"topic", "Topic", "记录和用户聊过的有意思的话题",
		UserSegment, ReadWrite,
	); err != nil {
		return err
	}

	// 创建交互段
	if _, err := cm.createSegmentWithRootPage(
		"interact", "Interaction", "最近和用户聊了什么",
		UserSegment, ReadWrite,
	); err != nil {
		return err
	}

	return nil
}

// createSegmentWithRootPage 创建 Segment 及其根 Page
// 返回根 Page 的索引
func (cm *ContextManager) createSegmentWithRootPage(
	id SegmentID,
	name, description string,
	segType SegmentType,
	permission SegmentPermission,
) (PageIndex, error) {
	// 创建 Segment
	seg := NewSegment(id, name, description, segType)
	seg.SetPermission(permission)
	if err := cm.system.AddSegment(*seg); err != nil {
		return "", err
	}

	// 获取段指针并生成索引
	segPtr, err := cm.system.getSegmentInternal(id)
	if err != nil {
		return "", fmt.Errorf("failed to get %s segment: %w", id, err)
	}
	rootIndex := segPtr.GenerateIndex()

	// 创建根 Page
	root, _ := NewContentsPage(name, description, "")
	root.SetVisibility(Expanded)
	root.SetIndex(rootIndex)

	// 设置关联关系并添加到系统
	if err := cm.system.SetSegmentRootIndex(id, root.GetIndex()); err != nil {
		return "", err
	}
	if err := cm.system.AddPage(root); err != nil {
		return "", err
	}

	return rootIndex, nil
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

// GetAgentContext 获取 AgentContext（用于工具调用）
func (cm *ContextManager) GetAgentContext() *AgentContext {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.agent
}

// ============ 系统级操作（绕过权限检查） ============

// CreateDetailPageSystem 系统级创建 DetailPage（绕过权限检查）
func (cm *ContextManager) CreateDetailPageSystem(
	name, description, detail string,
	parentIndex PageIndex,
) (PageIndex, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	return cm.system.createDetailPageInternal(name, description, detail, parentIndex)
}

// ExpandDetailsSystem 系统级展开 Page（绕过权限检查）
func (cm *ContextManager) ExpandDetailsSystem(pageIndex PageIndex) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	page, err := cm.system.GetPage(pageIndex)
	if err != nil {
		return err
	}

	page.SetVisibility(Expanded)

	// 持久化更新
	if cm.system.storage != nil {
		cm.system.storage.Save(page)
	}

	return nil
}

// GetSegmentSystem 系统级获取 Segment（绕过权限检查）
func (cm *ContextManager) GetSegmentSystem(id SegmentID) (*Segment, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.system.getSegmentInternal(id)
}

// ExportToFile 将当前ContextWindow导出到文件
func (cm *ContextManager) ExportToFile(outputDir string, turn int) (string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.window.ExportToFile(outputDir, turn)
}
