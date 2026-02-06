package context

import (
	"testing"
)

// TestNewAgentContext 测试创建AgentContext
func TestNewAgentContext(t *testing.T) {
	system := NewContextSystem()
	ac := NewAgentContext(system)

	if ac == nil {
		t.Fatal("NewAgentContext returned nil")
	}

	if ac.system != system {
		t.Error("AgentContext system not set correctly")
	}

	if ac.createdAt.IsZero() {
		t.Error("createdAt not set")
	}

	if ac.updatedAt.IsZero() {
		t.Error("updatedAt not set")
	}
}

// TestAgentContext_ReadOnlySegment 测试只读Segment的权限控制
func TestAgentContext_ReadOnlySegment(t *testing.T) {
	system := NewContextSystem()
	ac := NewAgentContext(system)

	// 创建只读Segment
	segment := NewSegment("readonly-seg", "Read Only Segment", "Test", UserSegment)
	segment.SetPermission(ReadOnly)
	if err := system.AddSegment(*segment); err != nil {
		t.Fatalf("Failed to add segment: %v", err)
	}

	// 创建root page (使用ContentsPage，这样可以有子节点)
	rootPage, _ := NewContentsPage("Root", "Root description", "")
	rootPage.SetIndex(PageIndex("readonly-seg-0"))
	if err := system.SetSegmentRootIndex("readonly-seg", rootPage.GetIndex()); err != nil {
		t.Fatalf("Failed to set root index: %v", err)
	}
	if err := system.AddPage(rootPage); err != nil {
		t.Fatalf("Failed to add root page: %v", err)
	}

	rootIndex := rootPage.GetIndex()

	// 测试只读操作（应该成功）
	t.Run("GetPage should succeed", func(t *testing.T) {
		_, err := ac.GetPage(rootIndex)
		if err != nil {
			t.Errorf("GetPage failed: %v", err)
		}
	})

	t.Run("GetChildren should succeed", func(t *testing.T) {
		_, err := ac.GetChildren(rootIndex)
		if err != nil {
			t.Errorf("GetChildren failed: %v", err)
		}
	})

	// 测试写操作（应该失败）
	t.Run("UpdatePage should fail", func(t *testing.T) {
		err := ac.UpdatePage(rootIndex, "New Name", "New Description")
		if err == nil {
			t.Error("UpdatePage should fail for ReadOnly segment")
		}
	})

	t.Run("ExpandDetails should fail", func(t *testing.T) {
		err := ac.ExpandDetails(rootIndex)
		if err == nil {
			t.Error("ExpandDetails should fail for ReadOnly segment")
		}
	})

	t.Run("HideDetails should fail", func(t *testing.T) {
		err := ac.HideDetails(rootIndex)
		if err == nil {
			t.Error("HideDetails should fail for ReadOnly segment")
		}
	})

	t.Run("CreateDetailPage should fail", func(t *testing.T) {
		_, err := ac.CreateDetailPage("New Page", "Description", "Detail", rootIndex)
		if err == nil {
			t.Error("CreateDetailPage should fail for ReadOnly segment")
		}
	})

	t.Run("RemovePage should fail", func(t *testing.T) {
		err := ac.RemovePage(rootIndex)
		if err == nil {
			t.Error("RemovePage should fail for ReadOnly segment")
		}
	})
}

// TestAgentContext_ReadWriteSegment 测试读写Segment的权限控制
func TestAgentContext_ReadWriteSegment(t *testing.T) {
	system := NewContextSystem()
	ac := NewAgentContext(system)

	// 创建读写Segment
	segment := NewSegment("rw-seg", "Read Write Segment", "Test", UserSegment)
	segment.SetPermission(ReadWrite)
	if err := system.AddSegment(*segment); err != nil {
		t.Fatalf("Failed to add segment: %v", err)
	}

	// 创建root page (使用ContentsPage，这样可以有子节点)
	rootPage, _ := NewContentsPage("Root", "Root description", "")
	rootPage.SetIndex(PageIndex("rw-seg-0"))
	if err := system.SetSegmentRootIndex("rw-seg", rootPage.GetIndex()); err != nil {
		t.Fatalf("Failed to set root index: %v", err)
	}
	if err := system.AddPage(rootPage); err != nil {
		t.Fatalf("Failed to add root page: %v", err)
	}
	rootIndex := rootPage.GetIndex()

	// 测试写操作（应该成功）
	t.Run("UpdatePage should succeed", func(t *testing.T) {
		err := ac.UpdatePage(rootIndex, "New Name", "New Description")
		if err != nil {
			t.Errorf("UpdatePage failed: %v", err)
		}

		page, _ := system.GetPage(rootIndex)
		if page.GetName() != "New Name" {
			t.Errorf("Expected name 'New Name', got '%s'", page.GetName())
		}
	})

	t.Run("ExpandDetails should succeed", func(t *testing.T) {
		err := ac.ExpandDetails(rootIndex)
		if err != nil {
			t.Errorf("ExpandDetails failed: %v", err)
		}

		page, _ := system.GetPage(rootIndex)
		if page.GetVisibility() != Expanded {
			t.Error("Page should be Expanded")
		}
	})

	t.Run("HideDetails should succeed", func(t *testing.T) {
		err := ac.HideDetails(rootIndex)
		if err != nil {
			t.Errorf("HideDetails failed: %v", err)
		}

		page, _ := system.GetPage(rootIndex)
		if page.GetVisibility() != Hidden {
			t.Error("Page should be Hidden")
		}
	})

	t.Run("CreateDetailPage should succeed", func(t *testing.T) {
		newIndex, err := ac.CreateDetailPage("Child", "Child description", "Child detail", rootIndex)
		if err != nil {
			t.Errorf("CreateDetailPage failed: %v", err)
		}

		if newIndex == "" {
			t.Error("Expected non-empty page index")
		}
	})

	t.Run("CreateContentsPage should succeed", func(t *testing.T) {
		newIndex, err := ac.CreateContentsPage("Contents", "Contents description", rootIndex)
		if err != nil {
			t.Errorf("CreateContentsPage failed: %v", err)
		}

		if newIndex == "" {
			t.Error("Expected non-empty page index")
		}
	})
}

// TestAgentContext_SystemSegmentProtection 测试系统Segment的保护
func TestAgentContext_SystemSegmentProtection(t *testing.T) {
	system := NewContextSystem()
	ac := NewAgentContext(system)

	// 创建系统Segment
	segment := NewSegment("system-seg", "System Segment", "Test", SystemSegment)
	segment.SetPermission(SystemManaged)
	if err := system.AddSegment(*segment); err != nil {
		t.Fatalf("Failed to add segment: %v", err)
	}

	// 创建root page (手动创建，使用ContentsPage以便有子节点)
	rootPage, _ := NewContentsPage("System Root", "System root description", "")
	rootPage.SetIndex(PageIndex("system-seg-0"))
	if err := system.SetSegmentRootIndex("system-seg", rootPage.GetIndex()); err != nil {
		t.Fatalf("Failed to set root index: %v", err)
	}
	if err := system.AddPage(rootPage); err != nil {
		t.Fatalf("Failed to add root page: %v", err)
	}
	rootIndex := rootPage.GetIndex()

	// 创建子page
	childIndex, err := system.createDetailPageInternal("System Child", "Child description", "Child detail", rootIndex)
	if err != nil {
		t.Fatalf("Failed to create child page: %v", err)
	}

	// 测试：禁止隐藏系统Segment的root page
	t.Run("HideDetails on system root should fail", func(t *testing.T) {
		err := ac.HideDetails(rootIndex)
		if err == nil {
			t.Error("HideDetails should fail for system segment root page")
		}
	})

	// 测试：可以隐藏系统Segment的非root page
	t.Run("HideDetails on system child should succeed", func(t *testing.T) {
		err := ac.HideDetails(childIndex)
		if err != nil {
			t.Errorf("HideDetails should succeed for system segment child page: %v", err)
		}
	})

	// 测试：可以展开系统Segment的root page
	t.Run("ExpandDetails on system root should succeed", func(t *testing.T) {
		err := ac.ExpandDetails(rootIndex)
		if err != nil {
			t.Errorf("ExpandDetails should succeed for system segment root page: %v", err)
		}
	})
}

// TestAgentContext_MovePage 测试移动Page的权限检查
func TestAgentContext_MovePage(t *testing.T) {
	system := NewContextSystem()
	ac := NewAgentContext(system)

	// 创建两个Segment：一个只读，一个读写
	readonlySeg := NewSegment("readonly-seg", "Read Only Segment", "Test", UserSegment)
	readonlySeg.SetPermission(ReadOnly)
	if err := system.AddSegment(*readonlySeg); err != nil {
		t.Fatalf("Failed to add readonly segment: %v", err)
	}

	rwSeg := NewSegment("rw-seg", "Read Write Segment", "Test", UserSegment)
	rwSeg.SetPermission(ReadWrite)
	if err := system.AddSegment(*rwSeg); err != nil {
		t.Fatalf("Failed to add rw segment: %v", err)
	}

	// 创建各自的root page (手动创建，使用ContentsPage以便有子节点)
	roRootPage, _ := NewContentsPage("RO Root", "RO description", "")
	roRootPage.SetIndex(PageIndex("readonly-seg-0"))
	if err := system.SetSegmentRootIndex("readonly-seg", roRootPage.GetIndex()); err != nil {
		t.Fatalf("Failed to set RO root index: %v", err)
	}
	if err := system.AddPage(roRootPage); err != nil {
		t.Fatalf("Failed to add RO root page: %v", err)
	}
	roRoot := roRootPage.GetIndex()

	rwRootPage, _ := NewContentsPage("RW Root", "RW description", "")
	rwRootPage.SetIndex(PageIndex("rw-seg-0"))
	if err := system.SetSegmentRootIndex("rw-seg", rwRootPage.GetIndex()); err != nil {
		t.Fatalf("Failed to set RW root index: %v", err)
	}
	if err := system.AddPage(rwRootPage); err != nil {
		t.Fatalf("Failed to add RW root page: %v", err)
	}
	rwRoot := rwRootPage.GetIndex()

	// 创建一个ContentsPage作为移动目标
	rwContents, err := system.createContentsPageInternal("RW Contents", "RW contents description", rwRoot)
	if err != nil {
		t.Fatalf("Failed to create RW contents: %v", err)
	}

	// 测试：从只读Segment移动应该失败
	t.Run("MovePage from ReadOnly should fail", func(t *testing.T) {
		err := ac.MovePage(roRoot, rwContents)
		if err == nil {
			t.Error("MovePage from ReadOnly segment should fail")
		}
	})

	// 测试：移动到只读Segment应该失败
	t.Run("MovePage to ReadOnly should fail", func(t *testing.T) {
		roContents, err := system.createContentsPageInternal("RO Contents", "RO contents description", roRoot)
		if err != nil {
			t.Fatalf("Failed to create RO contents: %v", err)
		}

		err = ac.MovePage(rwContents, roContents)
		if err == nil {
			t.Error("MovePage to ReadOnly segment should fail")
		}
	})

	// 测试：在读写Segment内移动应该成功
	t.Run("MovePage within ReadWrite should succeed", func(t *testing.T) {
		child1, _ := system.createDetailPageInternal("Child 1", "Child 1 description", "Child 1 detail", rwContents)

		// 创建另一个ContentsPage
		newContents, _ := system.createContentsPageInternal("New Contents", "New contents description", rwRoot)

		err := ac.MovePage(child1, newContents)
		if err != nil {
			t.Errorf("MovePage within ReadWrite segment should succeed: %v", err)
		}

		// 验证移动成功
		parent, _ := system.GetParent(child1)
		if parent.GetIndex() != newContents {
			t.Error("Page not moved to correct parent")
		}
	})
}

// TestAgentContext_GetSegment 测试获取Segment
func TestAgentContext_GetSegment(t *testing.T) {
	system := NewContextSystem()
	ac := NewAgentContext(system)

	segment := NewSegment("test-seg", "Test Segment", "Test", UserSegment)
	segment.SetPermission(ReadWrite)
	if err := system.AddSegment(*segment); err != nil {
		t.Fatalf("Failed to add segment: %v", err)
	}

	t.Run("GetSegment should return segment", func(t *testing.T) {
		retrieved, err := ac.GetSegment("test-seg")
		if err != nil {
			t.Errorf("GetSegment failed: %v", err)
		}

		if retrieved.GetID() != "test-seg" {
			t.Errorf("Expected ID 'test-seg', got '%s'", retrieved.GetID())
		}
	})

	t.Run("GetSegment with invalid ID should fail", func(t *testing.T) {
		_, err := ac.GetSegment("invalid-seg")
		if err == nil {
			t.Error("GetSegment should fail for invalid ID")
		}
	})
}

// TestAgentContext_ListSegments 测试列出所有Segment
func TestAgentContext_ListSegments(t *testing.T) {
	system := NewContextSystem()
	ac := NewAgentContext(system)

	// 添加多个Segment
	segments := []*Segment{
		NewSegment("seg1", "Segment 1", "Test", UserSegment),
		NewSegment("seg2", "Segment 2", "Test", UserSegment),
		NewSegment("seg3", "Segment 3", "Test", SystemSegment),
	}
	segments[0].SetPermission(ReadOnly)
	segments[1].SetPermission(ReadWrite)
	segments[2].SetPermission(SystemManaged)

	for _, seg := range segments {
		if err := system.AddSegment(*seg); err != nil {
			t.Fatalf("Failed to add segment: %v", err)
		}
	}

	list, err := ac.ListSegments()
	if err != nil {
		t.Fatalf("ListSegments failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("Expected 3 segments, got %d", len(list))
	}
}

// TestAgentContext_GetParent 测试获取父Page
func TestAgentContext_GetParent(t *testing.T) {
	system := NewContextSystem()
	ac := NewAgentContext(system)

	segment := NewSegment("test-seg", "Test Segment", "Test", UserSegment)
	segment.SetPermission(ReadWrite)
	if err := system.AddSegment(*segment); err != nil {
		t.Fatalf("Failed to add segment: %v", err)
	}

	// 手动创建root page (使用ContentsPage以便有子节点)
	rootPage, _ := NewContentsPage("Root", "Root description", "")
	rootPage.SetIndex(PageIndex("test-seg-0"))
	if err := system.SetSegmentRootIndex("test-seg", rootPage.GetIndex()); err != nil {
		t.Fatalf("Failed to set root index: %v", err)
	}
	if err := system.AddPage(rootPage); err != nil {
		t.Fatalf("Failed to add root page: %v", err)
	}
	root := rootPage.GetIndex()

	child, _ := system.createDetailPageInternal("Child", "Child description", "Child detail", root)

	t.Run("GetParent should succeed", func(t *testing.T) {
		parent, err := ac.GetParent(child)
		if err != nil {
			t.Errorf("GetParent failed: %v", err)
		}

		if parent.GetIndex() != root {
			t.Errorf("Expected parent index '%s', got '%s'", root, parent.GetIndex())
		}
	})

	t.Run("GetParent on root should fail", func(t *testing.T) {
		_, err := ac.GetParent(root)
		if err == nil {
			t.Error("GetParent on root page should fail")
		}
	})
}

// TestAgentContext_GetAncestors 测试获取祖先Page列表
func TestAgentContext_GetAncestors(t *testing.T) {
	system := NewContextSystem()
	ac := NewAgentContext(system)

	segment := NewSegment("test-seg", "Test Segment", "Test", UserSegment)
	segment.SetPermission(ReadWrite)
	if err := system.AddSegment(*segment); err != nil {
		t.Fatalf("Failed to add segment: %v", err)
	}

	// 创建三层结构
	rootPage, _ := NewContentsPage("Root", "Root description", "")
	rootPage.SetIndex(PageIndex("test-seg-0"))
	if err := system.SetSegmentRootIndex("test-seg", rootPage.GetIndex()); err != nil {
		t.Fatalf("Failed to set root index: %v", err)
	}
	if err := system.AddPage(rootPage); err != nil {
		t.Fatalf("Failed to add root page: %v", err)
	}
	root := rootPage.GetIndex()

	middle, _ := system.createContentsPageInternal("Middle", "Middle description", root)
	leaf, _ := system.createDetailPageInternal("Leaf", "Leaf description", "Leaf detail", middle)

	t.Run("GetAncestors should return all ancestors", func(t *testing.T) {
		ancestors, err := ac.GetAncestors(leaf)
		if err != nil {
			t.Errorf("GetAncestors failed: %v", err)
		}

		if len(ancestors) != 2 {
			t.Errorf("Expected 2 ancestors, got %d", len(ancestors))
		}

		if ancestors[0].GetIndex() != middle {
			t.Errorf("Expected first ancestor '%s', got '%s'", middle, ancestors[0].GetIndex())
		}

		if ancestors[1].GetIndex() != root {
			t.Errorf("Expected second ancestor '%s', got '%s'", root, ancestors[1].GetIndex())
		}
	})
}

// TestAgentContext_FindPage 测试查找Page
func TestAgentContext_FindPage(t *testing.T) {
	system := NewContextSystem()
	ac := NewAgentContext(system)

	segment := NewSegment("test-seg", "Test Segment", "Test", UserSegment)
	segment.SetPermission(ReadWrite)
	if err := system.AddSegment(*segment); err != nil {
		t.Fatalf("Failed to add segment: %v", err)
	}

	// 手动创建root page
	rootPage, _ := NewDetailPage("SearchTarget", "This is a searchable page", "Detail", "")
	rootPage.SetIndex(PageIndex("test-seg-0"))
	if err := system.SetSegmentRootIndex("test-seg", rootPage.GetIndex()); err != nil {
		t.Fatalf("Failed to set root index: %v", err)
	}
	if err := system.AddPage(rootPage); err != nil {
		t.Fatalf("Failed to add root page: %v", err)
	}
	root := rootPage.GetIndex()

	t.Run("FindPage by name should return results", func(t *testing.T) {
		results := ac.FindPage("SearchTarget")
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}

		if results[0].GetIndex() != root {
			t.Error("Found wrong page")
		}
	})

	t.Run("FindPage by description should return results", func(t *testing.T) {
		results := ac.FindPage("searchable")
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
	})

	t.Run("FindPage with no matches should return empty", func(t *testing.T) {
		results := ac.FindPage("nonexistent")
		if len(results) != 0 {
			t.Errorf("Expected 0 results, got %d", len(results))
		}
	})
}
