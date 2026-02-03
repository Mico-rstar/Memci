package context

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// PageStorage Page 存储接口
type PageStorage interface {
	Save(page *Page, index PageIndex) error
	Load(index PageIndex) (*Page, error)
	Delete(index PageIndex) error
	Exists(index PageIndex) bool
}

// PageCodec Page 序列化接口
type PageCodec interface {
	Encode(page *Page) ([]byte, error)
	Decode(data []byte) (*Page, error)
}

// JSONPageCodec JSON 编解码器
type JSONPageCodec struct{}

// Encode JSON 编码
func (jc *JSONPageCodec) Encode(page *Page) ([]byte, error) {
	return json.Marshal(page)
}

// Decode JSON 解码
func (jc *JSONPageCodec) Decode(data []byte) (*Page, error) {
	var page Page
	err := json.Unmarshal(data, &page)
	if err != nil {
		return nil, fmt.Errorf("failed to decode page: %w", err)
	}
	return &page, nil
}

// GzipPageCodec Gzip 压缩的 JSON 编解码器
type GzipPageCodec struct {
	inner PageCodec
}

// NewGzipPageCodec 创建 Gzip 压缩编解码器
func NewGzipPageCodec() *GzipPageCodec {
	return &GzipPageCodec{
		inner: &JSONPageCodec{},
	}
}

// Encode 使用 Gzip 压缩后编码
func (gc *GzipPageCodec) Encode(page *Page) ([]byte, error) {
	// 先使用内部编码器编码
	data, err := gc.inner.Encode(page)
	if err != nil {
		return nil, err
	}

	// 压缩数据
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)

	_, err = writer.Write(data)
	if err != nil {
		writer.Close()
		return nil, fmt.Errorf("failed to compress data: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return compressed.Bytes(), nil
}

// Decode 解压后解码
func (gc *GzipPageCodec) Decode(data []byte) (*Page, error) {
	// 解压数据
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress data: %w", err)
	}

	// 使用内部解码器解码
	return gc.inner.Decode(decompressed)
}

// FilePageStorage 文件系统存储实现
type FilePageStorage struct {
	mu         sync.RWMutex
	baseDir    string
	codec      PageCodec
	fileExt    string
}

// NewFilePageStorage 创建文件存储
func NewFilePageStorage(baseDir string, useGzip bool) (*FilePageStorage, error) {
	// 确保基础目录存在
	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	var codec PageCodec = &JSONPageCodec{}
	fileExt := ".json"

	if useGzip {
		codec = NewGzipPageCodec()
		fileExt = ".json.gz"
	}

	return &FilePageStorage{
		baseDir: baseDir,
		codec:   codec,
		fileExt: fileExt,
	}, nil
}

// getFilePath 获取 Page 的文件路径
func (fs *FilePageStorage) getFilePath(index PageIndex) string {
	filename := fmt.Sprintf("page-%03d%s", index, fs.fileExt)
	return filepath.Join(fs.baseDir, filename)
}

// Save 保存 Page 到文件
func (fs *FilePageStorage) Save(page *Page, index PageIndex) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// 编码 Page
	data, err := fs.codec.Encode(page)
	if err != nil {
		return fmt.Errorf("failed to encode page: %w", err)
	}

	// 写入文件
	filePath := fs.getFilePath(index)
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write page file: %w", err)
	}

	return nil
}

// Load 从文件加载 Page
func (fs *FilePageStorage) Load(index PageIndex) (*Page, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	filePath := fs.getFilePath(index)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("page %d not found", index)
	}

	// 读取文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read page file: %w", err)
	}

	// 解码 Page
	page, err := fs.codec.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode page: %w", err)
	}

	return page, nil
}

// Delete 删除 Page 文件
func (fs *FilePageStorage) Delete(index PageIndex) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	filePath := fs.getFilePath(index)

	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete page file: %w", err)
	}

	return nil
}

// Exists 检查 Page 是否存在
func (fs *FilePageStorage) Exists(index PageIndex) bool {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	filePath := fs.getFilePath(index)
	_, err := os.Stat(filePath)
	return err == nil
}

// ListAll 列出所有已保存的 Page 索引
func (fs *FilePageStorage) ListAll() ([]PageIndex, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	entries, err := os.ReadDir(fs.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}

	indices := make([]PageIndex, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 解析文件名获取索引
		var index PageIndex
		_, err := fmt.Sscanf(entry.Name(), "page-%d", &index)
		if err == nil {
			indices = append(indices, index)
		}
	}

	return indices, nil
}

// Clear 清空所有存储的 Page
func (fs *FilePageStorage) Clear() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	entries, err := os.ReadDir(fs.baseDir)
	if err != nil {
		return fmt.Errorf("failed to read storage directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(fs.baseDir, entry.Name())
		err := os.Remove(filePath)
		if err != nil {
			return fmt.Errorf("failed to delete %s: %w", filePath, err)
		}
	}

	return nil
}

// MemoryPageStorage 内存存储实现（用于测试）
type MemoryPageStorage struct {
	mu    sync.RWMutex
	pages map[PageIndex]*Page
}

// NewMemoryPageStorage 创建内存存储
func NewMemoryPageStorage() *MemoryPageStorage {
	return &MemoryPageStorage{
		pages: make(map[PageIndex]*Page),
	}
}

// Save 保存 Page 到内存
func (ms *MemoryPageStorage) Save(page *Page, index PageIndex) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// 深拷贝 Page
	entries := make([]*Entry, len(page.Entries))
	copy(entries, page.Entries)

	pageCopy := &Page{
		Entries:      entries,
		Name:         page.Name,
		MaxToken:     page.MaxToken,
		Description:  page.Description,
		CompactModel: page.CompactModel,
	}

	ms.pages[index] = pageCopy
	return nil
}

// Load 从内存加载 Page
func (ms *MemoryPageStorage) Load(index PageIndex) (*Page, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	page, ok := ms.pages[index]
	if !ok {
		return nil, fmt.Errorf("page %d not found", index)
	}

	// 返回深拷贝
	entries := make([]*Entry, len(page.Entries))
	copy(entries, page.Entries)

	pageCopy := &Page{
		Entries:      entries,
		Name:         page.Name,
		MaxToken:     page.MaxToken,
		Description:  page.Description,
		CompactModel: page.CompactModel,
	}

	return pageCopy, nil
}

// Delete 从内存删除 Page
func (ms *MemoryPageStorage) Delete(index PageIndex) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	delete(ms.pages, index)
	return nil
}

// Exists 检查 Page 是否存在
func (ms *MemoryPageStorage) Exists(index PageIndex) bool {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	_, ok := ms.pages[index]
	return ok
}

// ListAll 列出所有已保存的 Page 索引
func (ms *MemoryPageStorage) ListAll() ([]PageIndex, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	indices := make([]PageIndex, 0, len(ms.pages))
	for index := range ms.pages {
		indices = append(indices, index)
	}

	return indices, nil
}

// Clear 清空所有存储的 Page
func (ms *MemoryPageStorage) Clear() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.pages = make(map[PageIndex]*Page)
	return nil
}
