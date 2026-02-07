## Idea
Memci的工具调用主要是对自定义工具协议ATTP的实现
[ATTP](./ATTP.md)
一些不同之处在于：
1. 使用starlark而非python作为Agent进行工具调用的语言。starlark可以看作python语言的子集，它的基本语法和python几乎一致，但不能使用import引入外部库，同时 starlark-go 提供了一个非常低成本的go语言调用starlark的方案
2. 工具调用schema并不完全一致


## 工具协议
### 基本工具定义

#### 设计思路

通过 starlark-go 的 `go-starlark` API 将 Go 函数注册为 Starlark 可调用函数。每个 `AgentContext` 方法对应一个 Starlark 工具函数。



#### Starlark 工具签名规范

```starlark
# ============ Segment 查询工具 ============

# get_segment 根据 ID 获取 Segment
# 参数: id (string) - Segment ID
# 返回: dict - Segment 信息 {id, type, permission, root_index}
get_segment(id: str) -> dict

# list_segments 列出所有 Segment
# 返回: list[dict] - Segment 列表
list_segments() -> list

# ============ Page 状态变更工具 ============

# update_page 更新 Page 信息
# 参数: page_index (str), name (str), description (str)
# 返回: None
update_page(page_index: str, name: str, description: str) -> None

# expand_details 展开 Page 详情
# 参数: page_index (str)
# 返回: None
expand_details(page_index: str) -> None

# hide_details 隐藏 Page 详情
# 参数: page_index (str)
# 返回: None
hide_details(page_index: str) -> None

# ============ Page 结构操作工具 ============

# move_page 移动 Page
# 参数: source (str), target (str)
# 返回: None
move_page(source: str, target: str) -> None

# remove_page 删除 Page
# 参数: page_index (str)
# 返回: None
remove_page(page_index: str) -> None

# create_detail_page 创建 DetailPage
# 参数: name (str), description (str), detail (str), parent_index (str)
# 返回: str - 新 Page 的 index
create_detail_page(name: str, description: str, detail: str, parent_index: str) -> str

# create_contents_page 创建 ContentsPage
# 参数: name (str), description (str), parent_index (str), children (list[str])
# 返回: str - 新 Page 的 index
create_contents_page(name: str, description: str, parent_index: str, children: list) -> str

# ============ Page 查询工具 ============

# get_page 获取 Page
# 参数: page_index (str)
# 返回: dict - Page 信息 {index, name, description, type, has_detail}
get_page(page_index: str) -> dict

# get_children 获取子 Page 列表
# 参数: page_index (str)
# 返回: list[dict] - 子 Page 列表
get_children(page_index: str) -> list

# get_parent 获取父 Page
# 参数: page_index (str)
# 返回: dict or None - 父 Page 信息
get_parent(page_index: str) -> dict | None

# get_ancestors 获取祖先 Page 列表
# 参数: page_index (str)
# 返回: list[dict] - 祖先 Page 列表（从父到根）
get_ancestors(page_index: str) -> list

# find_page 查找 Page
# 参数: query (str)
# 返回: list[dict] - 匹配的 Page 列表
find_page(query: str) -> list
```


#### 工具调用示例

```starlark
# Agent 侧调用示例

# 1. 查询所有 Segment
segments = list_segments()
for seg in segments:
    print(f"Segment: {seg['id']}, Type: {seg['type']}")

# 2. 获取特定 Page
page = get_page("page:001")
print(f"Page: {page['name']}")

# 3. 创建新 Page
new_index = create_detail_page(
    name="新想法",
    description="关于工具调用的思考",
    detail="详细内容...",
    parent_index="page:001"
)

# 4. 展开/隐藏详情
expand_details(new_index)
hide_details(new_index)

# 5. 搜索
results = find_page("工具")
for r in results:
    print(f"Found: {r['name']}")

# 6. 结果赋值（符合 schema）
__result__ = new_index
```


### 使用规范：

```toml
[tool_call]
target = "创建新的详情页"
code = '''
# 规则：
# 1. 工具函数直接调用
# 2. 结果必须赋值给 __result__
# 3. __result__ 可为任意类型（str/dict/list/None）
# 4. 系统只会返回 __result__ 中的值，直接调用 print 什么都不会得到

__result__ = create_detail_page(
    name="新想法",
    description="工具调用的思考",
    detail="详细内容...",
    parent_index="page:001"
)
'''
```

