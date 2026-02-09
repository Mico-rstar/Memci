package context

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Storage 上下文持久化接口
type Storage interface {
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

	// SaveSegment 保存Segment元数据
	SaveSegment(segment *Segment) error

	// LoadSegment 加载Segment元数据
	LoadSegment(id SegmentID) (*Segment, error)

	// ListSegments 列出所有Segment
	ListSegments() ([]*Segment, error)

	// DeleteSegment 删除Segment元数据
	DeleteSegment(id SegmentID) error
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

// SaveSegment 内存存储不持久化Segment（无操作）
func (ms *MemoryStorage) SaveSegment(segment *Segment) error {
	// 内存存储不需要持久化Segment
	return nil
}

// LoadSegment 内存存储不支持加载Segment
func (ms *MemoryStorage) LoadSegment(id SegmentID) (*Segment, error) {
	return nil, fmt.Errorf("memory storage does not support segment persistence")
}

// ListSegments 内存存储不支持列出Segment
func (ms *MemoryStorage) ListSegments() ([]*Segment, error) {
	return []*Segment{}, nil
}

// DeleteSegment 内存存储不持久化Segment（无操作）
func (ms *MemoryStorage) DeleteSegment(id SegmentID) error {
	// 内存存储不需要持久化Segment
	return nil
}

// pageTypeJSON 用于识别Page类型的通用结构
type pageTypeJSON struct {
	Type string `json:"type"`
}

// FileStorage 文件存储实现
type FileStorage struct {
	dir string // 存储目录
	mu  sync.RWMutex
}

// NewFileStorage 创建新的文件存储
func NewFileStorage(dir string) (*FileStorage, error) {
	// 确保目录存在
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}
	return &FileStorage{
		dir: dir,
	}, nil
}

// filePath 获取Page的文件路径
func (fs *FileStorage) filePath(pageIndex PageIndex) string {
	// 使用安全的文件名
	return filepath.Join(fs.dir, filepath.FromSlash(string(pageIndex))+".json")
}

// Save 保存Page到文件
func (fs *FileStorage) Save(page Page) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := page.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal page: %w", err)
	}

	path := fs.filePath(page.GetIndex())
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

// Load 从文件加载Page
func (fs *FileStorage) Load(pageIndex PageIndex) (Page, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	path := fs.filePath(pageIndex)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("page %s not found", pageIndex)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// 先读取类型字段
	var typeData pageTypeJSON
	if err := json.Unmarshal(data, &typeData); err != nil {
		return nil, fmt.Errorf("failed to parse page type: %w", err)
	}

	var page Page

	// 根据类型字段反序列化
	switch typeData.Type {
	case "detail":
		detailPage := &DetailPage{}
		if err := detailPage.Unmarshal(data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal detail page: %w", err)
		}
		page = detailPage
	case "contents":
		contentsPage := &ContentsPage{}
		if err := contentsPage.Unmarshal(data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal contents page: %w", err)
		}
		page = contentsPage
	default:
		return nil, fmt.Errorf("unknown page type: %s", typeData.Type)
	}

	return page, nil
}

// Delete 删除Page文件
func (fs *FileStorage) Delete(pageIndex PageIndex) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := fs.filePath(pageIndex)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil // 不存在视为成功
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// Exists 检查Page文件是否存在
func (fs *FileStorage) Exists(pageIndex PageIndex) bool {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	path := fs.filePath(pageIndex)
	_, err := os.Stat(path)
	return err == nil
}

// List 列出所有Page索引
func (fs *FileStorage) List() ([]PageIndex, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	entries, err := os.ReadDir(fs.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	indices := make([]PageIndex, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// 跳过 segments 元数据文件
		if name == "segments.json" {
			continue
		}
		// 移除 .json 后缀
		if filepath.Ext(name) == ".json" {
			index := PageIndex(name[:len(name)-5])
			indices = append(indices, index)
		}
	}
	return indices, nil
}

// ============ Segment 持久化方法 ============

// segmentsFilePath 获取Segment元数据文件路径
func (fs *FileStorage) segmentsFilePath() string {
	return filepath.Join(fs.dir, "segments.json")
}

// SaveSegment 保存Segment元数据到文件
func (fs *FileStorage) SaveSegment(segment *Segment) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// 读取所有现有的 segments
	segments, err := fs.loadAllSegments()
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load existing segments: %w", err)
	}

	// 更新或添加当前 segment
	found := false
	for i, seg := range segments {
		if seg.GetID() == segment.GetID() {
			segments[i] = segment
			found = true
			break
		}
	}
	if !found {
		segments = append(segments, segment)
	}

	// 保存所有 segments
	return fs.saveAllSegments(segments)
}

// LoadSegment 加载单个Segment元数据
func (fs *FileStorage) LoadSegment(id SegmentID) (*Segment, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	segments, err := fs.loadAllSegments()
	if err != nil {
		return nil, fmt.Errorf("failed to load segments: %w", err)
	}

	for _, seg := range segments {
		if seg.GetID() == id {
			return seg, nil
		}
	}

	return nil, fmt.Errorf("segment %s not found", id)
}

// ListSegments 列出所有Segment
func (fs *FileStorage) ListSegments() ([]*Segment, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	return fs.loadAllSegments()
}

// DeleteSegment 删除Segment元数据
func (fs *FileStorage) DeleteSegment(id SegmentID) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	segments, err := fs.loadAllSegments()
	if err != nil {
		return fmt.Errorf("failed to load segments: %w", err)
	}

	// 找到并删除
	found := false
	newSegments := make([]*Segment, 0, len(segments))
	for _, seg := range segments {
		if seg.GetID() != id {
			newSegments = append(newSegments, seg)
		} else {
			found = true
		}
	}

	if !found {
		return nil // 不存在视为成功
	}

	return fs.saveAllSegments(newSegments)
}

// loadAllSegments 加载所有Segments（内部方法，不加锁）
func (fs *FileStorage) loadAllSegments() ([]*Segment, error) {
	path := fs.segmentsFilePath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Segment{}, nil // 空列表
		}
		return nil, err
	}

	var segments []*Segment
	if err := json.Unmarshal(data, &segments); err != nil {
		return nil, fmt.Errorf("failed to unmarshal segments: %w", err)
	}

	return segments, nil
}

// saveAllSegments 保存所有Segments（内部方法，不加锁）
func (fs *FileStorage) saveAllSegments(segments []*Segment) error {
	data, err := json.MarshalIndent(segments, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal segments: %w", err)
	}

	path := fs.segmentsFilePath()
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write segments file: %w", err)
	}

	return nil
}
