package context

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestNewSegment 测试创建Segment
func TestNewSegment(t *testing.T) {
	tests := []struct {
		name        string
		id          SegmentID
		segmentName string
		description string
		segType     SegmentType
	}{
		{
			name:        "SystemSegment",
			id:          "sys",
			segmentName: "System",
			description: "System prompts",
			segType:     SystemSegment,
		},
		{
			name:        "UserSegment",
			id:          "usr",
			segmentName: "User",
			description: "User interactions",
			segType:     UserSegment,
		},
		{
			name:        "ToolSegment",
			id:          "tool",
			segmentName: "Tools",
			description: "Tool calls",
			segType:     ToolSegment,
		},
		{
			name:        "CustomSegment",
			id:          "project-a",
			segmentName: "Project A",
			description: "Project A context",
			segType:     CustomSegment,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seg := NewSegment(tt.id, tt.segmentName, tt.description, tt.segType)

			if seg.GetID() != tt.id {
				t.Errorf("Expected ID %s, got %s", tt.id, seg.GetID())
			}

			if seg.GetName() != tt.segmentName {
				t.Errorf("Expected name %s, got %s", tt.segmentName, seg.GetName())
			}

			if seg.GetDescription() != tt.description {
				t.Errorf("Expected description %s, got %s", tt.description, seg.GetDescription())
			}

			if seg.GetType() != tt.segType {
				t.Errorf("Expected type %d, got %d", tt.segType, seg.GetType())
			}

			// 默认权限应该是 ReadOnly
			if seg.GetPermission() != ReadOnly {
				t.Errorf("Expected default permission ReadOnly, got %v", seg.GetPermission())
			}

			// 创建时间应该在最近
			if time.Since(seg.GetCreatedAt()) > time.Second {
				t.Errorf("CreatedAt seems too old: %v", seg.GetCreatedAt())
			}
		})
	}
}

// TestSegment_SetName 测试设置名称
func TestSegment_SetName(t *testing.T) {
	seg := NewSegment("test", "Test", "Test segment", CustomSegment)

	// 正常设置名称
	err := seg.SetName("New Name")
	if err != nil {
		t.Fatalf("Failed to set name: %v", err)
	}

	if seg.GetName() != "New Name" {
		t.Errorf("Expected name 'New Name', got '%s'", seg.GetName())
	}

	// 尝试设置空名称
	err = seg.SetName("")
	if err == nil {
		t.Error("Expected error when setting empty name, got nil")
	}

	// 更新时间应该被更新
	oldUpdatedAt := seg.GetUpdatedAt()
	time.Sleep(10 * time.Millisecond)
	seg.SetName("Another Name")
	if !seg.GetUpdatedAt().After(oldUpdatedAt) {
		t.Error("UpdatedAt should be updated after SetName")
	}
}

// TestSegment_SetDescription 测试设置描述
func TestSegment_SetDescription(t *testing.T) {
	seg := NewSegment("test", "Test", "Test segment", CustomSegment)

	newDesc := "New description"
	err := seg.SetDescription(newDesc)
	if err != nil {
		t.Fatalf("Failed to set description: %v", err)
	}

	if seg.GetDescription() != newDesc {
		t.Errorf("Expected description '%s', got '%s'", newDesc, seg.GetDescription())
	}
}

// TestSegment_SetMaxCapacity 测试设置最大容量
func TestSegment_SetMaxCapacity(t *testing.T) {
	seg := NewSegment("test", "Test", "Test segment", CustomSegment)

	// 正常设置容量
	err := seg.SetMaxCapacity(4000)
	if err != nil {
		t.Fatalf("Failed to set max capacity: %v", err)
	}

	if seg.GetMaxCapacity() != 4000 {
		t.Errorf("Expected max capacity 4000, got %d", seg.GetMaxCapacity())
	}

	// 尝试设置负数容量
	err = seg.SetMaxCapacity(-1)
	if err == nil {
		t.Error("Expected error when setting negative capacity, got nil")
	}

	// 零容量应该是允许的
	err = seg.SetMaxCapacity(0)
	if err != nil {
		t.Errorf("Failed to set zero capacity: %v", err)
	}
}

// TestSegment_SetPermission 测试设置权限
func TestSegment_SetPermission(t *testing.T) {
	seg := NewSegment("test", "Test", "Test segment", CustomSegment)

	// 测试所有权限级别
	permissions := []SegmentPermission{ReadOnly, ReadWrite, SystemManaged}
	for _, perm := range permissions {
		err := seg.SetPermission(perm)
		if err != nil {
			t.Errorf("Failed to set permission %v: %v", perm, err)
		}

		if seg.GetPermission() != perm {
			t.Errorf("Expected permission %v, got %v", perm, seg.GetPermission())
		}
	}
}

// TestSegment_SetRootIndex 测试设置根索引
func TestSegment_SetRootIndex(t *testing.T) {
	seg := NewSegment("test", "Test", "Test segment", CustomSegment)

	rootIndex := PageIndex("test-0")
	err := seg.SetRootIndex(rootIndex)
	if err != nil {
		t.Fatalf("Failed to set root index: %v", err)
	}

	if seg.GetRootIndex() != rootIndex {
		t.Errorf("Expected root index %s, got %s", rootIndex, seg.GetRootIndex())
	}
}

// TestSegment_IsReadOnly 测试是否只读
func TestSegment_IsReadOnly(t *testing.T) {
	seg := NewSegment("test", "Test", "Test segment", CustomSegment)

	// 默认是只读
	if !seg.IsReadOnly() {
		t.Error("Expected segment to be ReadOnly by default")
	}

	seg.SetPermission(ReadOnly)
	if !seg.IsReadOnly() {
		t.Error("Expected segment to be ReadOnly")
	}

	seg.SetPermission(ReadWrite)
	if seg.IsReadOnly() {
		t.Error("Expected segment not to be ReadOnly")
	}

	seg.SetPermission(SystemManaged)
	if seg.IsReadOnly() {
		t.Error("Expected segment not to be ReadOnly")
	}
}

// TestSegment_CanModify 测试是否可修改
func TestSegment_CanModify(t *testing.T) {
	seg := NewSegment("test", "Test", "Test segment", CustomSegment)

	// ReadOnly 不可修改
	seg.SetPermission(ReadOnly)
	if seg.CanModify() {
		t.Error("Expected ReadOnly segment to not be modifiable")
	}

	// ReadWrite 可修改
	seg.SetPermission(ReadWrite)
	if !seg.CanModify() {
		t.Error("Expected ReadWrite segment to be modifiable")
	}

	// SystemManaged 可修改（系统管理）
	seg.SetPermission(SystemManaged)
	if !seg.CanModify() {
		t.Error("Expected SystemManaged segment to be modifiable")
	}
}

// TestSegment_Marshal 测试序列化
func TestSegment_Marshal(t *testing.T) {
	seg := NewSegment("usr", "User", "User interactions", UserSegment)
	seg.SetPermission(ReadWrite)
	seg.SetMaxCapacity(4000)
	rootIndex := PageIndex("usr-0")
	seg.SetRootIndex(rootIndex)

	data, err := seg.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal segment: %v", err)
	}

	// 验证JSON格式
	var jsonData segmentJSON
	if err := json.Unmarshal(data, &jsonData); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if jsonData.ID != "usr" {
		t.Errorf("Expected ID 'usr', got '%s'", jsonData.ID)
	}

	if jsonData.Name != "User" {
		t.Errorf("Expected name 'User', got '%s'", jsonData.Name)
	}

	if jsonData.SegmentType != UserSegment {
		t.Errorf("Expected type %d, got %d", UserSegment, jsonData.SegmentType)
	}

	if jsonData.Permission != ReadWrite {
		t.Errorf("Expected permission %d, got %d", ReadWrite, jsonData.Permission)
	}

	if jsonData.MaxCapacity != 4000 {
		t.Errorf("Expected max capacity 4000, got %d", jsonData.MaxCapacity)
	}

	if jsonData.RootIndex != "usr-0" {
		t.Errorf("Expected root index 'usr-0', got '%s'", jsonData.RootIndex)
	}
}

// TestSegment_Unmarshal 测试反序列化
func TestSegment_Unmarshal(t *testing.T) {
	// 准备JSON数据
	jsonData := `{
		"id": "sys",
		"name": "System",
		"segmentType": 0,
		"description": "System prompts",
		"rootIndex": "sys-0",
		"maxCapacity": 8000,
		"permission": 0,
		"createdAt": "2025-02-06T10:00:00Z",
		"updatedAt": "2025-02-06T10:00:00Z"
	}`

	seg := &Segment{}
	err := seg.Unmarshal([]byte(jsonData))
	if err != nil {
		t.Fatalf("Failed to unmarshal segment: %v", err)
	}

	if seg.GetID() != "sys" {
		t.Errorf("Expected ID 'sys', got '%s'", seg.GetID())
	}

	if seg.GetName() != "System" {
		t.Errorf("Expected name 'System', got '%s'", seg.GetName())
	}

	if seg.GetType() != SystemSegment {
		t.Errorf("Expected type %d, got %d", SystemSegment, seg.GetType())
	}

	if seg.GetDescription() != "System prompts" {
		t.Errorf("Expected description 'System prompts', got '%s'", seg.GetDescription())
	}

	if seg.GetRootIndex() != "sys-0" {
		t.Errorf("Expected root index 'sys-0', got '%s'", seg.GetRootIndex())
	}

	if seg.GetMaxCapacity() != 8000 {
		t.Errorf("Expected max capacity 8000, got %d", seg.GetMaxCapacity())
	}

	if seg.GetPermission() != ReadOnly {
		t.Errorf("Expected permission ReadOnly, got %v", seg.GetPermission())
	}

	// 验证时间解析
	expectedTime, _ := time.Parse(time.RFC3339, "2025-02-06T10:00:00Z")
	if seg.GetCreatedAt() != expectedTime {
		t.Errorf("Expected created at %v, got %v", expectedTime, seg.GetCreatedAt())
	}

	if seg.GetUpdatedAt() != expectedTime {
		t.Errorf("Expected updated at %v, got %v", expectedTime, seg.GetUpdatedAt())
	}
}

// TestSegment_MarshalRoundTrip 测试序列化往返
func TestSegment_MarshalRoundTrip(t *testing.T) {
	original := NewSegment("test", "Test Segment", "Test description", CustomSegment)
	original.SetPermission(ReadWrite)
	original.SetMaxCapacity(5000)
	original.SetRootIndex(PageIndex("test-0"))

	// 序列化
	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// 反序列化
	restored := &Segment{}
	err = restored.Unmarshal(data)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// 验证所有字段
	if restored.GetID() != original.GetID() {
		t.Errorf("Round trip ID mismatch: expected %s, got %s",
			original.GetID(), restored.GetID())
	}

	if restored.GetName() != original.GetName() {
		t.Errorf("Round trip name mismatch: expected %s, got %s",
			original.GetName(), restored.GetName())
	}

	if restored.GetDescription() != original.GetDescription() {
		t.Errorf("Round trip description mismatch: expected %s, got %s",
			original.GetDescription(), restored.GetDescription())
	}

	if restored.GetType() != original.GetType() {
		t.Errorf("Round trip type mismatch: expected %d, got %d",
			original.GetType(), restored.GetType())
	}

	if restored.GetRootIndex() != original.GetRootIndex() {
		t.Errorf("Round trip root index mismatch: expected %s, got %s",
			original.GetRootIndex(), restored.GetRootIndex())
	}

	if restored.GetMaxCapacity() != original.GetMaxCapacity() {
		t.Errorf("Round trip max capacity mismatch: expected %d, got %d",
			original.GetMaxCapacity(), restored.GetMaxCapacity())
	}

	if restored.GetPermission() != original.GetPermission() {
		t.Errorf("Round trip permission mismatch: expected %v, got %v",
			original.GetPermission(), restored.GetPermission())
	}

	// 创建时间应该相同（忽略精度差异）
	if restored.GetCreatedAt().Unix() != original.GetCreatedAt().Unix() {
		t.Errorf("Round trip created at mismatch: expected %v, got %v",
			original.GetCreatedAt().Unix(), restored.GetCreatedAt().Unix())
	}
}

// TestSegment_String 测试字符串表示
func TestSegment_String(t *testing.T) {
	seg := NewSegment("test", "Test Segment", "Test description", CustomSegment)
	seg.SetPermission(ReadWrite)
	seg.SetRootIndex(PageIndex("test-0"))

	str := seg.String()
	expected := "[test]" // 应该包含 ID

	if !strings.Contains(str, expected) {
		t.Errorf("Expected string to contain '%s', got '%s'", expected, str)
	}

	// 验证其他信息
	if !strings.Contains(str, "Test Segment") {
		t.Error("Expected string to contain name")
	}

	if !strings.Contains(str, "ReadWrite") {
		t.Error("Expected string to contain permission")
	}

	if !strings.Contains(str, "test-0") {
		t.Error("Expected string to contain root index")
	}
}

// TestSegmentType_String 测试SegmentType字符串表示
func TestSegmentType_String(t *testing.T) {
	tests := []struct {
		segType     SegmentType
		expectedStr string
	}{
		{SystemSegment, "SystemSegment"},
		{UserSegment, "UserSegment"},
		{ToolSegment, "ToolSegment"},
		{CustomSegment, "CustomSegment"},
		{SegmentType(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedStr, func(t *testing.T) {
			if tt.segType.String() != tt.expectedStr {
				t.Errorf("Expected '%s', got '%s'", tt.expectedStr, tt.segType.String())
			}
		})
	}
}

// TestSegmentPermission_String 测试SegmentPermission字符串表示
func TestSegmentPermission_String(t *testing.T) {
	tests := []struct {
		permission   SegmentPermission
		expectedStr string
	}{
		{ReadOnly, "ReadOnly"},
		{ReadWrite, "ReadWrite"},
		{SystemManaged, "SystemManaged"},
		{SegmentPermission(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedStr, func(t *testing.T) {
			if tt.permission.String() != tt.expectedStr {
				t.Errorf("Expected '%s', got '%s'", tt.expectedStr, tt.permission.String())
			}
		})
	}
}

// TestSegment_DefaultValues 测试默认值
func TestSegment_DefaultValues(t *testing.T) {
	seg := NewSegment("test", "Test", "Test description", CustomSegment)

	// rootIndex 默认为空
	if seg.GetRootIndex() != "" {
		t.Errorf("Expected empty root index, got %s", seg.GetRootIndex())
	}

	// maxCapacity 默认为0
	if seg.GetMaxCapacity() != 0 {
		t.Errorf("Expected max capacity 0, got %d", seg.GetMaxCapacity())
	}

	// permission 默认为 ReadOnly
	if seg.GetPermission() != ReadOnly {
		t.Errorf("Expected ReadOnly permission, got %v", seg.GetPermission())
	}

	// 创建时间和更新时间应该相同
	if seg.GetCreatedAt() != seg.GetUpdatedAt() {
		t.Error("CreatedAt and UpdatedAt should be the same initially")
	}
}

// TestSegment_UpdatedAt 测试更新时间
func TestSegment_UpdatedAt(t *testing.T) {
	seg := NewSegment("test", "Test", "Test description", CustomSegment)

	oldUpdatedAt := seg.GetUpdatedAt()
	time.Sleep(10 * time.Millisecond)

	// 更新任一字段应该更新 updatedAt
	seg.SetPermission(ReadWrite)

	if !seg.GetUpdatedAt().After(oldUpdatedAt) {
		t.Error("UpdatedAt should be updated after modification")
	}
}
