package context

import (
	"fmt"
	"testing"
)

// TestNewMemoryStorage 测试创建内存存储
func TestNewMemoryStorage(t *testing.T) {
	storage := NewMemoryStorage()

	if storage == nil {
		t.Fatal("NewMemoryStorage returned nil")
	}

	if storage.Count() != 0 {
		t.Errorf("Expected empty storage, got count %d", storage.Count())
	}
}

// TestMemoryStorage_Save 测试保存Page
func TestMemoryStorage_Save(t *testing.T) {
	storage := NewMemoryStorage()

	page, err := NewDetailPage("Test Page", "Test description", "Test detail", "")
	if err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}
	page.SetIndex(PageIndex("test-1"))

	err = storage.Save(page)
	if err != nil {
		t.Fatalf("Failed to save page: %v", err)
	}

	if !storage.Exists(page.GetIndex()) {
		t.Error("Page should exist after save")
	}

	if storage.Count() != 1 {
		t.Errorf("Expected count 1, got %d", storage.Count())
	}
}

// TestMemoryStorage_Load 测试加载Page
func TestMemoryStorage_Load(t *testing.T) {
	storage := NewMemoryStorage()

	original, err := NewDetailPage("Test Page", "Test description", "Test detail", "")
	if err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}
	original.SetIndex(PageIndex("test-1"))

	// 保存
	if err := storage.Save(original); err != nil {
		t.Fatalf("Failed to save page: %v", err)
	}

	// 加载
	loaded, err := storage.Load(PageIndex("test-1"))
	if err != nil {
		t.Fatalf("Failed to load page: %v", err)
	}

	// 验证内容
	if loaded.GetIndex() != original.GetIndex() {
		t.Errorf("Expected index %s, got %s", original.GetIndex(), loaded.GetIndex())
	}

	if loaded.GetName() != original.GetName() {
		t.Errorf("Expected name %s, got %s", original.GetName(), loaded.GetName())
	}

	if loaded.GetDescription() != original.GetDescription() {
		t.Errorf("Expected description %s, got %s", original.GetDescription(), loaded.GetDescription())
	}
}

// TestMemoryStorage_Load_NotFound 测试加载不存在的Page
func TestMemoryStorage_Load_NotFound(t *testing.T) {
	storage := NewMemoryStorage()

	_, err := storage.Load(PageIndex("nonexistent"))
	if err == nil {
		t.Error("Expected error when loading non-existent page")
	}
}

// TestMemoryStorage_Delete 测试删除Page
func TestMemoryStorage_Delete(t *testing.T) {
	storage := NewMemoryStorage()

	page, err := NewDetailPage("Test Page", "Test description", "Test detail", "")
	if err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}
	page.SetIndex(PageIndex("test-1"))

	// 保存
	if err := storage.Save(page); err != nil {
		t.Fatalf("Failed to save page: %v", err)
	}

	// 验证存在
	if !storage.Exists(page.GetIndex()) {
		t.Fatal("Page should exist before delete")
	}

	// 删除
	if err := storage.Delete(page.GetIndex()); err != nil {
		t.Fatalf("Failed to delete page: %v", err)
	}

	// 验证不存在
	if storage.Exists(page.GetIndex()) {
		t.Error("Page should not exist after delete")
	}

	if storage.Count() != 0 {
		t.Errorf("Expected count 0, got %d", storage.Count())
	}
}

// TestMemoryStorage_Delete_NotFound 测试删除不存在的Page（不应报错）
func TestMemoryStorage_Delete_NotFound(t *testing.T) {
	storage := NewMemoryStorage()

	// 删除不存在的Page不应该报错
	err := storage.Delete(PageIndex("nonexistent"))
	if err != nil {
		t.Errorf("Delete should not error for non-existent page, got: %v", err)
	}
}

// TestMemoryStorage_Exists 测试检查Page是否存在
func TestMemoryStorage_Exists(t *testing.T) {
	storage := NewMemoryStorage()

	page, err := NewDetailPage("Test Page", "Test description", "Test detail", "")
	if err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}
	page.SetIndex(PageIndex("test-1"))

	// 保存前不存在
	if storage.Exists(page.GetIndex()) {
		t.Error("Page should not exist before save")
	}

	// 保存
	if err := storage.Save(page); err != nil {
		t.Fatalf("Failed to save page: %v", err)
	}

	// 保存后存在
	if !storage.Exists(page.GetIndex()) {
		t.Error("Page should exist after save")
	}
}

// TestMemoryStorage_List 测试列出所有Page索引
func TestMemoryStorage_List(t *testing.T) {
	storage := NewMemoryStorage()

	// 空存储
	list, err := storage.List()
	if err != nil {
		t.Fatalf("Failed to list pages: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("Expected empty list, got %d items", len(list))
	}

	// 添加多个Page
	for i := 1; i <= 3; i++ {
		page, _ := NewDetailPage("Page", "Description", "Detail", "")
		page.SetIndex(PageIndex(fmt.Sprintf("test-%d", i)))
		if err := storage.Save(page); err != nil {
			t.Fatalf("Failed to save page: %v", err)
		}
	}

	// 列出
	list, err = storage.List()
	if err != nil {
		t.Fatalf("Failed to list pages: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("Expected 3 items, got %d", len(list))
	}
}

// TestMemoryStorage_ConcurrentAccess 测试并发访问
func TestMemoryStorage_ConcurrentAccess(t *testing.T) {
	storage := NewMemoryStorage()
	done := make(chan bool)

	// 并发写入
	for i := 0; i < 10; i++ {
		go func(idx int) {
			page, _ := NewDetailPage("Page", "Description", "Detail", "")
			page.SetIndex(PageIndex(fmt.Sprintf("test-%d", idx)))
			storage.Save(page)
			done <- true
		}(i)
	}

	// 等待所有写入完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证数量
	if storage.Count() != 10 {
		t.Errorf("Expected 10 pages, got %d", storage.Count())
	}

	// 并发读取
	for i := 0; i < 10; i++ {
		go func(idx int) {
			index := PageIndex(fmt.Sprintf("test-%d", idx))
			storage.Exists(index)
			storage.Load(index)
			done <- true
		}(i)
	}

	// 等待所有读取完成
	for i := 0; i < 10; i++ {
		<-done
	}
}
