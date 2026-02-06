package context

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SegmentID Segment的唯一标识
type SegmentID string

// SegmentType Segment的类型
type SegmentType int

const (
	// SystemSegment 系统级Segment（如系统提示词、安全规则）
	SystemSegment SegmentType = iota
	// UserSegment 用户交互Segment
	UserSegment
	// ToolSegment 工具调用Segment
	ToolSegment
	// CustomSegment 自定义Segment
	CustomSegment
)

// String 返回Segment类型的字符串表示
func (st SegmentType) String() string {
	switch st {
	case SystemSegment:
		return "SystemSegment"
	case UserSegment:
		return "UserSegment"
	case ToolSegment:
		return "ToolSegment"
	case CustomSegment:
		return "CustomSegment"
	default:
		return "Unknown"
	}
}

// SegmentPermission Segment的权限控制
type SegmentPermission int

const (
	// ReadOnly 只读：Agent不能修改此Segment的任何Page
	ReadOnly SegmentPermission = iota
	// ReadWrite 读写：Agent可以修改此Segment的Page
	ReadWrite
	// SystemManaged 系统管理：只有系统代码可以修改，Agent完全不可操作
	SystemManaged
)

// String 返回权限级别的字符串表示
func (sp SegmentPermission) String() string {
	switch sp {
	case ReadOnly:
		return "ReadOnly"
	case ReadWrite:
		return "ReadWrite"
	case SystemManaged:
		return "SystemManaged"
	default:
		return "Unknown"
	}
}

// Segment 上下文空间的逻辑分段
//
// 设计说明：
// - Segment 作为其下所有 Page 索引的"所有权者"
// - 外部通过 Segment 的方法来获取新的索引，而不是自己生成
// - 每个Segment维护自己的索引计数器，避免全局计数器冲突
type Segment struct {
	// 基本信息
	id          SegmentID   // 唯一标识
	name        string      // Segment名称
	segType     SegmentType // Segment类型
	description string      // Segment描述

	// 根Page
	rootIndex PageIndex // root ContentsPage的索引

	// 索引生成（每个Segment自己的计数器）
	nextIndex int    // 当前索引计数器
	segmentID string // 用于生成索引的ID前缀（冗余，方便序列化）

	// 配置
	maxCapacity int               // 最大Token容量（可选，用于上下文窗口管理）
	permission  SegmentPermission // 权限控制

	// 元数据
	createdAt time.Time
	updatedAt time.Time
}

// NewSegment 创建新的Segment
func NewSegment(id SegmentID, name, description string, segType SegmentType) *Segment {
	now := time.Now()
	return &Segment{
		id:          id,
		name:        name,
		segType:     segType,
		description: description,
		rootIndex:   "",
		nextIndex:   0, // 从0开始，首次生成为1
		segmentID:   string(id),
		maxCapacity: 0,
		permission:  ReadOnly, // 默认只读，由调用者根据类型设置
		createdAt:   now,
		updatedAt:   now,
	}
}

// ============ 索引生成方法（核心改进）============

// GenerateIndex 生成该Segment下的新Page索引
//
// 这是外部获取新索引的唯一入口点，确保：
// 1. 索引格式正确（"{segmentID}-{number}"）
// 2. 计数器不会冲突（每个Segment独立计数）
// 3. 封装实现细节
func (s *Segment) GenerateIndex() PageIndex {
	s.nextIndex++
	return PageIndex(fmt.Sprintf("%s-%d", s.segmentID, s.nextIndex))
}

// GetNextIndex 获取下一个索引（但不递增计数器）
// 用于"预览"下一个索引是什么，但不实际分配
func (s *Segment) GetNextIndex() PageIndex {
	// 下一个索引是nextIndex + 1
	return PageIndex(fmt.Sprintf("%s-%d", s.segmentID, s.nextIndex+1))
}

// ResetIndexCounter 重置索引计数器（危险操作，仅用于内部管理）
// 使用场景：Segment重建、数据迁移等
func (s *Segment) ResetIndexCounter() {
	s.nextIndex = 0
}

// SetIndexCounter 设置索引计数器到指定值
// 使用场景：从持久化存储恢复时恢复状态
func (s *Segment) SetIndexCounter(count int) {
	s.nextIndex = count
}

// GetIndexCounter 获取当前索引计数器的值
func (s *Segment) GetIndexCounter() int {
	return s.nextIndex
}

// ============ 基本信息 Getter/Setter（保持不变）============

// GetID 获取Segment ID
func (s *Segment) GetID() SegmentID {
	return s.id
}

// GetName 获取Segment名称
func (s *Segment) GetName() string {
	return s.name
}

// SetName 设置Segment名称
func (s *Segment) SetName(name string) error {
	if name == "" {
		return fmt.Errorf("segment name cannot be empty")
	}
	s.name = name
	s.updatedAt = time.Now()
	return nil
}

// GetDescription 获取Segment描述
func (s *Segment) GetDescription() string {
	return s.description
}

// SetDescription 设置Segment描述
func (s *Segment) SetDescription(description string) error {
	s.description = description
	s.updatedAt = time.Now()
	return nil
}

// GetType 获取Segment类型
func (s *Segment) GetType() SegmentType {
	return s.segType
}

// ============ 根Page管理 ============

// GetRootIndex 获取root Page索引
func (s *Segment) GetRootIndex() PageIndex {
	return s.rootIndex
}

// SetRootIndex 设置root Page索引
func (s *Segment) SetRootIndex(index PageIndex) error {
	s.rootIndex = index
	s.updatedAt = time.Now()
	return nil
}

// ============ 配置管理 ============

// GetMaxCapacity 获取最大Token容量
func (s *Segment) GetMaxCapacity() int {
	return s.maxCapacity
}

// SetMaxCapacity 设置最大Token容量
func (s *Segment) SetMaxCapacity(capacity int) error {
	if capacity < 0 {
		return fmt.Errorf("max capacity cannot be negative")
	}
	s.maxCapacity = capacity
	s.updatedAt = time.Now()
	return nil
}

// GetPermission 获取权限级别
func (s *Segment) GetPermission() SegmentPermission {
	return s.permission
}

// SetPermission 设置权限级别
func (s *Segment) SetPermission(permission SegmentPermission) error {
	s.permission = permission
	s.updatedAt = time.Now()
	return nil
}

// ============ 权限检查方法 ============

// IsReadOnly 检查是否为只读
func (s *Segment) IsReadOnly() bool {
	return s.permission == ReadOnly
}

// CanModify 检查Agent是否可以修改此Segment的Page
func (s *Segment) CanModify() bool {
	return s.permission == ReadWrite || s.permission == SystemManaged
}

// ============ 元数据 ============

// GetCreatedAt 获取创建时间
func (s *Segment) GetCreatedAt() time.Time {
	return s.createdAt
}

// GetUpdatedAt 获取更新时间
func (s *Segment) GetUpdatedAt() time.Time {
	return s.updatedAt
}

// ============ 序列化/反序列化 ============

// MarshalJSON 序列化为JSON（用于存储）
func (s *Segment) MarshalJSON() ([]byte, error) {

	data, err := marshalSegmentJSON(s)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// UnmarshalJSON 从JSON反序列化
func (s *Segment) UnmarshalJSON(data []byte) error {

	seg, err := unmarshalSegmentJSON(data)
	if err != nil {
		return err
	}

	// 复制数据
	s.id = seg.id
	s.name = seg.name
	s.segType = seg.segType
	s.description = seg.description
	s.rootIndex = seg.rootIndex
	s.maxCapacity = seg.maxCapacity
	s.permission = seg.permission
	s.segmentID = seg.segmentID
	s.nextIndex = seg.nextIndex
	s.createdAt = seg.createdAt
	s.updatedAt = seg.updatedAt

	return nil
}

// Marshal 序列化Segment（用于存储）
func (s *Segment) Marshal() ([]byte, error) {
	return s.MarshalJSON()
}

// Unmarshal 反序列化Segment（用于存储）
func (s *Segment) Unmarshal(data []byte) error {
	return s.UnmarshalJSON(data)
}

// ============ 辅助方法 ============

// String 返回Segment的字符串表示
func (s *Segment) String() string {

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Segment[%s]", s.id))
	builder.WriteString(fmt.Sprintf(" Name=%s", s.name))
	builder.WriteString(fmt.Sprintf(" Type=%s", s.segType))
	builder.WriteString(fmt.Sprintf(" Permission=%s", s.permission))
	builder.WriteString(fmt.Sprintf(" NextIndex=%d", s.nextIndex))
	if s.rootIndex != "" {
		builder.WriteString(fmt.Sprintf(" RootIndex=%s", s.rootIndex))
	}
	return builder.String()
}

// ============ 序列化辅助结构 ============

// segmentJSON 用于JSON序列化的内部结构
type segmentJSON struct {
	ID          SegmentID         `json:"id"`
	Name        string            `json:"name"`
	SegmentType SegmentType       `json:"segmentType"`
	Description string            `json:"description"`
	RootIndex   string            `json:"rootIndex"`
	MaxCapacity int               `json:"maxCapacity"`
	Permission  SegmentPermission `json:"permission"`
	NextIndex   int               `json:"nextIndex"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

// marshalSegmentJSON 将Segment序列化为JSON
func marshalSegmentJSON(s *Segment) ([]byte, error) {
	jsonData := segmentJSON{
		ID:          s.id,
		Name:        s.name,
		SegmentType: s.segType,
		Description: s.description,
		RootIndex:   string(s.rootIndex),
		MaxCapacity: s.maxCapacity,
		Permission:  s.permission,
		NextIndex:   s.nextIndex,
		CreatedAt:   s.createdAt,
		UpdatedAt:   s.updatedAt,
	}

	return json.Marshal(jsonData)
}

// unmarshalSegmentJSON 从JSON反序列化Segment
func unmarshalSegmentJSON(data []byte) (*Segment, error) {
	var jsonData segmentJSON
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal segment: %w", err)
	}

	return &Segment{
		id:          jsonData.ID,
		name:        jsonData.Name,
		segType:     jsonData.SegmentType,
		description: jsonData.Description,
		rootIndex:   PageIndex(jsonData.RootIndex),
		maxCapacity: jsonData.MaxCapacity,
		permission:  jsonData.Permission,
		nextIndex:   jsonData.NextIndex,
		segmentID:   string(jsonData.ID),
		createdAt:   jsonData.CreatedAt,
		updatedAt:   jsonData.UpdatedAt,
	}, nil
}
