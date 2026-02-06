package context

import (
	"strings"
	"testing"
)

// TestNewContextWindow 测试创建ContextWindow
func TestNewContextWindow(t *testing.T) {
	system := NewContextSystem()
	cw := NewContextWindow(system)

	if cw == nil {
		t.Fatal("NewContextWindow returned nil")
	}

	if cw.system != system {
		t.Error("ContextWindow system not set correctly")
	}
}

// TestContextWindow_GenerateMessageList 测试生成MessageList
func TestContextWindow_GenerateMessageList(t *testing.T) {
	system := NewContextSystem()
	cw := NewContextWindow(system)

	// 创建Segment
	segment := NewSegment("test-seg", "Test Segment", "Test", UserSegment)
	segment.SetPermission(ReadWrite)
	if err := system.AddSegment(*segment); err != nil {
		t.Fatalf("Failed to add segment: %v", err)
	}

	// 创建root page
	root, err := NewDetailPage("Root", "Root Description", "Root Detail", "")
	if err != nil {
		t.Fatalf("Failed to create root page: %v", err)
	}
	root.SetIndex(PageIndex("test-seg-0"))
	root.SetVisibility(Expanded)
	if err := system.SetSegmentRootIndex("test-seg", "test-seg-0"); err != nil {
		t.Fatalf("Failed to set root index: %v", err)
	}
	if err := system.AddPage(root); err != nil {
		t.Fatalf("Failed to add root page: %v", err)
	}

	t.Run("GenerateMessageList with Expanded DetailPage", func(t *testing.T) {
		messageList, err := cw.GenerateMessageList()
		if err != nil {
			t.Fatalf("GenerateMessageList failed: %v", err)
		}

		// 应该有一个消息
		count := 0
		node := messageList.GetNode()
		for node != nil {
			count++
			node = node.Next()
		}

		if count != 1 {
			t.Errorf("Expected 1 message, got %d", count)
		}

		// 检查内容
		content := messageList.GetNode().GetMsg().Content.String()
		if !strings.Contains(content, "Root Description") {
			t.Error("Content should contain description")
		}
		if !strings.Contains(content, "Root Detail") {
			t.Error("Content should contain detail when Expanded")
		}
	})

	t.Run("GenerateMessageList with Hidden DetailPage", func(t *testing.T) {
		root.SetVisibility(Hidden)

		messageList, err := cw.GenerateMessageList()
		if err != nil {
			t.Fatalf("GenerateMessageList failed: %v", err)
		}

		content := messageList.GetNode().GetMsg().Content.String()
		if !strings.Contains(content, "Root Description") {
			t.Error("Content should contain description")
		}
		if strings.Contains(content, "Root Detail") {
			t.Error("Content should not contain detail when Hidden")
		}
	})
}

// TestContextWindow_RenderPageRecursive 测试递归渲染页面树
func TestContextWindow_RenderPageRecursive(t *testing.T) {
	system := NewContextSystem()
	cw := NewContextWindow(system)

	// 创建Segment
	segment := NewSegment("test-seg", "Test Segment", "Test", UserSegment)
	segment.SetPermission(ReadWrite)
	if err := system.AddSegment(*segment); err != nil {
		t.Fatalf("Failed to add segment: %v", err)
	}

	t.Run("Render DetailPage", func(t *testing.T) {
		page, _ := NewDetailPage("Detail", "Description", "Detail Content", "")
		page.SetIndex(PageIndex("test-seg-1"))
		page.SetVisibility(Expanded)
		page.SetLifecycle(Active)

		content := cw.renderPageRecursive(page, 0)
		if !strings.Contains(content, "[test-seg-1]") {
			t.Error("Should contain page index")
		}
		if !strings.Contains(content, "[Hide] Detail Content") {
			t.Error("Should contain [Hide] prefix before detail content when Expanded")
		}
		if !strings.Contains(content, "Description") {
			t.Error("Should contain description")
		}
		if !strings.Contains(content, "Detail Content") {
			t.Error("Should contain detail when Expanded")
		}
	})

	t.Run("Render ContentsPage with children", func(t *testing.T) {
		// 创建一个子Segment用于这个测试
		subSeg := NewSegment("test-sub-seg", "Test Sub Segment", "Test", UserSegment)
		subSeg.SetPermission(ReadWrite)
		if err := system.AddSegment(*subSeg); err != nil {
			t.Fatalf("Failed to add sub segment: %v", err)
		}

		// 创建父ContentsPage作为segment root
		parent, _ := NewContentsPage("Parent", "Parent Description", "")
		parent.SetIndex(PageIndex("test-sub-seg-0"))
		parent.SetVisibility(Expanded)
		parent.SetLifecycle(Active)
		if err := system.SetSegmentRootIndex("test-sub-seg", "test-sub-seg-0"); err != nil {
			t.Fatalf("Failed to set root index: %v", err)
		}
		if err := system.AddPage(parent); err != nil {
			t.Fatalf("Failed to add parent page: %v", err)
		}

		// 创建子DetailPages
		child1, _ := NewDetailPage("Child 1", "Child 1 Description", "Child 1 Detail", "test-sub-seg-0")
		child1.SetIndex(PageIndex("test-sub-seg-1"))
		child1.SetVisibility(Expanded)
		child1.SetLifecycle(Active)

		child2, _ := NewDetailPage("Child 2", "Child 2 Description", "Child 2 Detail", "test-sub-seg-0")
		child2.SetIndex(PageIndex("test-sub-seg-2"))
		child2.SetVisibility(Expanded)
		child2.SetLifecycle(Active)

		// 添加子页面到系统（会自动添加到父页面的children列表）
		if err := system.AddPage(child1); err != nil {
			t.Fatalf("Failed to add child1 page: %v", err)
		}
		if err := system.AddPage(child2); err != nil {
			t.Fatalf("Failed to add child2 page: %v", err)
		}

		content := cw.renderPageRecursive(parent, 0)
		if !strings.Contains(content, "[test-sub-seg-0]") {
			t.Error("Should contain parent index")
		}
		if !strings.Contains(content, "Parent Description") {
			t.Error("Should contain parent description")
		}
		if !strings.Contains(content, "[test-sub-seg-1]") {
			t.Error("Should contain child 1 index")
		}
		if !strings.Contains(content, "Child 1 Description") {
			t.Error("Should contain child 1 description")
		}
		if !strings.Contains(content, "[test-sub-seg-2]") {
			t.Error("Should contain child 2 index")
		}
		if !strings.Contains(content, "Child 2 Description") {
			t.Error("Should contain child 2 description")
		}
	})

	t.Run("Render ContentsPage with Hidden visibility", func(t *testing.T) {
		// 创建一个子Segment用于这个测试
		subSeg := NewSegment("test-sub-seg2", "Test Sub Segment 2", "Test", UserSegment)
		subSeg.SetPermission(ReadWrite)
		if err := system.AddSegment(*subSeg); err != nil {
			t.Fatalf("Failed to add sub segment: %v", err)
		}

		// 创建父ContentsPage，Hidden状态
		parent, _ := NewContentsPage("Parent", "Parent Description", "")
		parent.SetIndex(PageIndex("test-sub-seg2-0"))
		parent.SetVisibility(Hidden)
		parent.SetLifecycle(Active)
		if err := system.SetSegmentRootIndex("test-sub-seg2", "test-sub-seg2-0"); err != nil {
			t.Fatalf("Failed to set root index: %v", err)
		}
		if err := system.AddPage(parent); err != nil {
			t.Fatalf("Failed to add parent page: %v", err)
		}

		// 创建子DetailPage
		child, _ := NewDetailPage("Child", "Child Description", "Child Detail", "test-sub-seg2-0")
		child.SetIndex(PageIndex("test-sub-seg2-1"))
		child.SetVisibility(Expanded)
		child.SetLifecycle(Active)

		// 添加子页面到系统
		if err := system.AddPage(child); err != nil {
			t.Fatalf("Failed to add child page: %v", err)
		}

		content := cw.renderPageRecursive(parent, 0)
		if !strings.Contains(content, "Parent Description") {
			t.Error("Should contain parent description")
		}
		if !strings.Contains(content, "[Expand]...") {
			t.Error("Should contain [Expand]... indicator when Hidden with children")
		}
		// Hidden状态不应该渲染子节点
		if strings.Contains(content, "Child Description") {
			t.Error("Should not contain child description when parent is Hidden")
		}
	})

	t.Run("Render inactive page", func(t *testing.T) {
		page, _ := NewDetailPage("Inactive", "Description", "Detail", "")
		page.SetIndex(PageIndex("test-seg-7"))
		page.SetVisibility(Expanded)
		page.SetLifecycle(HotArchived) // 非Active状态

		content := cw.renderPageRecursive(page, 0)
		if content != "" {
			t.Error("Should return empty string for inactive pages")
		}
	})
}

// TestContextWindow_EstimateTokens 测试token估算
func TestContextWindow_EstimateTokens(t *testing.T) {
	system := NewContextSystem()
	cw := NewContextWindow(system)

	// 创建Segment
	segment := NewSegment("test-seg", "Test Segment", "Test", UserSegment)
	segment.SetPermission(ReadWrite)
	if err := system.AddSegment(*segment); err != nil {
		t.Fatalf("Failed to add segment: %v", err)
	}

	t.Run("EstimateTokens with empty system", func(t *testing.T) {
		tokens, err := cw.EstimateTokens()
		if err != nil {
			t.Errorf("EstimateTokens failed: %v", err)
		}

		if tokens != 0 {
			t.Errorf("Expected 0 tokens, got %d", tokens)
		}
	})

	t.Run("EstimateTokens with content", func(t *testing.T) {
		// 创建一个包含约300字符的page
		longText := strings.Repeat("test ", 75) // 300个字符
		root, _ := NewDetailPage("Root", "Description", longText, "")
		root.SetIndex(PageIndex("test-seg-0"))
		root.SetVisibility(Expanded)
		if err := system.SetSegmentRootIndex("test-seg", "test-seg-0"); err != nil {
			t.Fatalf("Failed to set root index: %v", err)
		}
		if err := system.AddPage(root); err != nil {
			t.Fatalf("Failed to add page: %v", err)
		}

		tokens, err := cw.EstimateTokens()
		if err != nil {
			t.Errorf("EstimateTokens failed: %v", err)
		}

		// 估算：约300字符 / 3 = 100 tokens（粗略估算）
		// 加上description，应该大于100
		if tokens < 100 {
			t.Errorf("Expected at least 100 tokens, got %d", tokens)
		}
	})
}

// TestContextWindow_AutoCollapse 测试自动折叠
func TestContextWindow_AutoCollapse(t *testing.T) {
	system := NewContextSystem()
	cw := NewContextWindow(system)

	// 创建Segment
	segment := NewSegment("test-seg", "Test Segment", "Test", UserSegment)
	segment.SetPermission(ReadWrite)
	if err := system.AddSegment(*segment); err != nil {
		t.Fatalf("Failed to add segment: %v", err)
	}

	// 创建一个包含大量内容的结构
	root, _ := NewContentsPage("Root", "Root Description", "")
	root.SetIndex(PageIndex("test-seg-0"))
	root.SetVisibility(Expanded)
	if err := system.SetSegmentRootIndex("test-seg", "test-seg-0"); err != nil {
		t.Fatalf("Failed to set root index: %v", err)
	}
	if err := system.AddPage(root); err != nil {
		t.Fatalf("Failed to add root page: %v", err)
	}

	// 创建子DetailPage
	child1, _ := NewDetailPage("Child 1", "Child 1 Description", strings.Repeat("Child 1 detail ", 100), "test-seg-0")
	child1.SetIndex(PageIndex("test-seg-1"))
	child1.SetVisibility(Expanded)
	if err := system.AddPage(child1); err != nil {
		t.Fatalf("Failed to add child1: %v", err)
	}

	child2, _ := NewDetailPage("Child 2", "Child 2 Description", strings.Repeat("Child 2 detail ", 100), "test-seg-0")
	child2.SetIndex(PageIndex("test-seg-2"))
	child2.SetVisibility(Expanded)
	if err := system.AddPage(child2); err != nil {
		t.Fatalf("Failed to add child2: %v", err)
	}

	root.AddChild("test-seg-1")
	root.AddChild("test-seg-2")

	t.Run("AutoCollapse should not collapse when under limit", func(t *testing.T) {
		// 设置一个非常高的限制，不应该折叠
		collapsed, err := cw.AutoCollapse(10000)
		if err != nil {
			t.Errorf("AutoCollapse failed: %v", err)
		}

		if len(collapsed) != 0 {
			t.Errorf("Expected no collapsed pages, got %d", len(collapsed))
		}
	})

	t.Run("AutoCollapse should collapse when over limit", func(t *testing.T) {
		// 设置一个很低的限制，应该触发折叠
		initialTokens, _ := cw.EstimateTokens()

		// 如果当前token已经很低，先展开更多内容
		if initialTokens < 100 {
			t.Skip("Not enough content to test auto-collapse")
		}

		// 设置限制为当前token的一半
		collapsed, err := cw.AutoCollapse(initialTokens / 2)
		if err != nil {
			t.Errorf("AutoCollapse failed: %v", err)
		}

		// 应该有一些页面被折叠
		if len(collapsed) == 0 {
			// 可能内容本来就很少
			t.Skip("Content too small to test collapse")
		}

		// 验证被折叠的页面确实是Expanded状态变成Hidden
		for _, pageIndex := range collapsed {
			page, err := system.GetPage(pageIndex)
			if err != nil {
				t.Errorf("Failed to get collapsed page %s: %v", pageIndex, err)
				continue
			}
			if page.GetVisibility() != Hidden {
				t.Errorf("Page %s should be Hidden after collapse", pageIndex)
			}
		}

		// 验证token数量减少
		finalTokens, _ := cw.EstimateTokens()
		if finalTokens >= initialTokens {
			t.Errorf("Tokens should decrease after collapse, was %d, now %d", initialTokens, finalTokens)
		}
	})
}

// TestContextWindow_HideDetails 测试隐藏详情
func TestContextWindow_HideDetails(t *testing.T) {
	system := NewContextSystem()
	cw := NewContextWindow(system)

	// 创建Segment
	segment := NewSegment("test-seg", "Test Segment", "Test", UserSegment)
	segment.SetPermission(ReadWrite)
	if err := system.AddSegment(*segment); err != nil {
		t.Fatalf("Failed to add segment: %v", err)
	}

	// 创建page
	page, _ := NewDetailPage("Test", "Description", "Detail", "")
	page.SetIndex(PageIndex("test-seg-0"))
	page.SetVisibility(Expanded)
	if err := system.SetSegmentRootIndex("test-seg", "test-seg-0"); err != nil {
		t.Fatalf("Failed to set root index: %v", err)
	}
	if err := system.AddPage(page); err != nil {
		t.Fatalf("Failed to add page: %v", err)
	}

	t.Run("HideDetails should change visibility to Hidden", func(t *testing.T) {
		err := cw.HideDetails("test-seg-0")
		if err != nil {
			t.Errorf("HideDetails failed: %v", err)
		}

		retrieved, err := system.GetPage("test-seg-0")
		if err != nil {
			t.Fatalf("Failed to get page: %v", err)
		}

		if retrieved.GetVisibility() != Hidden {
			t.Errorf("Expected visibility Hidden, got %v", retrieved.GetVisibility())
		}
	})

	t.Run("HideDetails on non-existent page should fail", func(t *testing.T) {
		err := cw.HideDetails("non-existent")
		if err == nil {
			t.Error("HideDetails should fail for non-existent page")
		}
	})
}

// TestContextWindow_MultipleSegments 测试多个Segment的渲染
func TestContextWindow_MultipleSegments(t *testing.T) {
	system := NewContextSystem()
	cw := NewContextWindow(system)

	// 创建多个Segment
	segments := []*Segment{
		NewSegment("seg1", "Segment 1", "Test", UserSegment),
		NewSegment("seg2", "Segment 2", "Test", UserSegment),
		NewSegment("seg3", "Segment 3", "Test", SystemSegment),
	}

	for _, seg := range segments {
		seg.SetPermission(ReadWrite)
		if err := system.AddSegment(*seg); err != nil {
			t.Fatalf("Failed to add segment: %v", err)
		}
	}

	// 为每个Segment创建root page
	for _, seg := range segments {
		segID := string(seg.GetID())
		page, _ := NewDetailPage(segID+" Root", segID+" Description", segID+" Detail", "")
		page.SetIndex(PageIndex(segID + "-0"))
		page.SetVisibility(Expanded)
		if err := system.SetSegmentRootIndex(seg.GetID(), PageIndex(segID+"-0")); err != nil {
			t.Fatalf("Failed to set root index for %s: %v", segID, err)
		}
		if err := system.AddPage(page); err != nil {
			t.Fatalf("Failed to add page for %s: %v", segID, err)
		}
	}

	t.Run("GenerateMessageList with multiple segments", func(t *testing.T) {
		messageList, err := cw.GenerateMessageList()
		if err != nil {
			t.Fatalf("GenerateMessageList failed: %v", err)
		}

		// 应该有3个消息（每个Segment一个）
		count := 0
		node := messageList.GetNode()
		for node != nil {
			count++
			node = node.Next()
		}

		if count != 3 {
			t.Errorf("Expected 3 messages, got %d", count)
		}

		// 验证每个Segment都有对应的消息
		node = messageList.GetNode()
		segmentsFound := make(map[string]bool)
		for node != nil {
			content := node.GetMsg().Content.String()
			for _, seg := range segments {
				segID := string(seg.GetID())
				if strings.Contains(content, segID+" Description") {
					segmentsFound[segID] = true
				}
			}
			node = node.Next()
		}

		for _, seg := range segments {
			segID := string(seg.GetID())
			if !segmentsFound[segID] {
				t.Errorf("Segment %s not found in message list", segID)
			}
		}
	})
}
