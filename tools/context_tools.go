package tools

import (
	"fmt"
	"go.starlark.net/starlark"
	"memci/context"
)

// ContextToolsProvider 为 AgentContext 提供 Starlark 工具注册
type ContextToolsProvider struct {
	agentContext *context.AgentContext
}

// NewContextToolsProvider 创建工具提供者
func NewContextToolsProvider(agentContext *context.AgentContext) *ContextToolsProvider {
	return &ContextToolsProvider{
		agentContext: agentContext,
	}
}

// RegisterTools 注册所有 AgentContext 工具到 Starlark 环境
func (p *ContextToolsProvider) RegisterTools() starlark.StringDict {
	return starlark.StringDict{
		// Segment 查询工具
		"get_segment":   starlark.NewBuiltin("get_segment", p.getSegmentFn),
		"list_segments": starlark.NewBuiltin("list_segments", p.listSegmentsFn),

		// Page 状态变更工具
		"update_page":    starlark.NewBuiltin("update_page", p.updatePageFn),
		"expand_details": starlark.NewBuiltin("expand_details", p.expandDetailsFn),
		"hide_details":   starlark.NewBuiltin("hide_details", p.hideDetailsFn),

		// Page 结构操作工具
		"move_page":           starlark.NewBuiltin("move_page", p.movePageFn),
		"remove_page":         starlark.NewBuiltin("remove_page", p.removePageFn),
		"create_detail_page":  starlark.NewBuiltin("create_detail_page", p.createDetailPageFn),
		"create_contents_page": starlark.NewBuiltin("create_contents_page", p.createContentsPageFn),

		// Page 查询工具
		"get_page":      starlark.NewBuiltin("get_page", p.getPageFn),
		"get_children":  starlark.NewBuiltin("get_children", p.getChildrenFn),
		"get_parent":    starlark.NewBuiltin("get_parent", p.getParentFn),
		"get_ancestors": starlark.NewBuiltin("get_ancestors", p.getAncestorsFn),
		"find_page":     starlark.NewBuiltin("find_page", p.findPageFn),
	}
}

// ============ Segment 查询工具实现 ============

// get_segment 根据 ID 获取 Segment
func (p *ContextToolsProvider) getSegmentFn(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var id string

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "id", &id); err != nil {
		return nil, err
	}

	segment, err := p.agentContext.GetSegment(context.SegmentID(id))
	if err != nil {
		return nil, fmt.Errorf("get_segment: %w", err)
	}

	return segmentToDict(segment), nil
}

// list_segments 列出所有 Segment
func (p *ContextToolsProvider) listSegmentsFn(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	segments, err := p.agentContext.ListSegments()
	if err != nil {
		return nil, fmt.Errorf("list_segments: %w", err)
	}

	elements := make([]starlark.Value, len(segments))
	for i, seg := range segments {
		elements[i] = segmentToDict(seg)
	}

	return starlark.NewList(elements), nil
}

// segmentToDict 将 context.Segment 转换为 Starlark Dict
func segmentToDict(seg context.Segment) *starlark.Dict {
	dict := starlark.NewDict(4)
	dict.SetKey(starlark.String("id"), starlark.String(string(seg.GetID())))
	dict.SetKey(starlark.String("type"), starlark.String(seg.GetType().String()))
	dict.SetKey(starlark.String("permission"), starlark.String(seg.GetPermission().String()))
	dict.SetKey(starlark.String("root_index"), starlark.String(string(seg.GetRootIndex())))
	return dict
}

// ============ Page 状态变更工具实现 ============

// update_page 更新 Page 信息
func (p *ContextToolsProvider) updatePageFn(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var pageIndex, name, description string

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "page_index", &pageIndex, "name", &name, "description", &description); err != nil {
		return nil, err
	}

	err := p.agentContext.UpdatePage(context.PageIndex(pageIndex), name, description)
	if err != nil {
		return nil, fmt.Errorf("update_page: %w", err)
	}

	return starlark.None, nil
}

// expand_details 展开 Page 详情
func (p *ContextToolsProvider) expandDetailsFn(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var pageIndex string

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "page_index", &pageIndex); err != nil {
		return nil, err
	}

	err := p.agentContext.ExpandDetails(context.PageIndex(pageIndex))
	if err != nil {
		return nil, fmt.Errorf("expand_details: %w", err)
	}

	return starlark.None, nil
}

// hide_details 隐藏 Page 详情
func (p *ContextToolsProvider) hideDetailsFn(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var pageIndex string

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "page_index", &pageIndex); err != nil {
		return nil, err
	}

	err := p.agentContext.HideDetails(context.PageIndex(pageIndex))
	if err != nil {
		return nil, fmt.Errorf("hide_details: %w", err)
	}

	return starlark.None, nil
}

// ============ Page 结构操作工具实现 ============

// move_page 移动 Page
func (p *ContextToolsProvider) movePageFn(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var source, target string

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "source", &source, "target", &target); err != nil {
		return nil, err
	}

	err := p.agentContext.MovePage(context.PageIndex(source), context.PageIndex(target))
	if err != nil {
		return nil, fmt.Errorf("move_page: %w", err)
	}

	return starlark.None, nil
}

// remove_page 删除 Page
func (p *ContextToolsProvider) removePageFn(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var pageIndex string

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "page_index", &pageIndex); err != nil {
		return nil, err
	}

	err := p.agentContext.RemovePage(context.PageIndex(pageIndex))
	if err != nil {
		return nil, fmt.Errorf("remove_page: %w", err)
	}

	return starlark.None, nil
}

// create_detail_page 创建 DetailPage
func (p *ContextToolsProvider) createDetailPageFn(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name, description, detail, parentIndex string

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "name", &name, "description", &description, "detail", &detail, "parent_index", &parentIndex); err != nil {
		return nil, err
	}

	index, err := p.agentContext.CreateDetailPage(name, description, detail, context.PageIndex(parentIndex))
	if err != nil {
		return nil, fmt.Errorf("create_detail_page: %w", err)
	}

	return starlark.String(string(index)), nil
}

// create_contents_page 创建 ContentsPage
func (p *ContextToolsProvider) createContentsPageFn(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name, description, parentIndex string
	var children *starlark.List

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs,
		"name", &name,
		"description", &description,
		"parent_index", &parentIndex,
		"children", &children,
	); err != nil {
		return nil, err
	}

	// 转换 children 列表
	childIndices := make([]context.PageIndex, children.Len())
	for i := 0; i < children.Len(); i++ {
		childStr, ok := children.Index(i).(starlark.String)
		if !ok {
			return nil, fmt.Errorf("children[%d]: expected string, got %T", i, children.Index(i))
		}
		childIndices[i] = context.PageIndex(childStr.GoString())
	}

	index, err := p.agentContext.CreateContentsPage(name, description, context.PageIndex(parentIndex), childIndices...)
	if err != nil {
		return nil, fmt.Errorf("create_contents_page: %w", err)
	}

	return starlark.String(string(index)), nil
}

// ============ Page 查询工具实现 ============

// get_page 获取 Page
func (p *ContextToolsProvider) getPageFn(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var pageIndex string

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "page_index", &pageIndex); err != nil {
		return nil, err
	}

	page, err := p.agentContext.GetPage(context.PageIndex(pageIndex))
	if err != nil {
		return nil, fmt.Errorf("get_page: %w", err)
	}

	return pageToDict(page), nil
}

// get_children 获取子 Page 列表
func (p *ContextToolsProvider) getChildrenFn(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var pageIndex string

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "page_index", &pageIndex); err != nil {
		return nil, err
	}

	pages, err := p.agentContext.GetChildren(context.PageIndex(pageIndex))
	if err != nil {
		return nil, fmt.Errorf("get_children: %w", err)
	}

	elements := make([]starlark.Value, len(pages))
	for i, page := range pages {
		elements[i] = pageToDict(page)
	}

	return starlark.NewList(elements), nil
}

// get_parent 获取父 Page
func (p *ContextToolsProvider) getParentFn(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var pageIndex string

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "page_index", &pageIndex); err != nil {
		return nil, err
	}

	page, err := p.agentContext.GetParent(context.PageIndex(pageIndex))
	if err != nil {
		return nil, fmt.Errorf("get_parent: %w", err)
	}

	if page == nil {
		return starlark.None, nil
	}

	return pageToDict(page), nil
}

// get_ancestors 获取祖先 Page 列表
func (p *ContextToolsProvider) getAncestorsFn(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var pageIndex string

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "page_index", &pageIndex); err != nil {
		return nil, err
	}

	pages, err := p.agentContext.GetAncestors(context.PageIndex(pageIndex))
	if err != nil {
		return nil, fmt.Errorf("get_ancestors: %w", err)
	}

	elements := make([]starlark.Value, len(pages))
	for i, page := range pages {
		elements[i] = pageToDict(page)
	}

	return starlark.NewList(elements), nil
}

// find_page 查找 Page
func (p *ContextToolsProvider) findPageFn(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var query string

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "query", &query); err != nil {
		return nil, err
	}

	pages := p.agentContext.FindPage(query)

	elements := make([]starlark.Value, len(pages))
	for i, page := range pages {
		elements[i] = pageToDict(page)
	}

	return starlark.NewList(elements), nil
}

// pageToDict 将 context.Page 转换为 Starlark Dict
func pageToDict(page context.Page) *starlark.Dict {
	dict := starlark.NewDict(6)
	dict.SetKey(starlark.String("index"), starlark.String(string(page.GetIndex())))
	dict.SetKey(starlark.String("name"), starlark.String(page.GetName()))
	dict.SetKey(starlark.String("description"), starlark.String(page.GetDescription()))
	dict.SetKey(starlark.String("lifecycle"), starlark.String(page.GetLifecycle().String()))
	dict.SetKey(starlark.String("visibility"), starlark.String(page.GetVisibility().String()))

	// 判断页面类型
	var pageType string
	if _, ok := page.(*context.DetailPage); ok {
		pageType = "DetailPage"
	} else if _, ok := page.(*context.ContentsPage); ok {
		pageType = "ContentsPage"
	} else {
		pageType = "Unknown"
	}
	dict.SetKey(starlark.String("type"), starlark.String(pageType))

	return dict
}
