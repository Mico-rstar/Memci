package context

import (
	"fmt"
	"sync"
)

// PageStorage Page持久化接口
type PageStorage interface {
	// Save 保存Page
	Save(page Page) error

	// Load 加载Page
	Load(pageIndex PageIndex) (Page, error)

	// Delete 删除Page
	Delete(pageIndex PageIndex) error

	// Exists 检查Page是否存在
	Exists(pageIndex PageIndex) bool

	// List 列出所有Page索引
	List() ([]PageIndex, error)
}

// MemoryStorage 内存存储实现（默认）
type MemoryStorage struct {
	pages map[PageIndex]Page
	mu    sync.RWMutex
}

// NewMemoryStorage 创建新的内存存储
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		pages: make(map[PageIndex]Page),
	}
}

// Save 保存Page到内存
func (ms *MemoryStorage) Save(page Page) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.pages[page.GetIndex()] = page
	return nil
}

// Load 从内存加载Page
func (ms *MemoryStorage) Load(pageIndex PageIndex) (Page, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	page, exists := ms.pages[pageIndex]
	if !exists {
		return nil, fmt.Errorf("page %s not found", pageIndex)
	}
	return page, nil
}

// Delete 从内存删除Page
func (ms *MemoryStorage) Delete(pageIndex PageIndex) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	delete(ms.pages, pageIndex)
	return nil
}

// Exists 检查Page是否存在
func (ms *MemoryStorage) Exists(pageIndex PageIndex) bool {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	_, exists := ms.pages[pageIndex]
	return exists
}

// List 列出所有Page索引
func (ms *MemoryStorage) List() ([]PageIndex, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	indices := make([]PageIndex, 0, len(ms.pages))
	for index := range ms.pages {
		indices = append(indices, index)
	}
	return indices, nil
}

// Count 返回Page数量（辅助方法，用于测试）
func (ms *MemoryStorage) Count() int {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return len(ms.pages)
}
