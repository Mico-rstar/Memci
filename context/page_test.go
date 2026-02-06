package context

import (
	"testing"
	"time"
)

func TestNewDetailPage(t *testing.T) {
	tests := []struct {
		name        string
		pageName    string
		description string
		detail      string
		parentIndex PageIndex
		wantErr     bool
	}{
		{
			name:        "valid detail page",
			pageName:    "test page",
			description: "test description",
			detail:      "test detail content",
			parentIndex: "0",
			wantErr:     false,
		},
		{
			name:        "empty name should error",
			pageName:    "",
			description: "test description",
			detail:      "test detail",
			parentIndex: "0",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := NewDetailPage(tt.pageName, tt.description, tt.detail, tt.parentIndex)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDetailPage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if page.GetName() != tt.pageName {
					t.Errorf("GetName() = %v, want %v", page.GetName(), tt.pageName)
				}
				if page.GetDescription() != tt.description {
					t.Errorf("GetDescription() = %v, want %v", page.GetDescription(), tt.description)
				}
				if page.GetDetail() != tt.detail {
					t.Errorf("GetDetail() = %v, want %v", page.GetDetail(), tt.detail)
				}
				if page.GetParent() != tt.parentIndex {
					t.Errorf("GetParent() = %v, want %v", page.GetParent(), tt.parentIndex)
				}
				if page.GetLifecycle() != Active {
					t.Errorf("GetLifecycle() = %v, want %v", page.GetLifecycle(), Active)
				}
				if page.GetVisibility() != Hidden {
					t.Errorf("GetVisibility() = %v, want %v", page.GetVisibility(), Hidden)
				}
			}
		})
	}
}

func TestDetailPage_Setters(t *testing.T) {
	page, _ := NewDetailPage("test", "desc", "detail", "0")

	// Test SetVisibility
	if err := page.SetVisibility(Expanded); err != nil {
		t.Errorf("SetVisibility() error = %v", err)
	}
	if page.GetVisibility() != Expanded {
		t.Errorf("GetVisibility() = %v, want %v", page.GetVisibility(), Expanded)
	}

	// Test SetLifecycle
	if err := page.SetLifecycle(HotArchived); err != nil {
		t.Errorf("SetLifecycle() error = %v", err)
	}
	if page.GetLifecycle() != HotArchived {
		t.Errorf("GetLifecycle() = %v, want %v", page.GetLifecycle(), HotArchived)
	}

	// Test SetDescription
	newDesc := "new description"
	if err := page.SetDescription(newDesc); err != nil {
		t.Errorf("SetDescription() error = %v", err)
	}
	if page.GetDescription() != newDesc {
		t.Errorf("GetDescription() = %v, want %v", page.GetDescription(), newDesc)
	}

	// Test SetName
	newName := "new name"
	if err := page.SetName(newName); err != nil {
		t.Errorf("SetName() error = %v", err)
	}
	if page.GetName() != newName {
		t.Errorf("GetName() = %v, want %v", page.GetName(), newName)
	}

	// Test SetName with empty string
	if err := page.SetName(""); err == nil {
		t.Error("SetName() with empty string should return error")
	}

	// Test SetParent
	newParent := PageIndex("1")
	if err := page.SetParent(newParent); err != nil {
		t.Errorf("SetParent() error = %v", err)
	}
	if page.GetParent() != newParent {
		t.Errorf("GetParent() = %v, want %v", page.GetParent(), newParent)
	}

	// Test SetDetail
	newDetail := "new detail content"
	if err := page.SetDetail(newDetail); err != nil {
		t.Errorf("SetDetail() error = %v", err)
	}
	if page.GetDetail() != newDetail {
		t.Errorf("GetDetail() = %v, want %v", page.GetDetail(), newDetail)
	}
}

func TestDetailPage_MarshalUnmarshal(t *testing.T) {
	originalPage, _ := NewDetailPage("test page", "test description", "test detail", "0")
	originalPage.SetIndex(PageIndex("test-index"))
	originalPage.SetVisibility(Expanded)
	originalPage.SetLifecycle(Active)

	// Marshal
	data, err := originalPage.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Unmarshal
	restorePage := &DetailPage{}
	err = restorePage.Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Verify all fields
	if restorePage.GetIndex() != originalPage.GetIndex() {
		t.Errorf("GetIndex() = %v, want %v", restorePage.GetIndex(), originalPage.GetIndex())
	}
	if restorePage.GetName() != originalPage.GetName() {
		t.Errorf("GetName() = %v, want %v", restorePage.GetName(), originalPage.GetName())
	}
	if restorePage.GetDescription() != originalPage.GetDescription() {
		t.Errorf("GetDescription() = %v, want %v", restorePage.GetDescription(), originalPage.GetDescription())
	}
	if restorePage.GetDetail() != originalPage.GetDetail() {
		t.Errorf("GetDetail() = %v, want %v", restorePage.GetDetail(), originalPage.GetDetail())
	}
	if restorePage.GetVisibility() != originalPage.GetVisibility() {
		t.Errorf("GetVisibility() = %v, want %v", restorePage.GetVisibility(), originalPage.GetVisibility())
	}
	if restorePage.GetLifecycle() != originalPage.GetLifecycle() {
		t.Errorf("GetLifecycle() = %v, want %v", restorePage.GetLifecycle(), originalPage.GetLifecycle())
	}
	if restorePage.GetParent() != originalPage.GetParent() {
		t.Errorf("GetParent() = %v, want %v", restorePage.GetParent(), originalPage.GetParent())
	}
}

func TestNewContentsPage(t *testing.T) {
	tests := []struct {
		name        string
		pageName    string
		description string
		parentIndex PageIndex
		wantErr     bool
	}{
		{
			name:        "valid contents page",
			pageName:    "test contents",
			description: "test contents description",
			parentIndex: "0",
			wantErr:     false,
		},
		{
			name:        "empty name should error",
			pageName:    "",
			description: "test description",
			parentIndex: "0",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := NewContentsPage(tt.pageName, tt.description, tt.parentIndex)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewContentsPage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if page.GetName() != tt.pageName {
					t.Errorf("GetName() = %v, want %v", page.GetName(), tt.pageName)
				}
				if page.GetDescription() != tt.description {
					t.Errorf("GetDescription() = %v, want %v", page.GetDescription(), tt.description)
				}
				if page.GetParent() != tt.parentIndex {
					t.Errorf("GetParent() = %v, want %v", page.GetParent(), tt.parentIndex)
				}
				if page.GetLifecycle() != Active {
					t.Errorf("GetLifecycle() = %v, want %v", page.GetLifecycle(), Active)
				}
				if page.GetVisibility() != Hidden {
					t.Errorf("GetVisibility() = %v, want %v", page.GetVisibility(), Hidden)
				}
				if page.ChildCount() != 0 {
					t.Errorf("ChildCount() = %v, want %v", page.ChildCount(), 0)
				}
			}
		})
	}
}

func TestContentsPage_ChildManagement(t *testing.T) {
	page, _ := NewContentsPage("parent", "parent description", "0")

	// Test AddChild
	child1 := PageIndex("child1")
	child2 := PageIndex("child2")

	if err := page.AddChild(child1); err != nil {
		t.Errorf("AddChild() error = %v", err)
	}
	if page.ChildCount() != 1 {
		t.Errorf("ChildCount() = %v, want %v", page.ChildCount(), 1)
	}
	if !page.HasChild(child1) {
		t.Error("HasChild() should return true for child1")
	}

	if err := page.AddChild(child2); err != nil {
		t.Errorf("AddChild() error = %v", err)
	}
	if page.ChildCount() != 2 {
		t.Errorf("ChildCount() = %v, want %v", page.ChildCount(), 2)
	}

	// Test GetChildren
	children := page.GetChildren()
	if len(children) != 2 {
		t.Errorf("GetChildren() length = %v, want %v", len(children), 2)
	}
	if children[0] != child1 || children[1] != child2 {
		t.Errorf("GetChildren() = %v, want [%v, %v]", children, child1, child2)
	}

	// Test AddChild duplicate
	if err := page.AddChild(child1); err == nil {
		t.Error("AddChild() duplicate should return error")
	}

	// Test RemoveChild
	if err := page.RemoveChild(child1); err != nil {
		t.Errorf("RemoveChild() error = %v", err)
	}
	if page.ChildCount() != 1 {
		t.Errorf("ChildCount() = %v, want %v", page.ChildCount(), 1)
	}
	if page.HasChild(child1) {
		t.Error("HasChild() should return false for removed child1")
	}
	if !page.HasChild(child2) {
		t.Error("HasChild() should return true for child2")
	}

	// Test RemoveChild non-existent
	child3 := PageIndex("child3")
	if err := page.RemoveChild(child3); err == nil {
		t.Error("RemoveChild() non-existent child should return error")
	}
}

func TestContentsPage_MarshalUnmarshal(t *testing.T) {
	originalPage, _ := NewContentsPage("test contents", "test description", "0")
	originalPage.SetIndex(PageIndex("contents-index"))
	originalPage.SetVisibility(Expanded)
	originalPage.AddChild(PageIndex("child1"))
	originalPage.AddChild(PageIndex("child2"))

	// Marshal
	data, err := originalPage.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Unmarshal
	restorePage := &ContentsPage{}
	err = restorePage.Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Verify all fields
	if restorePage.GetIndex() != originalPage.GetIndex() {
		t.Errorf("GetIndex() = %v, want %v", restorePage.GetIndex(), originalPage.GetIndex())
	}
	if restorePage.GetName() != originalPage.GetName() {
		t.Errorf("GetName() = %v, want %v", restorePage.GetName(), originalPage.GetName())
	}
	if restorePage.GetDescription() != originalPage.GetDescription() {
		t.Errorf("GetDescription() = %v, want %v", restorePage.GetDescription(), originalPage.GetDescription())
	}
	if restorePage.GetVisibility() != originalPage.GetVisibility() {
		t.Errorf("GetVisibility() = %v, want %v", restorePage.GetVisibility(), originalPage.GetVisibility())
	}
	if restorePage.GetLifecycle() != originalPage.GetLifecycle() {
		t.Errorf("GetLifecycle() = %v, want %v", restorePage.GetLifecycle(), originalPage.GetLifecycle())
	}
	if restorePage.GetParent() != originalPage.GetParent() {
		t.Errorf("GetParent() = %v, want %v", restorePage.GetParent(), originalPage.GetParent())
	}
	if restorePage.ChildCount() != originalPage.ChildCount() {
		t.Errorf("ChildCount() = %v, want %v", restorePage.ChildCount(), originalPage.ChildCount())
	}

	children := restorePage.GetChildren()
	origChildren := originalPage.GetChildren()
	for i := range children {
		if children[i] != origChildren[i] {
			t.Errorf("GetChildren()[%v] = %v, want %v", i, children[i], origChildren[i])
		}
	}
}

func TestPageLifecycle_String(t *testing.T) {
	tests := []struct {
		lifecycle PageLifecycle
		want      string
	}{
		{Active, "Active"},
		{HotArchived, "HotArchived"},
		{ColdArchived, "ColdArchived"},
		{PageLifecycle(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.lifecycle.String(); got != tt.want {
				t.Errorf("PageLifecycle.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPageVisibility_String(t *testing.T) {
	tests := []struct {
		visibility PageVisibility
		want       string
	}{
		{Expanded, "Expanded"},
		{Hidden, "Hidden"},
		{PageVisibility(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.visibility.String(); got != tt.want {
				t.Errorf("PageVisibility.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetailPage_UpdatedAt(t *testing.T) {
	page, _ := NewDetailPage("test", "desc", "detail", "0")
	oldTime := page.updatedAt

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// SetVisibility should update updatedAt
	page.SetVisibility(Expanded)
	if !page.updatedAt.After(oldTime) {
		t.Error("SetVisibility() should update updatedAt")
	}

	oldTime = page.updatedAt
	time.Sleep(10 * time.Millisecond)

	// SetLifecycle should update updatedAt
	page.SetLifecycle(HotArchived)
	if !page.updatedAt.After(oldTime) {
		t.Error("SetLifecycle() should update updatedAt")
	}
}

func TestContentsPage_UpdatedAt(t *testing.T) {
	page, _ := NewContentsPage("test", "desc", "0")
	oldTime := page.updatedAt

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// AddChild should update updatedAt
	page.AddChild(PageIndex("child1"))
	if !page.updatedAt.After(oldTime) {
		t.Error("AddChild() should update updatedAt")
	}

	oldTime = page.updatedAt
	time.Sleep(10 * time.Millisecond)

	// RemoveChild should update updatedAt
	page.RemoveChild(PageIndex("child1"))
	if !page.updatedAt.After(oldTime) {
		t.Error("RemoveChild() should update updatedAt")
	}
}
