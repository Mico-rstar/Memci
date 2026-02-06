package context

import (
	"fmt"
	"time"
)

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

// AgentContext Agent的上下文代理
type AgentContext struct {
	// 被代理的ContextSystem
	system *ContextSystem

	// 元数据
	createdAt time.Time
	updatedAt time.Time
}

// NewAgentContext 创建新的AgentContext
func NewAgentContext(system *ContextSystem) *AgentContext {
	now := time.Now()
	return &AgentContext{
		system:    system,
		createdAt: now,
		updatedAt: now,
	}
}

// ============ 权限检查方法 ============

// checkPermission 统一的权限检查入口
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

// getRequiredLevel 根据操作类型确定所需权限级别
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

// isPermissionSufficient 检查Segment权限是否足够
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

// ============ 核心 API（Agent 可见方法） ============

// ============ Segment 查询方法 ============

// GetSegment 获取Segment（只读，返回副本）
func (ac *AgentContext) GetSegment(id SegmentID) (Segment, error) {
	return ac.system.GetSegment(id)
}

// ListSegments 列出所有Segment（只读，返回副本）
func (ac *AgentContext) ListSegments() ([]Segment, error) {
	return ac.system.ListSegments()
}

// ============ Page 操作方法 ============

// ============ 状态变更方法 ============

// UpdatePage 更新Page信息（写权限）
func (ac *AgentContext) UpdatePage(pageIndex PageIndex, name, description string) error {
	// 1. 权限检查
	if err := ac.checkPermission(pageIndex, "updatePage"); err != nil {
		return err
	}

	// 2. 调用ContextSystem内部方法
	return ac.system.updatePageInternal(pageIndex, name, description)
}

// ExpandDetails 展开Page详情（写权限）
func (ac *AgentContext) ExpandDetails(pageIndex PageIndex) error {
	// 1. 权限检查
	if err := ac.checkPermission(pageIndex, "expandDetails"); err != nil {
		return err
	}

	// 2. 调用ContextSystem内部方法
	return ac.system.expandDetailsInternal(pageIndex)
}

// HideDetails 隐藏Page详情（写权限）
func (ac *AgentContext) HideDetails(pageIndex PageIndex) error {
	// 1. 权限检查
	if err := ac.checkPermission(pageIndex, "hideDetails"); err != nil {
		return err
	}

	// 2. 调用ContextSystem内部方法
	return ac.system.hideDetailsInternal(pageIndex)
}

// ============ 结构操作方法 ============

// MovePage 移动Page（写权限）
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

// RemovePage 删除Page（写权限）
func (ac *AgentContext) RemovePage(pageIndex PageIndex) error {
	// 1. 权限检查
	if err := ac.checkPermission(pageIndex, "removePage"); err != nil {
		return err
	}

	// 2. 调用ContextSystem内部方法
	return ac.system.RemovePage(pageIndex)
}

// CreateDetailPage 创建DetailPage（写权限）
func (ac *AgentContext) CreateDetailPage(name, description, detail string, parentIndex PageIndex) (PageIndex, error) {
	// 1. 权限检查（父Page必须在可写Segment中）
	if err := ac.checkPermission(parentIndex, "createPage"); err != nil {
		return "", err
	}

	// 2. 调用ContextSystem内部方法
	return ac.system.createDetailPageInternal(name, description, detail, parentIndex)
}

// CreateContentsPage 创建ContentsPage（写权限）
func (ac *AgentContext) CreateContentsPage(name, description string, parentIndex PageIndex, children ...PageIndex) (PageIndex, error) {
	// 1. 权限检查
	if parentIndex != "" {
		if err := ac.checkPermission(parentIndex, "createPage"); err != nil {
			return "", err
		}
	}

	// 2. 如果有子节点，也需要检查权限
	for _, childIndex := range children {
		if err := ac.checkPermission(childIndex, "createPage"); err != nil {
			return "", err
		}
	}

	// 3. 调用ContextSystem内部方法
	return ac.system.createContentsPageInternal(name, description, parentIndex, children...)
}

// ============ 查询方法 ============

// GetPage 获取Page（只读）
func (ac *AgentContext) GetPage(pageIndex PageIndex) (Page, error) {
	// 1. 权限检查
	if err := ac.checkPermission(pageIndex, "getPage"); err != nil {
		return nil, err
	}

	// 2. 调用ContextSystem方法
	return ac.system.GetPage(pageIndex)
}

// GetChildren 获取子Page列表（只读）
func (ac *AgentContext) GetChildren(pageIndex PageIndex) ([]Page, error) {
	// 1. 权限检查
	if err := ac.checkPermission(pageIndex, "getChildren"); err != nil {
		return nil, err
	}

	// 2. 调用ContextSystem方法
	return ac.system.GetChildren(pageIndex)
}

// GetParent 获取父Page（只读）
func (ac *AgentContext) GetParent(pageIndex PageIndex) (Page, error) {
	// 1. 权限检查
	if err := ac.checkPermission(pageIndex, "getParent"); err != nil {
		return nil, err
	}

	// 2. 调用ContextSystem方法
	return ac.system.GetParent(pageIndex)
}

// GetAncestors 获取祖先Page列表（只读）
func (ac *AgentContext) GetAncestors(pageIndex PageIndex) ([]Page, error) {
	// 1. 权限检查
	if err := ac.checkPermission(pageIndex, "getAncestors"); err != nil {
		return nil, err
	}

	// 2. 调用ContextSystem方法
	return ac.system.GetAncestors(pageIndex)
}

// FindPage 查找Page（只读）
func (ac *AgentContext) FindPage(query string) []Page {
	// FindPage 不需要权限检查，返回所有匹配的结果
	// Agent可以搜索任何内容，但只能读取有权限的Page
	return ac.system.FindPage(query)
}
