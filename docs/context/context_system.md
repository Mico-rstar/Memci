# ContextSystem 设计文档

## 概述

ContextSystem 是上下文管理系统的核心，负责管理所有 Segment 和 Page，提供统一的页面操作能力。

## 核心职责

1. **Segment 管理**：管理多个 Segment 的生命周期
2. **Page 存储**：全局 Page 注册表，通过索引快速查找
3. **树结构维护**：维护 Page 之间的父子关系
4. **内部操作**：提供不包含权限检查的核心操作方法

## PageStorage 接口设计

ContextSystem 通过 **PageStorage 接口** 与底层存储解耦，支持可插拔的存储实现。

```go
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

func NewMemoryStorage() *MemoryStorage
func (ms *MemoryStorage) Save(page Page) error
func (ms *MemoryStorage) Load(pageIndex PageIndex) (Page, error)
func (ms *MemoryStorage) Delete(pageIndex PageIndex) error
func (ms *MemoryStorage) Exists(pageIndex PageIndex) bool
func (ms *MemoryStorage) List() ([]PageIndex, error)

// FileStorage 文件存储实现（示例）
type FileStorage struct {
	baseDir string
	mu      sync.RWMutex
}

func NewFileStorage(baseDir string) *FileStorage
func (fs *FileStorage) Save(page Page) error
func (fs *FileStorage) Load(pageIndex PageIndex) (Page, error)
func (fs *FileStorage) Delete(pageIndex PageIndex) error
func (fs *FileStorage) Exists(pageIndex PageIndex) bool
func (fs *FileStorage) List() ([]PageIndex, error)

// DatabaseStorage 数据库存储实现（示例）
type DatabaseStorage struct {
	db *sql.DB
}

func NewDatabaseStorage(db *sql.DB) *DatabaseStorage
func (ds *DatabaseStorage) Save(page Page) error
func (ds *DatabaseStorage) Load(pageIndex PageIndex) (Page, error)
func (ds *DatabaseStorage) Delete(pageIndex PageIndex) error
func (ds *DatabaseStorage) Exists(pageIndex PageIndex) bool
func (ds *DatabaseStorage) List() ([]PageIndex, error)
```

## ContextSystem 结构体定义

```go
// ContextSystem 上下文系统核心
type ContextSystem struct {
	// Segment 管理
	segments    []*Segment           // 按添加顺序存储，显示顺序
	segmentMap  map[SegmentID]*Segment  // 快速查找

	// Page 存储
	pages       map[PageIndex]Page    // 全局 Page 注册表（内存缓存）
	storage     PageStorage           // 持久化存储接口

	// 索引生成
	nextIndex   int                     // 用于生成新的 PageIndex

	// 元数据
	createdAt   time.Time
	updatedAt   time.Time

	// 并发控制
	mu          sync.RWMutex           // 读写锁
}

// NewContextSystem 创建新的上下文系统（使用内存存储）
func NewContextSystem() *ContextSystem {
	return &ContextSystem{
		segments:   make([]*Segment, 0),
		segmentMap: make(map[SegmentID]*Segment),
		pages:      make(map[PageIndex]Page),
		storage:    NewMemoryStorage(), // 默认使用内存存储
		nextIndex:  0,
		createdAt:  time.Now(),
		updatedAt:  time.Now(),
	}
}

// NewContextSystemWithStorage 创建指定存储的上下文系统
func NewContextSystemWithStorage(storage PageStorage) *ContextSystem {
	return &ContextSystem{
		segments:   make([]*Segment, 0),
		segmentMap: make(map[SegmentID]*Segment),
		pages:      make(map[PageIndex]Page),
		storage:    storage,
		nextIndex:  0,
		createdAt:  time.Now(),
		updatedAt:  time.Now(),
	}
}

// SetStorage 设置存储实现
func (cs *ContextSystem) SetStorage(storage PageStorage)

// GetStorage 获取当前存储实现
func (cs *ContextSystem) GetStorage() PageStorage
```

**所有权说明**：
- ContextSystem 通过**值传递**获得 Segment 的所有权
- `AddSegment` 接受 `Segment` 类型（非指针），系统内部存储副本
- 开发者在调用 `AddSegment(*segment)` 后，对原始变量的修改不影响系统内的副本
- 这是 Go 中实现所有权转移的标准方式（类似 Rust 的 Move 语义）

## 核心 API（内部方法，无权限检查）

### Segment 管理方法

```go
// AddSegment 添加Segment（按值传递，获取所有权）
func (cs *ContextSystem) AddSegment(segment Segment) error

// RemoveSegment 移除Segment
func (cs *ContextSystem) RemoveSegment(id SegmentID) error

// GetSegment 获取Segment（返回副本，防止外部修改）
func (cs *ContextSystem) GetSegment(id SegmentID) (Segment, error)

// ListSegments 列出所有Segment（按顺序，返回副本）
func (cs *ContextSystem) ListSegments() ([]Segment, error)

// GetSegmentByPageIndex 根据pageIndex查找所属Segment（返回副本）
func (cs *ContextSystem) GetSegmentByPageIndex(pageIndex PageIndex) (Segment, error)
```

**内部方法（返回指针，供系统内部使用）**：
```go
// getSegmentByPageIndexInternal 根据pageIndex查找所属Segment（内部方法）
func (cs *ContextSystem) getSegmentByPageIndexInternal(pageIndex PageIndex) (*Segment, error)

// UpdateSegment 更新Segment（通过ContextSystem控制）
func (cs *ContextSystem) UpdateSegment(id SegmentID, name, description string) error

// SetSegmentPermission 设置Segment权限（通过ContextSystem控制）
func (cs *ContextSystem) SetSegmentPermission(id SegmentID, permission SegmentPermission) error
```

**实现示例**：

```go
func (cs *ContextSystem) AddSegment(segment Segment) error {
	if _, exists := cs.segmentMap[segment.GetID()]; exists {
		return fmt.Errorf("segment %s already exists", segment.GetID())
	}

	// 设置索引
	if segment.GetRootIndex() != "" {
		if _, pageExists := cs.pages[segment.GetRootIndex()]; pageExists {
			return fmt.Errorf("root page %s not found", segment.GetRootIndex())
		}
	}

	// 存储副本（值传递已经发生，这里直接存储）
	cs.segments = append(cs.segments, &segment)
	cs.segmentMap[segment.GetID()] = &segment
	cs.updatedAt = time.Now()

	return nil
}

func (cs *ContextSystem) GetSegmentByPageIndex(pageIndex PageIndex) (Segment, error) {
	// 通过 pageIndex 前缀找到对应的 Segment
	for _, seg := range cs.segments {
		if seg == nil {
			continue
		}
		prefix := string(seg.GetID()) + "-"
		if strings.HasPrefix(string(pageIndex), prefix) {
			return *seg, nil  // 返回副本
		}
	}
	return Segment{}, fmt.Errorf("no segment found for page %s", pageIndex)
}

// 内部方法版本（返回指针）
func (cs *ContextSystem) getSegmentByPageIndexInternal(pageIndex PageIndex) (*Segment, error) {
	for _, seg := range cs.segments {
		if seg == nil {
			continue
		}
		prefix := string(seg.GetID()) + "-"
		if strings.HasPrefix(string(pageIndex), prefix) {
			return seg, nil  // 返回指针，供内部使用
		}
	}
	return nil, fmt.Errorf("no segment found for page %s", pageIndex)
}
```

**为什么返回值类型副本**：
- 防止外部直接修改系统内的 Segment
- 保证 ContextSystem 对状态的完全控制
- Segment 结构体较小，复制开销可接受
- 如果需要修改，应该通过 ContextSystem 提供的方法

### Page 存储方法

```go
// AddPage 添加Page到系统（自动持久化）
func (cs *ContextSystem) AddPage(page Page) error

// GetPage 获取Page（支持懒加载）
func (cs *ContextSystem) GetPage(pageIndex PageIndex) (Page, error)

// RemovePage 移除Page（自动删除持久化）
func (cs *ContextSystem) RemovePage(pageIndex PageIndex) error

// ListPages 列出所有Page（内存中）
func (cs *ContextSystem) ListPages() []Page

// LoadPageFromStorage 从存储加载Page
func (cs *ContextSystem) LoadPageFromStorage(pageIndex PageIndex) (Page, error)

// EvictPage 从内存缓存驱逐Page（保留存储）
func (cs *ContextSystem) EvictPage(pageIndex PageIndex) error

// GenerateIndex 生成新的PageIndex
func (cs *ContextSystem) GenerateIndex(segmentID SegmentID) PageIndex
```

**实现示例（包含自动持久化）**：

```go
func (cs *ContextSystem) AddPage(page Page) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	pageIndex := page.GetIndex()
	parentIndex := page.GetParent()

	// 验证 1: Page 索引唯一性（先检查内存，再检查存储）
	if _, exists := cs.pages[pageIndex]; exists {
		return fmt.Errorf("page %s already exists", pageIndex)
	}
	if cs.storage != nil && cs.storage.Exists(pageIndex) {
		return fmt.Errorf("page %s already exists in storage", pageIndex)
	}

	// 验证 2: 父节点完整性（除了 root page，所有 page 必须有父节点）
	if parentIndex == "" {
		// 没有 parent，必须是某个 Segment 的 root page
		isRoot := false
		for _, seg := range cs.segments {
			if seg.GetRootIndex() == pageIndex {
				isRoot = true
				break
			}
		}
		if !isRoot {
			return fmt.Errorf("page %s has no parent and is not a segment root", pageIndex)
		}
	} else {
		// 有 parent，必须验证 parent 存在且是 ContentsPage
		parent, exists := cs.pages[parentIndex]
		if !exists {
			// 尝试从存储加载
			if cs.storage != nil && cs.storage.Exists(parentIndex) {
				loadedPage, err := cs.storage.Load(parentIndex)
				if err != nil {
					return fmt.Errorf("failed to load parent page %s: %w", parentIndex, err)
				}
				parent = loadedPage
				cs.pages[parentIndex] = parent // 缓存到内存
			} else {
				return fmt.Errorf("parent page %s not found for page %s", parentIndex, pageIndex)
			}
		}
		if parentPage, ok := parent.(*ContentsPage); ok {
			parentPage.AddChild(pageIndex)
		} else {
			return fmt.Errorf("parent page %s is not a ContentsPage", parentIndex)
		}
	}

	// 验证通过，添加到内存
	cs.pages[pageIndex] = page
	cs.updatedAt = time.Now()

	// 自动持久化到存储
	if cs.storage != nil {
		if err := cs.storage.Save(page); err != nil {
			// 持久化失败，回滚内存操作
			delete(cs.pages, pageIndex)
			return fmt.Errorf("failed to save page %s to storage: %w", pageIndex, err)
		}
	}

	return nil
}

func (cs *ContextSystem) GenerateIndex(segmentID SegmentID) PageIndex {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.nextIndex++
	return PageIndex(fmt.Sprintf("%s-%d", segmentID, cs.nextIndex))
}

// GetPage 获取Page（支持懒加载）
func (cs *ContextSystem) GetPage(pageIndex PageIndex) (Page, error) {
	cs.mu.RLock()
	page, exists := cs.pages[pageIndex]
	cs.mu.RUnlock()

	if !exists {
		// 尝试从存储加载（懒加载）
		if cs.storage != nil && cs.storage.Exists(pageIndex) {
			loadedPage, err := cs.storage.Load(pageIndex)
			if err != nil {
				return nil, fmt.Errorf("failed to load page %s from storage: %w", pageIndex, err)
			}
			// 加载到内存缓存
			cs.mu.Lock()
			cs.pages[pageIndex] = loadedPage
			cs.mu.Unlock()
			return loadedPage, nil
		}
		return nil, fmt.Errorf("page %s not found", pageIndex)
	}

	return page, nil
}

// RemovePage 移除Page（自动删除持久化）
func (cs *ContextSystem) RemovePage(pageIndex PageIndex) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	page, exists := cs.pages[pageIndex]
	if !exists {
		// 尝试从存储删除
		if cs.storage != nil && cs.storage.Exists(pageIndex) {
			if err := cs.storage.Delete(pageIndex); err != nil {
				return fmt.Errorf("failed to delete page %s from storage: %w", pageIndex, err)
			}
			cs.updatedAt = time.Now()
			return nil
		}
		return fmt.Errorf("page %s not found", pageIndex)
	}

	// 递归删除子节点
	if contentsPage, ok := page.(*ContentsPage); ok {
		for _, childIndex := range contentsPage.GetChildren() {
			if err := cs.removePageInternal(childIndex); err != nil {
				return err
			}
		}
	}

	// 从父节点移除
	if page.GetParent() != "" {
		parent, err := cs.GetPage(page.GetParent())
		if err != nil {
			return err
		}
		if parentPage, ok := parent.(*ContentsPage); ok {
			parentPage.RemoveChild(pageIndex)
		}
	}

	// 从存储删除
	if cs.storage != nil && cs.storage.Exists(pageIndex) {
		if err := cs.storage.Delete(pageIndex); err != nil {
			return fmt.Errorf("failed to delete page %s from storage: %w", pageIndex, err)
		}
	}

	// 从内存删除
	delete(cs.pages, pageIndex)
	cs.updatedAt = time.Now()

	return nil
}

// LoadPageFromStorage 从存储加载Page（强制加载）
func (cs *ContextSystem) LoadPageFromStorage(pageIndex PageIndex) (Page, error) {
	if cs.storage == nil {
		return nil, fmt.Errorf("no storage configured")
	}

	if !cs.storage.Exists(pageIndex) {
		return nil, fmt.Errorf("page %s not found in storage", pageIndex)
	}

	page, err := cs.storage.Load(pageIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to load page %s: %w", pageIndex, err)
	}

	// 更新内存缓存
	cs.mu.Lock()
	cs.pages[pageIndex] = page
	cs.mu.Unlock()

	return page, nil
}

// EvictPage 从内存缓存驱逐Page（保留存储）
func (cs *ContextSystem) EvictPage(pageIndex PageIndex) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if _, exists := cs.pages[pageIndex]; !exists {
		return fmt.Errorf("page %s not found in memory", pageIndex)
	}

	delete(cs.pages, pageIndex)
	return nil
}
```

## 存储生命周期管理

ContextSystem 采用 **双层存储架构**：内存缓存 + 持久化存储。

### 架构图

```
ContextSystem
    │
    ├── 内存缓存 (map[PageIndex]Page)
    │   ├── 快速访问
    │   ├── 读写操作
    │   └── 驱逐管理
    │
    └── PageStorage 接口
        ├── MemoryStorage（默认）
        ├── FileStorage（文件系统）
        └── DatabaseStorage（数据库）
```

### 自动持久化机制

**写操作**（AddPage、UpdatePage、RemovePage）：
```go
// 1. 更新内存缓存
cs.pages[pageIndex] = page

// 2. 自动持久化到存储
if cs.storage != nil {
    if err := cs.storage.Save(page); err != nil {
        // 持久化失败，回滚内存操作
        delete(cs.pages, pageIndex)
        return err
    }
}
```

**读操作**（GetPage）：
```go
// 1. 先检查内存缓存
if page, exists := cs.pages[pageIndex]; exists {
    return page, nil
}

// 2. 内存未命中，从存储懒加载
if cs.storage != nil && cs.storage.Exists(pageIndex) {
    page, err := cs.storage.Load(pageIndex)
    if err != nil {
        return nil, err
    }
    // 加载到内存缓存
    cs.pages[pageIndex] = page
    return page, nil
}

return nil, ErrNotFound
```

### 存储实现示例

**1. MemoryStorage（内存存储，默认）**

```go
type MemoryStorage struct {
	pages map[PageIndex]Page
	mu    sync.RWMutex
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		pages: make(map[PageIndex]Page),
	}
}

func (ms *MemoryStorage) Save(page Page) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.pages[page.GetIndex()] = page
	return nil
}

func (ms *MemoryStorage) Load(pageIndex PageIndex) (Page, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	page, exists := ms.pages[pageIndex]
	if !exists {
		return nil, fmt.Errorf("page %s not found", pageIndex)
	}
	return page, nil
}

func (ms *MemoryStorage) Delete(pageIndex PageIndex) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	delete(ms.pages, pageIndex)
	return nil
}

func (ms *MemoryStorage) Exists(pageIndex PageIndex) bool {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	_, exists := ms.pages[pageIndex]
	return exists
}

func (ms *MemoryStorage) List() ([]PageIndex, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	indices := make([]PageIndex, 0, len(ms.pages))
	for index := range ms.pages {
		indices = append(indices, index)
	}
	return indices, nil
}
```

**2. FileStorage（文件系统存储）**

```go
type FileStorage struct {
	baseDir string
	mu      sync.RWMutex
}

func NewFileStorage(baseDir string) *FileStorage {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Fatalf("Failed to create base dir: %v", err)
	}
	return &FileStorage{baseDir: baseDir}
}

func (fs *FileStorage) pagePath(pageIndex PageIndex) string {
	return filepath.Join(fs.baseDir, string(pageIndex)+".json")
}

func (fs *FileStorage) Save(page Page) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := page.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal page: %w", err)
	}

	path := fs.pagePath(page.GetIndex())
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write page: %w", err)
	}

	return nil
}

func (fs *FileStorage) Load(pageIndex PageIndex) (Page, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	path := fs.pagePath(pageIndex)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read page: %w", err)
	}

	// 根据类型创建相应的Page
	// 这里需要实现Page接口的反序列化
	// ...
}

func (fs *FileStorage) Delete(pageIndex PageIndex) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := fs.pagePath(pageIndex)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete page: %w", err)
	}

	return nil
}

func (fs *FileStorage) Exists(pageIndex PageIndex) bool {
	path := fs.pagePath(pageIndex)
	_, err := os.Stat(path)
	return err == nil
}

func (fs *FileStorage) List() ([]PageIndex, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	entries, err := os.ReadDir(fs.baseDir)
	if err != nil {
		return nil, err
	}

	indices := make([]PageIndex, 0)
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".json") {
			index := PageIndex(strings.TrimSuffix(entry.Name(), ".json"))
			indices = append(indices, index)
		}
	}

	return indices, nil
}
```

**3. DatabaseStorage（数据库存储）**

```go
type DatabaseStorage struct {
	db *sql.DB
}

func NewDatabaseStorage(db *sql.DB) *DatabaseStorage {
	// 创建表
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS pages (
			index TEXT PRIMARY KEY,
			data BLOB NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create pages table: %v", err)
	}

	return &DatabaseStorage{db: db}
}

func (ds *DatabaseStorage) Save(page Page) error {
	data, err := page.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal page: %w", err)
	}

	_, err = ds.db.Exec(`
		INSERT OR REPLACE INTO pages (index, data, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`, page.GetIndex(), data)

	if err != nil {
		return fmt.Errorf("failed to save page: %w", err)
	}

	return nil
}

func (ds *DatabaseStorage) Load(pageIndex PageIndex) (Page, error) {
	var data []byte
	err := ds.db.QueryRow(`
		SELECT data FROM pages WHERE index = ?
	`, pageIndex).Scan(&data)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("page %s not found", pageIndex)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load page: %w", err)
	}

	// 反序列化...
	// ...

	return page, nil
}

func (ds *DatabaseStorage) Delete(pageIndex PageIndex) error {
	_, err := ds.db.Exec(`DELETE FROM pages WHERE index = ?`, pageIndex)
	if err != nil {
		return fmt.Errorf("failed to delete page: %w", err)
	}
	return nil
}

func (ds *DatabaseStorage) Exists(pageIndex PageIndex) bool {
	var count int
	err := ds.db.QueryRow(`
		SELECT COUNT(*) FROM pages WHERE index = ?
	`, pageIndex).Scan(&count)
	return err == nil && count > 0
}

func (ds *DatabaseStorage) List() ([]PageIndex, error) {
	rows, err := ds.db.Query(`SELECT index FROM pages ORDER BY index`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indices []PageIndex
	for rows.Next() {
		var index string
		if err := rows.Scan(&index); err != nil {
			return nil, err
		}
		indices = append(indices, PageIndex(index))
	}

	return indices, nil
}
```

### 存储切换

**运行时切换存储**：

```go
// 1. 创建系统（默认内存存储）
contextSystem := NewContextSystem()

// 2. 执行一些操作
// ...

// 3. 切换到文件存储
fileStorage := NewFileStorage("./data/pages")
contextSystem.SetStorage(fileStorage)

// 4. 现在所有写操作都会持久化到文件
contextSystem.AddPage(page)

// 5. 也可以切换回内存存储
contextSystem.SetStorage(NewMemoryStorage())
```

**批量数据迁移**：

```go
// 从内存存储迁移到文件存储
func MigrateToDisk(contextSystem *ContextSystem, targetDir string) error {
	fileStorage := NewFileStorage(targetDir)

	// 遍历所有Page
	for _, page := range contextSystem.ListPages() {
		if err := fileStorage.Save(page); err != nil {
			return fmt.Errorf("failed to migrate page %s: %w", page.GetIndex(), err)
		}
	}

	// 切换存储
	contextSystem.SetStorage(fileStorage)
	return nil
}
```

### 内存管理策略

**驱逐策略（LRU示例）**：

```go
type LRUCache struct {
	maxSize int
	items   map[PageIndex]*list.Element
	lru     *list.List
}

type cacheItem struct {
	index PageIndex
	page  Page
}

func (cs *ContextSystem) EvictLRU(maxSize int) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if len(cs.pages) <= maxSize {
		return nil
	}

	// 简单驱逐策略：移除最旧的Page（实际应该使用LRU）
	// 这里简化实现
	toEvict := len(cs.pages) - maxSize
	count := 0

	for index := range cs.pages {
		if count >= toEvict {
			break
		}
		// 不要驱逐正在使用的Page
		delete(cs.pages, index)
		count++
	}

	return nil
}
```

## Page 操作的内部方法（无权限检查）

这些方法是核心业务逻辑，不进行权限检查，由 AgentContext 代理调用。

### 状态变更方法

```go
// updatePageInternal 更新Page（内部方法）
func (cs *ContextSystem) updatePageInternal(pageIndex PageIndex, name, description string) error

// expandDetailsInternal 展开详情（内部方法）
func (cs *ContextSystem) expandDetailsInternal(pageIndex PageIndex) error

// hideDetailsInternal 隐藏详情（内部方法）
func (cs *ContextSystem) hideDetailsInternal(pageIndex PageIndex) error
```

**实现示例**：

```go
func (cs *ContextSystem) updatePageInternal(pageIndex PageIndex, name, description string) error {
	page, err := cs.GetPage(pageIndex)
	if err != nil {
		return err
	}

	if name != "" {
		if err := page.SetName(name); err != nil {
			return err
		}
	}

	if description != "" {
		if err := page.SetDescription(description); err != nil {
			return err
		}
	}

	return nil
}
```

### 结构操作方法

```go
// movePageInternal 移动Page（内部方法）
func (cs *ContextSystem) movePageInternal(source, target PageIndex) error

// removePageInternal 删除Page（内部方法）
func (cs *ContextSystem) removePageInternal(pageIndex PageIndex) error

// createDetailPageInternal 创建DetailPage（内部方法）
func (cs *ContextSystem) createDetailPageInternal(name, description, detail string, parentIndex PageIndex) (PageIndex, error)

// createContentsPageInternal 创建ContentsPage（内部方法）
func (cs *ContextSystem) createContentsPageInternal(name, description string, children ...PageIndex) (PageIndex, error)
```

**实现示例**：

```go
func (cs *ContextSystem) movePageInternal(source, target PageIndex) error {
	// 1. 获取源Page和目标父Page
	sourcePage, err := cs.GetPage(source)
	if err != nil {
		return err
	}

	targetParent, err := cs.GetPage(target)
	if err != nil {
		return err
	}

	// 2. 检查目标必须是ContentsPage
	if targetPage, ok := targetParent.(*ContentsPage); !ok {
		return fmt.Errorf("target %s is not a ContentsPage", target)
	}

	// 3. 从原父节点移除
	if sourcePage.GetParent() != "" {
		oldParent, err := cs.GetPage(sourcePage.GetParent())
		if err != nil {
			return err
		}
		if oldParentPage, ok := oldParent.(*ContentsPage); ok {
			oldParentPage.RemoveChild(source)
		}
	}

	// 4. 添加到新父节点
	if err := targetPage.AddChild(source); err != nil {
		return err
	}

	// 5. 更新Page的父引用
	sourcePage.SetParent(target)

	return nil
}

func (cs *ContextSystem) removePageInternal(pageIndex PageIndex) error {
	page, err := cs.GetPage(pageIndex)
	if err != nil {
		return err
	}

	// 递归删除子节点
	if contentsPage, ok := page.(*ContentsPage); ok {
		for _, childIndex := range contentsPage.GetChildren() {
			if err := cs.removePageInternal(childIndex); err != nil {
				return err
			}
		}
	}

	// 从父节点移除
	if page.GetParent() != "" {
		parent, err := cs.GetPage(page.GetParent())
		if err != nil {
			return err
		}
		if parentPage, ok := parent.(*ContentsPage); ok {
			parentPage.RemoveChild(pageIndex)
		}
	}

	// 从注册表删除
	delete(cs.pages, pageIndex)

	return nil
}
```

### 查询方法

```go
// GetChildren 获取子Page列表
func (cs *ContextSystem) GetChildren(pageIndex PageIndex) ([]Page, error)

// GetParent 获取父Page
func (cs *ContextSystem) GetParent(pageIndex PageIndex) (Page, error)

// GetAncestors 获取祖先Page列表
func (cs *ContextSystem) GetAncestors(pageIndex PageIndex) ([]Page, error)

// FindPage 查找Page（支持名称、描述搜索）
func (cs *ContextSystem) FindPage(query string) []Page
```

## 序列化与持久化

```go
// Marshal 序列化ContextSystem
func (cs *ContextSystem) Marshal() ([]byte, error)

// Unmarshal 反序列化ContextSystem
func (cs *ContextSystem) Unmarshal(data []byte) error

// SaveToFile 保存到文件
func (cs *ContextSystem) SaveToFile(path string) error

// LoadFromFile 从文件加载
func (cs *ContextSystem) LoadFromFile(path string) error
```

**JSON 序列化格式**：

```json
{
  "segments": [
    {
      "id": "sys",
      "name": "System",
      "rootIndex": "sys-0",
      "permission": 0,
      "maxCapacity": 0
    },
    {
      "id": "usr",
      "name": "User",
      "rootIndex": "usr-0",
      "permission": 1,
      "maxCapacity": 4000
    }
  ],
  "pages": {
    "sys-0": { "type": "ContentsPage", "name": "System", ... },
    "sys-1": { "type": "DetailPage", "name": "System Prompt", ... },
    "usr-0": { "type": "ContentsPage", "name": "User", ... },
    "usr-1": { "type": "DetailPage", "name": "Chat 1", ... }
  },
  "nextIndex": 5,
  "createdAt": "2025-02-06T10:00:00Z",
  "updatedAt": "2025-02-06T10:00:00Z"
}
```

## 与其他组件的关系

### ContextSystem vs Segment

| 特性 | ContextSystem | Segment |
|------|--------------|---------|
| 抽象层级 | 系统容器 | 逻辑分组 |
| 管理对象 | Segment 和 Page | Page 索引 |
| 职责 | 统一管理、索引分配 | 权限控制、容量限制 |

### ContextSystem vs Page

| 特性 | ContextSystem | Page |
|------|--------------|------|
| 管理方式 | 通过索引引用 | 直接持有内容 |
| 可见性 | Agent 不可见 | Agent 可见 |
| 生命周期 | 系统级 | Page 级别 |

## 设计要点

### 1. 存储解耦（接口模式）

ContextSystem 通过 **PageStorage 接口** 与底层存储解耦：

```go
// 优势
// - 灵活性：可运行时切换存储实现
// - 可测试性：可注入 Mock Storage 进行单元测试
// - 可扩展性：轻松添加新的存储后端
// - 性能优化：支持内存缓存 + 持久化存储双层架构
```

**使用场景**：

| 存储类型 | 适用场景 | 优势 |
|---------|---------|------|
| MemoryStorage | 单元测试、短期运行 | 零延迟 |
| FileStorage | 本地持久化、调试 | 简单可靠 |
| DatabaseStorage | 生产环境、分布式 | 事务支持 |
| S3Storage | 云存储、大规模扩展 | 高可用 |

### 2. 索引唯一性

```go
// 通过前缀确保不同 Segment 的索引不冲突
sysIndex := contextSystem.GenerateIndex("sys")  // "sys-1"
usrIndex := contextSystem.GenerateIndex("usr")  // "usr-1"
```

### 2. 父子关系维护与完整性约束

**核心约束**：除了 root page 外，所有 Page 都必须有父节点。

```go
// 添加Page时验证父节点完整性
func (cs *ContextSystem) AddPage(page Page) error {
    parentIndex := page.GetParent()

    if parentIndex == "" {
        // 没有 parent，必须是某个 Segment 的 root
        isRoot := false
        for _, seg := range cs.segments {
            if seg.GetRootIndex() == page.GetIndex() {
                isRoot = true
                break
            }
        }
        if !isRoot {
            return fmt.Errorf("page must have parent unless it's a segment root")
        }
    } else {
        // 有 parent，必须验证 parent 存在
        parent, exists := cs.pages[parentIndex]
        if !exists {
            return fmt.Errorf("parent page %s not found", parentIndex)
        }
        if parentPage, ok := parent.(*ContentsPage); ok {
            parentPage.AddChild(page.GetIndex())
        } else {
            return fmt.Errorf("parent must be ContentsPage")
        }
    }

    cs.pages[page.GetIndex()] = page
    return nil
}
```

**完整性保证**：
- ✅ 防止孤儿节点：parent 必须存在
- ✅ 防止悬空引用：parent 必须是 ContentsPage
- ✅ root 唯一性：只有 Segment 的 root 可以没有 parent

### 3. 线程安全

ContextSystem **不保证**线程安全，如果需要并发访问，应该在外层加锁：

```go
type ContextSystem struct {
    mu   sync.RWMutex
    // ... 其他字段
}

func (cs *ContextSystem) AddPage(page Page) error {
    cs.mu.Lock()
    defer cs.mu.Unlock()
    // ... 操作逻辑
}
```

## 注意事项

1. **职责单一**：ContextSystem 只负责状态管理，不处理权限和业务逻辑
2. **内部方法**：所有 `Internal` 结尾的方法都是内部实现，不进行权限检查
3. **代理调用**：外部应该通过 AgentContext 调用，而不是直接调用 ContextSystem
4. **索引一致性**：确保 Page 索引与 Segment ID 的前缀匹配
5. **内存管理**：大量 Page 时考虑使用 LRU 或其他淘汰策略
6. **存储事务**：持久化失败会自动回滚内存操作，保证数据一致性
7. **懒加载**：GetPage 支持从存储懒加载，未访问的 Page 不会占用内存
8. **存储切换**：可以在运行时切换存储实现，但要注意数据迁移

## 典型使用流程

### 使用默认存储（内存）

```go
// 1. 创建系统（默认内存存储）
contextSystem := NewContextSystem()

// 2. 创建并添加Segment
sysSeg := NewSegment("sys", "System", "System context", SystemSegment)
sysSeg.SetPermission(ReadOnly)
contextSystem.AddSegment(*sysSeg)

// 3. 创建并添加Page
sysRoot, _ := NewContentsPage("System", "System prompts", "")
sysRoot.SetIndex(PageIndex("sys-0"))
sysSeg.SetRootIndex(sysRoot.GetIndex())
contextSystem.AddPage(sysRoot)
```

### 使用文件存储

```go
// 1. 创建文件存储
fileStorage := NewFileStorage("./data/pages")

// 2. 创建系统（指定存储）
contextSystem := NewContextSystemWithStorage(fileStorage)

// 3. 正常操作，会自动持久化
sysSeg := NewSegment("sys", "System", "System context", SystemSegment)
contextSystem.AddSegment(*sysSeg)

sysRoot := NewContentsPage("System", "System prompts", "")
sysRoot.SetIndex(PageIndex("sys-0"))
contextSystem.AddPage(sysRoot) // 自动保存到文件
```

### 运行时切换存储

```go
// 1. 启动时使用内存存储（快速启动）
contextSystem := NewContextSystem()

// 2. 执行初始化操作
// ...

// 3. 切换到持久化存储
fileStorage := NewFileStorage("./data/pages")
contextSystem.SetStorage(fileStorage)

// 4. 后续操作会自动持久化
```

```go
// 1. 创建系统
contextSystem := NewContextSystem()

// 2. 创建并添加Segment
sysSeg := NewSegment("sys", "System", "System context", SystemSegment)
sysSeg.SetPermission(ReadOnly)
contextSystem.AddSegment(sysSeg)

// 3. 创建并添加Page
sysRoot, _ := NewContentsPage("System", "System prompts", "")
sysRoot.SetIndex(PageIndex("sys-0"))
sysSeg.SetRootIndex(sysRoot.GetIndex())
contextSystem.AddPage(sysRoot)

// 4. 创建子Page
sysPrompt, _ := NewDetailPage("System Prompt", "Main prompt", "You are...", "sys-0")
sysPrompt.SetIndex(PageIndex("sys-1"))
contextSystem.AddPage(sysPrompt)

// 5. 创建代理供Agent使用
agentCtx := NewAgentContext(contextSystem)
```
