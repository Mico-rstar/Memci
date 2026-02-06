package context

import (
	"fmt"
	"testing"
)

// TestNewContextManager 测试创建ContextManager
func TestNewContextManager(t *testing.T) {
	cm := NewContextManager()

	if cm == nil {
		t.Fatal("NewContextManager returned nil")
	}

	if cm.system == nil {
		t.Error("system should not be nil")
	}

	if cm.agent == nil {
		t.Error("agent should not be nil")
	}

	if cm.window == nil {
		t.Error("window should not be nil")
	}
}

// TestContextManager_Initialize 测试初始化
func TestContextManager_Initialize(t *testing.T) {
	cm := NewContextManager()

	err := cm.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// 验证系统段已创建
	sysSeg, err := cm.GetSegment("sys")
	if err != nil {
		t.Errorf("Failed to get sys segment: %v", err)
	}

	if sysSeg.GetID() != "sys" {
		t.Errorf("Expected ID 'sys', got '%s'", sysSeg.GetID())
	}

	// 验证用户段已创建
	usrSeg, err := cm.GetSegment("usr")
	if err != nil {
		t.Errorf("Failed to get usr segment: %v", err)
	}

	if usrSeg.GetID() != "usr" {
		t.Errorf("Expected ID 'usr', got '%s'", usrSeg.GetID())
	}
}

// TestContextManager_SetupSegment 测试创建自定义Segment
func TestContextManager_SetupSegment(t *testing.T) {
	cm := NewContextManager()

	err := cm.SetupSegment(
		"custom",
		"Custom Segment",
		"Custom context",
		CustomSegment,
		ReadWrite,
	)
	if err != nil {
		t.Fatalf("SetupSegment failed: %v", err)
	}

	// 验证Segment已创建
	customSeg, err := cm.GetSegment("custom")
	if err != nil {
		t.Errorf("Failed to get custom segment: %v", err)
	}

	if customSeg.GetID() != "custom" {
		t.Errorf("Expected ID 'custom', got '%s'", customSeg.GetID())
	}
}

// TestContextManager_UpdatePage 测试更新Page
func TestContextManager_UpdatePage(t *testing.T) {
	cm := NewContextManager()
	if err := cm.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// 获取用户段的root索引
	usrSeg, err := cm.GetSegment("usr")
	if err != nil {
		t.Fatalf("Failed to get usr segment: %v", err)
	}
	parentIndex := usrSeg.GetRootIndex()

	// 创建一个DetailPage
	childIndex, err := cm.CreateDetailPage("Test Page", "Test description", "Test detail", parentIndex)
	if err != nil {
		t.Fatalf("CreateDetailPage failed: %v", err)
	}

	// 更新Page
	err = cm.UpdatePage(childIndex, "Updated Name", "Updated description")
	if err != nil {
		t.Errorf("UpdatePage failed: %v", err)
	}

	// 验证更新
	page, _ := cm.GetPage(childIndex)
	if page.GetName() != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got '%s'", page.GetName())
	}
}

// TestContextManager_ExpandDetails 测试展开详情
func TestContextManager_ExpandDetails(t *testing.T) {
	cm := NewContextManager()
	if err := cm.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// 获取用户段的root索引
	usrSeg, _ := cm.GetSegment("usr")
	parentIndex := usrSeg.GetRootIndex()

	childIndex, err := cm.CreateDetailPage("Test", "Description", "Detail", parentIndex)
	if err != nil {
		t.Fatalf("CreateDetailPage failed: %v", err)
	}

	err = cm.ExpandDetails(childIndex)
	if err != nil {
		t.Errorf("ExpandDetails failed: %v", err)
	}

	page, _ := cm.GetPage(childIndex)
	if page.GetVisibility() != Expanded {
		t.Error("Page should be Expanded")
	}
}

// TestContextManager_HideDetails 测试隐藏详情
func TestContextManager_HideDetails(t *testing.T) {
	cm := NewContextManager()
	if err := cm.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// 获取用户段的root索引
	usrSeg, _ := cm.GetSegment("usr")
	parentIndex := usrSeg.GetRootIndex()

	childIndex, err := cm.CreateDetailPage("Test", "Description", "Detail", parentIndex)
	if err != nil {
		t.Fatalf("CreateDetailPage failed: %v", err)
	}

	// 先展开
	cm.ExpandDetails(childIndex)

	// 再隐藏
	err = cm.HideDetails(childIndex)
	if err != nil {
		t.Errorf("HideDetails failed: %v", err)
	}

	page, _ := cm.GetPage(childIndex)
	if page.GetVisibility() != Hidden {
		t.Error("Page should be Hidden")
	}
}

// TestContextManager_MovePage 测试移动Page
func TestContextManager_MovePage(t *testing.T) {
	cm := NewContextManager()
	if err := cm.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// 获取用户段的root索引
	usrSeg, _ := cm.GetSegment("usr")
	parentIndex := usrSeg.GetRootIndex()

	// 创建一个ContentsPage作为目标
	targetIndex, err := cm.CreateContentsPage("Target", "Target page", parentIndex)
	if err != nil {
		t.Fatalf("CreateContentsPage failed: %v", err)
	}

	// 创建一个DetailPage
	childIndex, err := cm.CreateDetailPage("Child", "Child description", "Child detail", parentIndex)
	if err != nil {
		t.Fatalf("CreateDetailPage failed: %v", err)
	}

	// 移动Page
	err = cm.MovePage(childIndex, targetIndex)
	if err != nil {
		t.Errorf("MovePage failed: %v", err)
	}

	// 验证移动成功
	child, _ := cm.GetPage(childIndex)
	if child.GetParent() != targetIndex {
		t.Errorf("Expected parent '%s', got '%s'", targetIndex, child.GetParent())
	}
}

// TestContextManager_RemovePage 测试删除Page
func TestContextManager_RemovePage(t *testing.T) {
	cm := NewContextManager()
	if err := cm.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// 获取用户段的root索引
	usrSeg, _ := cm.GetSegment("usr")
	parentIndex := usrSeg.GetRootIndex()

	childIndex, err := cm.CreateDetailPage("Test", "Description", "Detail", parentIndex)
	if err != nil {
		t.Fatalf("CreateDetailPage failed: %v", err)
	}

	err = cm.RemovePage(childIndex)
	if err != nil {
		t.Errorf("RemovePage failed: %v", err)
	}

	_, err = cm.GetPage(childIndex)
	if err == nil {
		t.Error("Page should not exist after removal")
	}
}

// TestContextManager_CreateDetailPage 测试创建DetailPage
func TestContextManager_CreateDetailPage(t *testing.T) {
	cm := NewContextManager()
	if err := cm.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// 获取用户段的root索引
	usrSeg, _ := cm.GetSegment("usr")
	parentIndex := usrSeg.GetRootIndex()

	index, err := cm.CreateDetailPage("My Question", "About Go", "How to...", parentIndex)
	if err != nil {
		t.Fatalf("CreateDetailPage failed: %v", err)
	}

	if index == "" {
		t.Error("Expected non-empty index")
	}

	// 验证Page已创建
	page, err := cm.GetPage(index)
	if err != nil {
		t.Errorf("Failed to get created page: %v", err)
	}

	if page.GetName() != "My Question" {
		t.Errorf("Expected name 'My Question', got '%s'", page.GetName())
	}
}

// TestContextManager_CreateContentsPage 测试创建ContentsPage
func TestContextManager_CreateContentsPage(t *testing.T) {
	cm := NewContextManager()
	if err := cm.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// 获取用户段的root索引
	usrSeg, _ := cm.GetSegment("usr")
	parentIndex := usrSeg.GetRootIndex()

	// 先创建一些子页面
	child1, _ := cm.CreateDetailPage("Child 1", "Child 1", "Detail 1", parentIndex)
	child2, _ := cm.CreateDetailPage("Child 2", "Child 2", "Detail 2", parentIndex)

	// 创建ContentsPage包含这些子页面
	index, err := cm.CreateContentsPage("Directory", "A directory", parentIndex, child1, child2)
	if err != nil {
		t.Fatalf("CreateContentsPage failed: %v", err)
	}

	// 验证ContentsPage已创建
	page, err := cm.GetPage(index)
	if err != nil {
		t.Errorf("Failed to get created page: %v", err)
	}

	if _, ok := page.(*ContentsPage); !ok {
		t.Error("Expected ContentsPage")
	}
}

// TestContextManager_GenerateMessageList 测试生成消息列表
func TestContextManager_GenerateMessageList(t *testing.T) {
	cm := NewContextManager()
	if err := cm.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// 获取用户段的root索引
	usrSeg, _ := cm.GetSegment("usr")
	parentIndex := usrSeg.GetRootIndex()

	// 创建一些子页面
	child1, err := cm.CreateDetailPage("Child 1", "Child 1 description", "Detail 1", parentIndex)
	child2, err := cm.CreateDetailPage("Child 2", "Child 2 description", "Detail 2", parentIndex)
	if err != nil {
		t.Fatalf("CreateDetailPage failed: %v", err)
	}

	cp, err := cm.CreateContentsPage("economy", "something about economic talk", parentIndex, child1, child2)
	if err != nil {
		t.Fatalf("CreateContentsPage failed: %v", err)
	}
	err = cm.ExpandDetails(cp)
	if err != nil {
		t.Fatalf("ExpandDetails failed: %v", err)
	}

	err = cm.ExpandDetails(child1)
	if err != nil {
		t.Fatalf("ExpandDetails failed: %v", err)
	}


	messageList, err := cm.GenerateMessageList()
	if err != nil {
		t.Fatalf("GenerateMessageList failed: %v", err)
	}

	jsonData, err := messageList.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	fmt.Println(string(jsonData))

	if messageList == nil {
		t.Fatal("Expected non-nil message list")
	}
}

// TestContextManager_EstimateTokens 测试估算token
func TestContextManager_EstimateTokens(t *testing.T) {
	cm := NewContextManager()
	if err := cm.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	tokens, err := cm.EstimateTokens()
	if err != nil {
		t.Errorf("EstimateTokens failed: %v", err)
	}

	if tokens < 0 {
		t.Errorf("Expected non-negative tokens, got %d", tokens)
	}
}

// TestContextManager_AutoCollapse 测试自动折叠
func TestContextManager_AutoCollapse(t *testing.T) {
	cm := NewContextManager()
	if err := cm.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// 获取用户段的root索引
	usrSeg, _ := cm.GetSegment("usr")
	parentIndex := usrSeg.GetRootIndex()

	// 创建一些页面
	cm.CreateDetailPage("Test 1", "Description 1", "Detail text 1", parentIndex)
	cm.CreateDetailPage("Test 2", "Description 2", "Detail text 2", parentIndex)

	// 设置一个很高的限制，不应该折叠
	collapsed, err := cm.AutoCollapse(10000)
	if err != nil {
		t.Errorf("AutoCollapse failed: %v", err)
	}

	if len(collapsed) != 0 {
		t.Errorf("Expected no collapsed pages, got %d", len(collapsed))
	}
}

// TestContextManager_GetPage 测试获取Page
func TestContextManager_GetPage(t *testing.T) {
	cm := NewContextManager()
	if err := cm.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// 获取root page
	usrSeg, _ := cm.GetSegment("usr")
	rootIndex := usrSeg.GetRootIndex()

	page, err := cm.GetPage(rootIndex)
	if err != nil {
		t.Errorf("GetPage failed: %v", err)
	}

	if page.GetIndex() != rootIndex {
		t.Errorf("Expected index '%s', got '%s'", rootIndex, page.GetIndex())
	}
}

// TestContextManager_GetChildren 测试获取子Page
func TestContextManager_GetChildren(t *testing.T) {
	cm := NewContextManager()
	if err := cm.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// 获取用户段的root索引
	usrSeg, _ := cm.GetSegment("usr")
	parentIndex := usrSeg.GetRootIndex()

	// 添加子页面
	cm.CreateDetailPage("Child", "Child description", "Detail", parentIndex)

	children, err := cm.GetChildren(parentIndex)
	if err != nil {
		t.Errorf("GetChildren failed: %v", err)
	}

	if len(children) == 0 {
		t.Error("Expected at least one child")
	}
}

// TestContextManager_GetSegment 测试获取Segment
func TestContextManager_GetSegment(t *testing.T) {
	cm := NewContextManager()
	if err := cm.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	seg, err := cm.GetSegment("usr")
	if err != nil {
		t.Errorf("GetSegment failed: %v", err)
	}

	if seg.GetID() != "usr" {
		t.Errorf("Expected ID 'usr', got '%s'", seg.GetID())
	}
}

// TestContextManager_ListSegments 测试列出所有Segment
func TestContextManager_ListSegments(t *testing.T) {
	cm := NewContextManager()
	if err := cm.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	segments, err := cm.ListSegments()
	if err != nil {
		t.Fatalf("ListSegments failed: %v", err)
	}

	if len(segments) != 2 {
		t.Errorf("Expected 2 segments, got %d", len(segments))
	}
}
