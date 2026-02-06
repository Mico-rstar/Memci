package context

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ContextSystem 上下文系统核心
type ContextSystem struct {
	// Segment 管理
	segments   []*Segment         // 按添加顺序存储，显示顺序
	segmentMap map[SegmentID]*Segment // 快速查找

	// Page 存储
	pages   map[PageIndex]Page // 全局 Page 注册表（内存缓存）
	storage PageStorage        // 持久化存储接口

	// 索引生成
	nextIndex int // 用于生成新的 PageIndex

	// 元数据
	createdAt time.Time
	updatedAt time.Time

	// 并发控制
	mu sync.RWMutex // 读写锁
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
func (cs *ContextSystem) SetStorage(storage PageStorage) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.storage = storage
	cs.updatedAt = time.Now()
}

// GetStorage 获取当前存储实现
func (cs *ContextSystem) GetStorage() PageStorage {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.storage
}

// ============ Segment 管理方法 ============

// AddSegment 添加Segment（按值传递，获取所有权）
func (cs *ContextSystem) AddSegment(segment Segment) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// 检查ID唯一性
	if _, exists := cs.segmentMap[segment.GetID()]; exists {
		return fmt.Errorf("segment %s already exists", segment.GetID())
	}

	// 如果有rootIndex，验证root page存在
	if segment.GetRootIndex() != "" {
		if _, pageExists := cs.pages[segment.GetRootIndex()]; !pageExists {
			if cs.storage != nil && !cs.storage.Exists(segment.GetRootIndex()) {
				return fmt.Errorf("root page %s not found", segment.GetRootIndex())
			}
		}
	}

	// 创建副本并存储指针
	seg := &Segment{}
	*seg = segment
	cs.segments = append(cs.segments, seg)
	cs.segmentMap[segment.GetID()] = seg
	cs.updatedAt = time.Now()

	return nil
}

// RemoveSegment 移除Segment
func (cs *ContextSystem) RemoveSegment(id SegmentID) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	segment, exists := cs.segmentMap[id]
	if !exists {
		return fmt.Errorf("segment %s not found", id)
	}

	// 从切片中移除
	for i, seg := range cs.segments {
		if seg.GetID() == id {
			cs.segments = append(cs.segments[:i], cs.segments[i+1:]...)
			break
		}
	}

	// 从map中移除
	delete(cs.segmentMap, id)
	cs.updatedAt = time.Now()

	// 注意：不删除Segment下的Page，由外部管理
	_ = segment // 避免未使用警告
	return nil
}

// SetSegmentRootIndex 设置Segment的rootIndex（内部方法）
func (cs *ContextSystem) SetSegmentRootIndex(id SegmentID, rootIndex PageIndex) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	segment, exists := cs.segmentMap[id]
	if !exists {
		return fmt.Errorf("segment %s not found", id)
	}

	segment.rootIndex = rootIndex
	cs.updatedAt = time.Now()
	return nil
}

// GetSegment 获取Segment（返回副本，防止外部修改）
func (cs *ContextSystem) GetSegment(id SegmentID) (Segment, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	segment, exists := cs.segmentMap[id]
	if !exists {
		return Segment{}, fmt.Errorf("segment %s not found", id)
	}

	return *segment, nil // 返回副本
}

// ListSegments 列出所有Segment（按顺序，返回副本）
func (cs *ContextSystem) ListSegments() ([]Segment, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	result := make([]Segment, 0, len(cs.segments))
	for _, seg := range cs.segments {
		if seg != nil {
			result = append(result, *seg) // 返回副本
		}
	}
	return result, nil
}

// GetSegmentByPageIndex 根据pageIndex查找所属Segment（返回副本）
func (cs *ContextSystem) GetSegmentByPageIndex(pageIndex PageIndex) (Segment, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	for _, seg := range cs.segments {
		if seg == nil {
			continue
		}
		prefix := string(seg.GetID()) + "-"
		if strings.HasPrefix(string(pageIndex), prefix) {
			return *seg, nil // 返回副本
		}
	}
	return Segment{}, fmt.Errorf("no segment found for page %s", pageIndex)
}

// getSegmentByPageIndexInternal 根据pageIndex查找所属Segment（内部方法，返回指针）
func (cs *ContextSystem) getSegmentByPageIndexInternal(pageIndex PageIndex) (*Segment, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	for _, seg := range cs.segments {
		if seg == nil {
			continue
		}
		prefix := string(seg.GetID()) + "-"
		if strings.HasPrefix(string(pageIndex), prefix) {
			return seg, nil // 返回指针，供内部使用
		}
	}
	return nil, fmt.Errorf("no segment found for page %s", pageIndex)
}

// getSegmentInternal 根据ID获取Segment（内部方法，返回指针）
func (cs *ContextSystem) getSegmentInternal(id SegmentID) (*Segment, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	segment, exists := cs.segmentMap[id]
	if !exists {
		return nil, fmt.Errorf("segment %s not found", id)
	}
	return segment, nil
}

// ============ Page 存储方法 ============

// AddPage 添加Page到系统（自动持久化）
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

// GetPage 获取Page（支持懒加载）
func (cs *ContextSystem) GetPage(pageIndex PageIndex) (Page, error) {
	cs.mu.RLock()
	page, exists := cs.pages[pageIndex]
	cs.mu.RUnlock()

	if exists {
		return page, nil
	}

	// Page不在内存中，尝试从存储加载
	if cs.storage != nil && cs.storage.Exists(pageIndex) {
		loadedPage, err := cs.storage.Load(pageIndex)
		if err != nil {
			return nil, fmt.Errorf("failed to load page %s from storage: %w", pageIndex, err)
		}
		// 加载到内存缓存
		cs.mu.Lock()
		// 再次检查，防止在加载过程中被其他goroutine添加
		if _, exists := cs.pages[pageIndex]; !exists {
			cs.pages[pageIndex] = loadedPage
		}
		cs.mu.Unlock()
		return loadedPage, nil
	}

	return nil, fmt.Errorf("page %s not found", pageIndex)
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
		parent, exists := cs.pages[page.GetParent()]
		if !exists {
			return fmt.Errorf("parent page %s not found", page.GetParent())
		}
		if parentPage, ok := parent.(*ContentsPage); ok {
			parentPage.RemoveChild(pageIndex)
		} else {
			return fmt.Errorf("parent page %s is not a ContentsPage", page.GetParent())
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

// removePageInternal 删除Page的内部实现（递归）
func (cs *ContextSystem) removePageInternal(pageIndex PageIndex) error {
	page, exists := cs.pages[pageIndex]
	if !exists {
		return nil // 不存在，直接返回
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
		if parent, exists := cs.pages[page.GetParent()]; exists {
			if parentPage, ok := parent.(*ContentsPage); ok {
				parentPage.RemoveChild(pageIndex)
			}
		}
	}

	// 从存储删除
	if cs.storage != nil && cs.storage.Exists(pageIndex) {
		cs.storage.Delete(pageIndex)
	}

	// 从内存删除
	delete(cs.pages, pageIndex)
	return nil
}

// ListPages 列出所有Page（内存中）
func (cs *ContextSystem) ListPages() []Page {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	pages := make([]Page, 0, len(cs.pages))
	for _, page := range cs.pages {
		pages = append(pages, page)
	}
	return pages
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

// GenerateIndex 生成新的PageIndex
func (cs *ContextSystem) GenerateIndex(segmentID SegmentID) PageIndex {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.nextIndex++
	return PageIndex(fmt.Sprintf("%s-%d", segmentID, cs.nextIndex))
}

// ============ Page 操作的内部方法（无权限检查） ============

// updatePageInternal 更新Page（内部方法）
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

	// 持久化更新
	if cs.storage != nil {
		cs.storage.Save(page)
	}

	return nil
}

// expandDetailsInternal 展开详情（内部方法）
func (cs *ContextSystem) expandDetailsInternal(pageIndex PageIndex) error {
	page, err := cs.GetPage(pageIndex)
	if err != nil {
		return err
	}

	page.SetVisibility(Expanded)

	// 持久化更新
	if cs.storage != nil {
		cs.storage.Save(page)
	}

	return nil
}

// hideDetailsInternal 隐藏详情（内部方法）
func (cs *ContextSystem) hideDetailsInternal(pageIndex PageIndex) error {
	page, err := cs.GetPage(pageIndex)
	if err != nil {
		return err
	}

	page.SetVisibility(Hidden)

	// 持久化更新
	if cs.storage != nil {
		cs.storage.Save(page)
	}

	return nil
}

// movePageInternal 移动Page（内部方法）
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
	targetPage, ok := targetParent.(*ContentsPage)
	if !ok {
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

	// 持久化更新
	if cs.storage != nil {
		cs.storage.Save(sourcePage)
		cs.storage.Save(targetParent)
	}

	return nil
}

// createDetailPageInternal 创建DetailPage（内部方法）
func (cs *ContextSystem) createDetailPageInternal(name, description, detail string, parentIndex PageIndex) (PageIndex, error) {
	// 1. 获取父Page所属Segment（使用内部方法）
	segment, err := cs.getSegmentByPageIndexInternal(parentIndex)
	if err != nil {
		return "", err
	}

	// 2. 生成新PageIndex（使用Segment的索引生成）
	newPageIndex := segment.GenerateIndex()

	// 3. 创建DetailPage
	page, err := NewDetailPage(name, description, detail, parentIndex)
	if err != nil {
		return "", err
	}
	page.SetIndex(newPageIndex)

	// 4. 添加到系统
	if err := cs.AddPage(page); err != nil {
		return "", err
	}

	return newPageIndex, nil
}

// createContentsPageInternal 创建ContentsPage（内部方法）
func (cs *ContextSystem) createContentsPageInternal(name, description string, parentIndex PageIndex, children ...PageIndex) (PageIndex, error) {
	// 1. 获取父Page所属Segment（使用内部方法）
	var segment *Segment
	var err error

	if parentIndex != "" {
		segment, err = cs.getSegmentByPageIndexInternal(parentIndex)
		if err != nil {
			return "", err
		}
	} else {
		// 如果没有parent，使用第一个child的segment
		if len(children) > 0 {
			segment, err = cs.getSegmentByPageIndexInternal(children[0])
			if err != nil {
				return "", err
			}
		} else {
			return "", fmt.Errorf("cannot determine segment for new ContentsPage")
		}
	}

	// 2. 生成新PageIndex（使用Segment的索引生成）
	newPageIndex := segment.GenerateIndex()

	// 3. 创建ContentsPage
	page, err := NewContentsPage(name, description, parentIndex)
	if err != nil {
		return "", err
	}
	page.SetIndex(newPageIndex)

	// 4. 添加子节点
	for _, childIndex := range children {
		if err := page.AddChild(childIndex); err != nil {
			return "", err
		}
	}

	// 5. 添加到系统
	if err := cs.AddPage(page); err != nil {
		return "", err
	}

	// 6. 更新子节点的父引用
	for _, childIndex := range children {
		childPage, err := cs.GetPage(childIndex)
		if err != nil {
			continue
		}

		// 从原父节点移除（如果存在）
		oldParentIndex := childPage.GetParent()
		if oldParentIndex != "" && oldParentIndex != newPageIndex {
			oldParent, err := cs.GetPage(oldParentIndex)
			if err == nil {
				if oldParentPage, ok := oldParent.(*ContentsPage); ok {
					oldParentPage.RemoveChild(childIndex)
					// 持久化原父节点的更新
					if cs.storage != nil {
						cs.storage.Save(oldParent)
					}
				}
			}
		}

		// 更新子节点的父引用
		childPage.SetParent(newPageIndex)

		// 持久化更新
		if cs.storage != nil {
			cs.storage.Save(childPage)
		}
	}

	return newPageIndex, nil
}

// ============ 查询方法 ============

// GetChildren 获取子Page列表
func (cs *ContextSystem) GetChildren(pageIndex PageIndex) ([]Page, error) {
	page, err := cs.GetPage(pageIndex)
	if err != nil {
		return nil, err
	}

	contentsPage, ok := page.(*ContentsPage)
	if !ok {
		return nil, fmt.Errorf("page %s is not a ContentsPage", pageIndex)
	}

	children := make([]Page, 0)
	for _, childIndex := range contentsPage.GetChildren() {
		child, err := cs.GetPage(childIndex)
		if err != nil {
			continue // 跳过无法加载的子节点
		}
		children = append(children, child)
	}

	return children, nil
}

// GetParent 获取父Page
func (cs *ContextSystem) GetParent(pageIndex PageIndex) (Page, error) {
	page, err := cs.GetPage(pageIndex)
	if err != nil {
		return nil, err
	}

	parentIndex := page.GetParent()
	if parentIndex == "" {
		return nil, fmt.Errorf("page %s has no parent", pageIndex)
	}

	return cs.GetPage(parentIndex)
}

// GetAncestors 获取祖先Page列表
func (cs *ContextSystem) GetAncestors(pageIndex PageIndex) ([]Page, error) {
	ancestors := make([]Page, 0)

	currentIndex := pageIndex
	for currentIndex != "" {
		page, err := cs.GetPage(currentIndex)
		if err != nil {
			break
		}

		parentIndex := page.GetParent()
		if parentIndex == "" {
			break
		}

		parent, err := cs.GetPage(parentIndex)
		if err != nil {
			break
		}

		ancestors = append(ancestors, parent)
		currentIndex = parentIndex
	}

	return ancestors, nil
}

// FindPage 查找Page（支持名称、描述搜索）
func (cs *ContextSystem) FindPage(query string) []Page {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	results := make([]Page, 0)
	queryLower := strings.ToLower(query)

	for _, page := range cs.pages {
		if strings.Contains(strings.ToLower(page.GetName()), queryLower) ||
			strings.Contains(strings.ToLower(page.GetDescription()), queryLower) {
			results = append(results, page)
		}
	}

	return results
}
