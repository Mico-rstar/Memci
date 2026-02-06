package context

import (
	"fmt"
	"sync"
	"testing"
)

// TestNewContextSystem 测试创建ContextSystem
func TestNewContextSystem(t *testing.T) {
	cs := NewContextSystem()

	if cs == nil {
		t.Fatal("NewContextSystem returned nil")
	}

	// 验证默认使用内存存储
	if cs.GetStorage() == nil {
		t.Error("Expected default storage to be set")
	}

	// 验证初始状态
	segments, _ := cs.ListSegments()
	if len(segments) != 0 {
		t.Errorf("Expected 0 segments, got %d", len(segments))
	}

	pages := cs.ListPages()
	if len(pages) != 0 {
		t.Errorf("Expected 0 pages, got %d", len(pages))
	}
}

// TestNewContextSystemWithStorage 测试使用指定存储创建ContextSystem
func TestNewContextSystemWithStorage(t *testing.T) {
	customStorage := NewMemoryStorage()
	cs := NewContextSystemWithStorage(customStorage)

	if cs == nil {
		t.Fatal("NewContextSystemWithStorage returned nil")
	}

	if cs.GetStorage() != customStorage {
		t.Error("Storage not set correctly")
	}
}

// TestContextSystem_SetStorage 测试设置存储
func TestContextSystem_SetStorage(t *testing.T) {
	cs := NewContextSystem()
	newStorage := NewMemoryStorage()

	cs.SetStorage(newStorage)

	if cs.GetStorage() != newStorage {
		t.Error("SetStorage failed")
	}
}

// ============ Segment 管理测试 ============

// TestContextSystem_AddSegment 测试添加Segment
func TestContextSystem_AddSegment(t *testing.T) {
	cs := NewContextSystem()

	seg := NewSegment("sys", "System", "System context", SystemSegment)
	err := cs.AddSegment(*seg)
	if err != nil {
		t.Fatalf("Failed to add segment: %v", err)
	}

	// 验证Segment已添加
	retrieved, err := cs.GetSegment("sys")
	if err != nil {
		t.Fatalf("Failed to get segment: %v", err)
	}

	if retrieved.GetID() != "sys" {
		t.Errorf("Expected ID 'sys', got '%s'", retrieved.GetID())
	}

	if retrieved.GetName() != "System" {
		t.Errorf("Expected name 'System', got '%s'", retrieved.GetName())
	}
}

// TestContextSystem_AddSegment_Duplicate 测试添加重复Segment
func TestContextSystem_AddSegment_Duplicate(t *testing.T) {
	cs := NewContextSystem()

	seg1 := NewSegment("sys", "System", "System context", SystemSegment)
	cs.AddSegment(*seg1)

	seg2 := NewSegment("sys", "System 2", "Duplicate", SystemSegment)
	err := cs.AddSegment(*seg2)
	if err == nil {
		t.Error("Expected error when adding duplicate segment")
	}
}

// TestContextSystem_RemoveSegment 测试移除Segment
func TestContextSystem_RemoveSegment(t *testing.T) {
	cs := NewContextSystem()

	seg := NewSegment("sys", "System", "System context", SystemSegment)
	cs.AddSegment(*seg)

	err := cs.RemoveSegment("sys")
	if err != nil {
		t.Fatalf("Failed to remove segment: %v", err)
	}

	// 验证Segment已移除
	_, err = cs.GetSegment("sys")
	if err == nil {
		t.Error("Expected error when getting removed segment")
	}
}

// TestContextSystem_RemoveSegment_NotFound 测试移除不存在的Segment
func TestContextSystem_RemoveSegment_NotFound(t *testing.T) {
	cs := NewContextSystem()

	err := cs.RemoveSegment("nonexistent")
	if err == nil {
		t.Error("Expected error when removing non-existent segment")
	}
}

// TestContextSystem_ListSegments 测试列出所有Segment
func TestContextSystem_ListSegments(t *testing.T) {
	cs := NewContextSystem()

	// 添加多个Segment
	cs.AddSegment(*NewSegment("sys", "System", "System context", SystemSegment))
	cs.AddSegment(*NewSegment("usr", "User", "User context", UserSegment))
	cs.AddSegment(*NewSegment("tool", "Tool", "Tool context", ToolSegment))

	segments, err := cs.ListSegments()
	if err != nil {
		t.Fatalf("Failed to list segments: %v", err)
	}

	if len(segments) != 3 {
		t.Errorf("Expected 3 segments, got %d", len(segments))
	}

	// 验证顺序
	if segments[0].GetID() != "sys" {
		t.Errorf("Expected first segment ID 'sys', got '%s'", segments[0].GetID())
	}
	if segments[1].GetID() != "usr" {
		t.Errorf("Expected second segment ID 'usr', got '%s'", segments[1].GetID())
	}
	if segments[2].GetID() != "tool" {
		t.Errorf("Expected third segment ID 'tool', got '%s'", segments[2].GetID())
	}
}

// TestContextSystem_GetSegmentByPageIndex 测试根据PageIndex查找Segment
func TestContextSystem_GetSegmentByPageIndex(t *testing.T) {
	cs := NewContextSystem()

	cs.AddSegment(*NewSegment("sys", "System", "System context", SystemSegment))
	cs.AddSegment(*NewSegment("usr", "User", "User context", UserSegment))

	// 测试sys-1应该找到sys segment
	seg, err := cs.GetSegmentByPageIndex("sys-1")
	if err != nil {
		t.Fatalf("Failed to get segment by page index: %v", err)
	}

	if seg.GetID() != "sys" {
		t.Errorf("Expected segment ID 'sys', got '%s'", seg.GetID())
	}

	// 测试usr-5应该找到usr segment
	seg, err = cs.GetSegmentByPageIndex("usr-5")
	if err != nil {
		t.Fatalf("Failed to get segment by page index: %v", err)
	}

	if seg.GetID() != "usr" {
		t.Errorf("Expected segment ID 'usr', got '%s'", seg.GetID())
	}

	// 测试不存在的索引
	_, err = cs.GetSegmentByPageIndex("nonexistent-1")
	if err == nil {
		t.Error("Expected error for non-existent page index")
	}
}

// ============ Page 存储测试 ============

// TestContextSystem_AddPage 测试添加Page
func TestContextSystem_AddPage(t *testing.T) {
	cs := NewContextSystem()

	// 先添加Segment
	seg := NewSegment("usr", "User", "User context", UserSegment)
	cs.AddSegment(*seg)

	// 创建root page
	rootPage, _ := NewContentsPage("User", "User interactions", "")
	rootPage.SetIndex(PageIndex("usr-0"))

	// 设置Segment的rootIndex
	cs.SetSegmentRootIndex("usr", rootPage.GetIndex())

	cs.AddPage(rootPage)

	// 创建detail page
	detailPage, _ := NewDetailPage("Question", "User question", "How to...", "usr-0")
	detailPage.SetIndex(PageIndex("usr-1"))
	err := cs.AddPage(detailPage)
	if err != nil {
		t.Fatalf("Failed to add page: %v", err)
	}

	// 验证Page已添加
	retrieved, err := cs.GetPage("usr-1")
	if err != nil {
		t.Fatalf("Failed to get page: %v", err)
	}

	if retrieved.GetIndex() != "usr-1" {
		t.Errorf("Expected index 'usr-1', got '%s'", retrieved.GetIndex())
	}
}

func TestContextSystem_AddPage_MultiplePages(t *testing.T) {
	cs := NewContextSystem()
	seg := NewSegment("sys", "system segment", "system prompt", SystemSegment)
	
	// create root page
	rootPage, _ := NewContentsPage("System", "System interactions", "")
	rootPage.SetIndex(cs.GenerateIndex(seg.GetID()))
	cs.SetSegmentRootIndex(seg.GetID(), rootPage.GetIndex())
	cs.AddPage(rootPage)

	// create detail page
	detailPage, _ := NewDetailPage("Question", "System question", "How to...", rootPage.GetIndex())
	detailPage.SetIndex(cs.GenerateIndex(seg.GetID()))
	cs.AddPage(detailPage)

}

// TestContextSystem_AddPage_RootPage 测试添加root page（无父节点）
func TestContextSystem_AddPage_RootPage(t *testing.T) {
	cs := NewContextSystem()

	// 添加Segment
	seg := NewSegment("usr", "User", "User context", UserSegment)
	cs.AddSegment(*seg)

	// 创建root page
	rootPage, _ := NewContentsPage("User", "User interactions", "")
	rootPage.SetIndex(PageIndex("usr-0"))
	cs.SetSegmentRootIndex(seg.GetID(), rootPage.GetIndex())

	// root page没有父节点，应该是允许的
	err := cs.AddPage(rootPage)
	if err != nil {
		t.Fatalf("Failed to add root page: %v", err)
	}
}

// TestContextSystem_AddPage_OrphanPage 测试添加孤儿Page（无父节点且不是root）
func TestContextSystem_AddPage_OrphanPage(t *testing.T) {
	cs := NewContextSystem()

	// 创建一个没有父节点的page
	page, _ := NewDetailPage("Orphan", "Orphan page", "detail", "")
	page.SetIndex(PageIndex("usr-1"))

	// 应该被拒绝，因为它既没有父节点，也不是任何segment的root
	err := cs.AddPage(page)
	if err == nil {
		t.Error("Expected error when adding orphan page")
	}
}

// TestContextSystem_AddPage_Duplicate 测试添加重复Page
func TestContextSystem_AddPage_Duplicate(t *testing.T) {
	cs := NewContextSystem()

	// 添加Segment和root page
	seg := NewSegment("usr", "User", "User context", UserSegment)
	cs.AddSegment(*seg)

	rootPage, _ := NewContentsPage("User", "User interactions", "")
	rootPage.SetIndex(PageIndex("usr-0"))
	cs.SetSegmentRootIndex(seg.GetID(), rootPage.GetIndex())
	cs.AddPage(rootPage)

	// 添加同一个page两次
	page, _ := NewDetailPage("Question", "User question", "How to...", "usr-0")
	page.SetIndex(PageIndex("usr-1"))
	cs.AddPage(page)

	err := cs.AddPage(page)
	if err == nil {
		t.Error("Expected error when adding duplicate page")
	}
}

// TestContextSystem_GetPage 测试获取Page
func TestContextSystem_GetPage(t *testing.T) {
	cs := NewContextSystem()

	// 添加Segment和root page
	seg := NewSegment("usr", "User", "User context", UserSegment)
	cs.AddSegment(*seg)

	rootPage, _ := NewContentsPage("User", "User interactions", "")
	rootPage.SetIndex(PageIndex("usr-0"))
	cs.SetSegmentRootIndex(seg.GetID(), rootPage.GetIndex())
	cs.AddPage(rootPage)

	// 添加detail page
	detailPage, _ := NewDetailPage("Question", "User question", "How to...", "usr-0")
	detailPage.SetIndex(PageIndex("usr-1"))
	cs.AddPage(detailPage)

	// 获取page
	retrieved, err := cs.GetPage("usr-1")
	if err != nil {
		t.Fatalf("Failed to get page: %v", err)
	}

	if retrieved.GetIndex() != "usr-1" {
		t.Errorf("Expected index 'usr-1', got '%s'", retrieved.GetIndex())
	}

	if retrieved.GetName() != "Question" {
		t.Errorf("Expected name 'Question', got '%s'", retrieved.GetName())
	}
}

// TestContextSystem_GetPage_NotFound 测试获取不存在的Page
func TestContextSystem_GetPage_NotFound(t *testing.T) {
	cs := NewContextSystem()

	_, err := cs.GetPage("nonexistent")
	if err == nil {
		t.Error("Expected error when getting non-existent page")
	}
}

// TestContextSystem_RemovePage 测试删除Page
func TestContextSystem_RemovePage(t *testing.T) {
	cs := NewContextSystem()

	// 添加Segment和root page
	seg := NewSegment("usr", "User", "User context", UserSegment)
	cs.AddSegment(*seg)

	rootPage, _ := NewContentsPage("User", "User interactions", "")
	rootPage.SetIndex(PageIndex("usr-0"))
	cs.SetSegmentRootIndex(seg.GetID(), rootPage.GetIndex())
	cs.AddPage(rootPage)

	// 添加detail page
	detailPage, _ := NewDetailPage("Question", "User question", "How to...", "usr-0")
	detailPage.SetIndex(PageIndex("usr-1"))
	cs.AddPage(detailPage)

	// 删除page
	err := cs.RemovePage("usr-1")
	if err != nil {
		t.Fatalf("Failed to remove page: %v", err)
	}

	// 验证page已删除
	_, err = cs.GetPage("usr-1")
	if err == nil {
		t.Error("Expected error when getting removed page")
	}
}

// TestContextSystem_RemovePage_WithChildren 测试删除带子节点的Page
func TestContextSystem_RemovePage_WithChildren(t *testing.T) {
	cs := NewContextSystem()

	// 添加Segment
	seg := NewSegment("usr", "User", "User context", UserSegment)
	cs.AddSegment(*seg)

	// 创建root page
	rootPage, _ := NewContentsPage("User", "User interactions", "")
	rootPage.SetIndex(PageIndex("usr-0"))
	cs.SetSegmentRootIndex(seg.GetID(), rootPage.GetIndex())
	cs.AddPage(rootPage)

	// 创建directory page
	dirPage, _ := NewContentsPage("Directory", "A directory", "usr-0")
	dirPage.SetIndex(PageIndex("usr-1"))
	dirPage.AddChild("usr-2")
	dirPage.AddChild("usr-3")
	cs.AddPage(dirPage)

	// 创建子pages
	child1, _ := NewDetailPage("Child 1", "First child", "detail1", "usr-1")
	child1.SetIndex(PageIndex("usr-2"))
	cs.AddPage(child1)

	child2, _ := NewDetailPage("Child 2", "Second child", "detail2", "usr-1")
	child2.SetIndex(PageIndex("usr-3"))
	cs.AddPage(child2)

	// 删除directory page（应该级联删除子节点）
	err := cs.RemovePage("usr-1")
	if err != nil {
		t.Fatalf("Failed to remove page: %v", err)
	}

	// 验证所有page都已删除
	_, err = cs.GetPage("usr-1")
	if err == nil {
		t.Error("Expected error when getting removed directory page")
	}

	_, err = cs.GetPage("usr-2")
	if err == nil {
		t.Error("Expected error when getting removed child page 1")
	}

	_, err = cs.GetPage("usr-3")
	if err == nil {
		t.Error("Expected error when getting removed child page 2")
	}
}

// TestContextSystem_ListPages 测试列出所有Page
func TestContextSystem_ListPages(t *testing.T) {
	cs := NewContextSystem()

	// 添加Segment
	seg := NewSegment("usr", "User", "User context", UserSegment)
	cs.AddSegment(*seg)

	rootPage, _ := NewContentsPage("User", "User interactions", "")
	rootPage.SetIndex(PageIndex("usr-0"))
	cs.SetSegmentRootIndex(seg.GetID(), rootPage.GetIndex())
	cs.AddPage(rootPage)

	// 添加多个pages
	for i := 1; i <= 3; i++ {
		page, _ := NewDetailPage("Page", "Description", "detail", "usr-0")
		page.SetIndex(PageIndex(fmt.Sprintf("usr-%d", i)))
		cs.AddPage(page)
	}

	pages := cs.ListPages()
	if len(pages) != 4 { // 1 root + 3 children
		t.Errorf("Expected 4 pages, got %d", len(pages))
	}
}

// ============ 索引生成测试 ============

// TestContextSystem_GenerateIndex 测试生成索引
func TestContextSystem_GenerateIndex(t *testing.T) {
	cs := NewContextSystem()

	// 为sys segment生成索引（nextIndex: 0 -> 1）
	index1 := cs.GenerateIndex("sys")
	expected1 := PageIndex("sys-1")
	if index1 != expected1 {
		t.Errorf("Expected index '%s', got '%s'", expected1, index1)
	}

	// 为usr segment生成索引（nextIndex: 1 -> 2，索引是全局递增的）
	index2 := cs.GenerateIndex("usr")
	expected2 := PageIndex("usr-2")
	if index2 != expected2 {
		t.Errorf("Expected index '%s', got '%s'", expected2, index2)
	}

	// 继续为sys segment生成索引（nextIndex: 2 -> 3）
	index3 := cs.GenerateIndex("sys")
	expected3 := PageIndex("sys-3")
	if index3 != expected3 {
		t.Errorf("Expected index '%s', got '%s'", expected3, index3)
	}
}

// ============ 内部方法测试 ============

// TestContextSystem_updatePageInternal 测试更新Page
func TestContextSystem_updatePageInternal(t *testing.T) {
	cs := NewContextSystem()

	// 添加Segment和page
	seg := NewSegment("usr", "User", "User context", UserSegment)
	cs.AddSegment(*seg)

	rootPage, _ := NewContentsPage("User", "User interactions", "")
	rootPage.SetIndex(PageIndex("usr-0"))
	cs.SetSegmentRootIndex(seg.GetID(), rootPage.GetIndex())
	cs.AddPage(rootPage)

	detailPage, _ := NewDetailPage("Question", "User question", "How to...", "usr-0")
	detailPage.SetIndex(PageIndex("usr-1"))
	cs.AddPage(detailPage)

	// 更新page
	err := cs.updatePageInternal("usr-1", "New Name", "New description")
	if err != nil {
		t.Fatalf("Failed to update page: %v", err)
	}

	// 验证更新
	updated, _ := cs.GetPage("usr-1")
	if updated.GetName() != "New Name" {
		t.Errorf("Expected name 'New Name', got '%s'", updated.GetName())
	}

	if updated.GetDescription() != "New description" {
		t.Errorf("Expected description 'New description', got '%s'", updated.GetDescription())
	}
}

// TestContextSystem_expandDetailsInternal 测试展开详情
func TestContextSystem_expandDetailsInternal(t *testing.T) {
	cs := NewContextSystem()

	// 添加Segment和page
	seg := NewSegment("usr", "User", "User context", UserSegment)
	cs.AddSegment(*seg)

	rootPage, _ := NewContentsPage("User", "User interactions", "")
	rootPage.SetIndex(PageIndex("usr-0"))
	rootPage.SetVisibility(Hidden)
	cs.SetSegmentRootIndex(seg.GetID(), rootPage.GetIndex())
	cs.AddPage(rootPage)

	detailPage, _ := NewDetailPage("Question", "User question", "How to...", "usr-0")
	detailPage.SetIndex(PageIndex("usr-1"))
	detailPage.SetVisibility(Hidden)
	cs.AddPage(detailPage)

	// 展开详情
	err := cs.expandDetailsInternal("usr-1")
	if err != nil {
		t.Fatalf("Failed to expand details: %v", err)
	}

	// 验证展开
	expanded, _ := cs.GetPage("usr-1")
	if expanded.GetVisibility() != Expanded {
		t.Errorf("Expected visibility Expanded, got %v", expanded.GetVisibility())
	}
}

// TestContextSystem_hideDetailsInternal 测试隐藏详情
func TestContextSystem_hideDetailsInternal(t *testing.T) {
	cs := NewContextSystem()

	// 添加Segment和page
	seg := NewSegment("usr", "User", "User context", UserSegment)
	cs.AddSegment(*seg)

	rootPage, _ := NewContentsPage("User", "User interactions", "")
	rootPage.SetIndex(PageIndex("usr-0"))
	cs.SetSegmentRootIndex(seg.GetID(), rootPage.GetIndex())
	cs.AddPage(rootPage)

	detailPage, _ := NewDetailPage("Question", "User question", "How to...", "usr-0")
	detailPage.SetIndex(PageIndex("usr-1"))
	detailPage.SetVisibility(Expanded)
	cs.AddPage(detailPage)

	// 隐藏详情
	err := cs.hideDetailsInternal("usr-1")
	if err != nil {
		t.Fatalf("Failed to hide details: %v", err)
	}

	// 验证隐藏
	hidden, _ := cs.GetPage("usr-1")
	if hidden.GetVisibility() != Hidden {
		t.Errorf("Expected visibility Hidden, got %v", hidden.GetVisibility())
	}
}

// TestContextSystem_createDetailPageInternal 测试创建DetailPage
func TestContextSystem_createDetailPageInternal(t *testing.T) {
	cs := NewContextSystem()

	// 添加Segment和root page
	seg := NewSegment("usr", "User", "User context", UserSegment)
	cs.AddSegment(*seg)

	rootPage, _ := NewContentsPage("User", "User interactions", "")
	rootPage.SetIndex(PageIndex("usr-0"))
	cs.SetSegmentRootIndex(seg.GetID(), rootPage.GetIndex())
	cs.AddPage(rootPage)

	// 创建detail page
	newIndex, err := cs.createDetailPageInternal("Question", "User question", "How to...", "usr-0")
	if err != nil {
		t.Fatalf("Failed to create detail page: %v", err)
	}

	// 验证索引格式
	if newIndex == "" {
		t.Error("Expected non-empty index")
	}

	// 验证page已创建
	created, _ := cs.GetPage(newIndex)
	if created.GetIndex() != newIndex {
		t.Errorf("Expected index '%s', got '%s'", newIndex, created.GetIndex())
	}

	if created.GetName() != "Question" {
		t.Errorf("Expected name 'Question', got '%s'", created.GetName())
	}
}

// TestContextSystem_createContentsPageInternal 测试创建ContentsPage
func TestContextSystem_createContentsPageInternal(t *testing.T) {
	cs := NewContextSystem()

	// 添加Segment和root page
	seg := NewSegment("usr", "User", "User context", UserSegment)
	cs.AddSegment(*seg)

	rootPage, _ := NewContentsPage("User", "User interactions", "")
	rootPage.SetIndex(PageIndex("usr-0"))
	cs.SetSegmentRootIndex(seg.GetID(), rootPage.GetIndex())
	cs.AddPage(rootPage)

	// 创建子pages（使用createDetailPageInternal，这样索引会正确递增）
	child1Index, err := cs.createDetailPageInternal("Child 1", "First child", "detail1", "usr-0")
	if err != nil {
		t.Fatalf("Failed to create child1: %v", err)
	}

	child2Index, err := cs.createDetailPageInternal("Child 2", "Second child", "detail2", "usr-0")
	if err != nil {
		t.Fatalf("Failed to create child2: %v", err)
	}

	// 创建ContentsPage，包含已有的子页面
	newIndex, err := cs.createContentsPageInternal("Directory", "A directory", "usr-0", child1Index, child2Index)
	if err != nil {
		t.Fatalf("Failed to create contents page: %v", err)
	}

	// 验证page已创建
	created, _ := cs.GetPage(newIndex)
	if contentsPage, ok := created.(*ContentsPage); ok {
		children := contentsPage.GetChildren()
		if len(children) != 2 {
			t.Errorf("Expected 2 children, got %d", len(children))
		}
	} else {
		t.Error("Expected ContentsPage")
	}

	// 验证子节点的父引用已更新
	child1Updated, _ := cs.GetPage(child1Index)
	if child1Updated.GetParent() != newIndex {
		t.Errorf("Expected parent '%s', got '%s'", newIndex, child1Updated.GetParent())
	}
}

// ============ 查询方法测试 ============

// TestContextSystem_GetChildren 测试获取子Page
func TestContextSystem_GetChildren(t *testing.T) {
	cs := NewContextSystem()

	// 添加Segment
	seg := NewSegment("usr", "User", "User context", UserSegment)
	cs.AddSegment(*seg)

	rootPage, _ := NewContentsPage("User", "User interactions", "")
	rootPage.SetIndex(PageIndex("usr-0"))
	cs.SetSegmentRootIndex(seg.GetID(), rootPage.GetIndex())
	cs.AddPage(rootPage)

	// 创建ContentsPage
	dirPage, _ := NewContentsPage("Directory", "A directory", "usr-0")
	dirPage.SetIndex(PageIndex("usr-1"))
	cs.AddPage(dirPage)

	// 添加子pages
	child1, _ := NewDetailPage("Child 1", "First child", "detail1", "usr-1")
	child1.SetIndex(PageIndex("usr-2"))
	cs.AddPage(child1)

	child2, _ := NewDetailPage("Child 2", "Second child", "detail2", "usr-1")
	child2.SetIndex(PageIndex("usr-3"))
	cs.AddPage(child2)

	// 获取子page
	children, err := cs.GetChildren("usr-1")
	if err != nil {
		t.Fatalf("Failed to get children: %v", err)
	}

	if len(children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(children))
	}
}

// TestContextSystem_GetChildren_NotContentsPage 测试获取非ContentsPage的子节点
func TestContextSystem_GetChildren_NotContentsPage(t *testing.T) {
	cs := NewContextSystem()

	seg := NewSegment("usr", "User", "User context", UserSegment)
	cs.AddSegment(*seg)

	rootPage, _ := NewContentsPage("User", "User interactions", "")
	rootPage.SetIndex(PageIndex("usr-0"))
	cs.SetSegmentRootIndex(seg.GetID(), rootPage.GetIndex())
	cs.AddPage(rootPage)

	detailPage, _ := NewDetailPage("Detail", "Detail page", "detail", "usr-0")
	detailPage.SetIndex(PageIndex("usr-1"))
	cs.AddPage(detailPage)

	_, err := cs.GetChildren("usr-1")
	if err == nil {
		t.Error("Expected error when getting children of DetailPage")
	}
}

// TestContextSystem_FindPage 测试查找Page
func TestContextSystem_FindPage(t *testing.T) {
	cs := NewContextSystem()

	// 添加Segment和pages
	seg := NewSegment("usr", "User", "User context", UserSegment)
	cs.AddSegment(*seg)

	rootPage, _ := NewContentsPage("User", "User interactions", "")
	rootPage.SetIndex(PageIndex("usr-0"))
	cs.SetSegmentRootIndex(seg.GetID(), rootPage.GetIndex())
	cs.AddPage(rootPage)

	page1, _ := NewDetailPage("Go Question", "About Go language", "How to...", "usr-0")
	page1.SetIndex(PageIndex("usr-1"))
	cs.AddPage(page1)

	page2, _ := NewDetailPage("Python Question", "About Python", "How to...", "usr-0")
	page2.SetIndex(PageIndex("usr-2"))
	cs.AddPage(page2)

	// 按名称搜索
	results := cs.FindPage("Go")
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'Go', got %d", len(results))
	}

	// 按描述搜索
	results = cs.FindPage("Python")
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'Python', got %d", len(results))
	}

	// 按部分名称搜索
	results = cs.FindPage("Question")
	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'Question', got %d", len(results))
	}
}

// ============ 并发测试 ============

// TestContextSystem_ConcurrentAccess 测试并发访问
func TestContextSystem_ConcurrentAccess(t *testing.T) {
	cs := NewContextSystem()

	// 添加Segment
	seg := NewSegment("usr", "User", "User context", UserSegment)
	cs.AddSegment(*seg)

	rootPage, _ := NewContentsPage("User", "User interactions", "")
	rootPage.SetIndex(PageIndex("usr-0"))
	cs.SetSegmentRootIndex(seg.GetID(), rootPage.GetIndex())
	cs.AddPage(rootPage)

	var wg sync.WaitGroup
	done := make(chan bool, 20)

	// 并发添加page
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			page, _ := NewDetailPage("Page", "Description", "detail", "usr-0")
			page.SetIndex(PageIndex(fmt.Sprintf("usr-%d", idx+1)))
			cs.AddPage(page)
			done <- true
		}(i)
	}

	// 并发读取
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cs.GetPage("usr-0")
			cs.ListPages()
			done <- true
		}(i)
	}

	// 等待所有操作完成
	wg.Wait()
	close(done)

	// 验证所有操作都完成了
	count := 0
	for range done {
		count++
	}

	if count != 20 {
		t.Errorf("Expected 20 operations, got %d", count)
	}
}
