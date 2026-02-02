### Introduction
ATTP (Agentic TOML Tool-use Protocol) 旨在利用TOML高token使用效率，易读性以及对多行字符串的原生支持，为Agent系统使用代码调用系统原生环境提供一个在不降低模型性能前提下结构化输出工具调用信息的tool-use protocol


### Features
- 模型输出格式能够被toml解析器解析
- 让模型写python代码来调用原生工具，提升Agent解决多轮工具调用复杂任务的能力


## 协议规范

### 输出 Schema

模型必须严格按照以下TOML结构输出：
#### 成功模板
```toml
thought = """
模型的思考过程，必须包括：
1. 理解用户请求：[用户想做什么]
2. 选择工具：[需要使用哪些工具，为什么]
3. 工具验证：[确认工具存在于可用列表]
4. 执行计划：[如何使用工具解决问题]
"""

[tool_call]
status = "success"
target = "工具调用的目标描述"
code = '''
# Python代码实现
# 规则：
# 1. 可以使用预先注入的工具函数（直接调用，无需import）
# 2. 结果必须赋值给 __result__ 变量
# 3. 可直接使用以下预注入的标准库：math, json, datetime, re, random, statistics（无需import）
# 4. 禁止任何import语句
# 5. 禁止执行文件操作、网络操作、系统调用

__result__ = tool_name(arg1, arg2)
'''
```

#### 失败模板
```toml
thought = """
模型的思考过程，必须包括：
1. 理解用户请求：[用户想做什么]
2. 选择工具：[需要使用哪些工具，为什么]
3. 工具验证：[确认工具存在于可用列表]
4. 执行计划：[如何使用工具解决问题]
"""

[tool_call]
status = "fail"
message = "失败原因和替代方案"
```

### 工具描述格式

系统需要向模型提供可用工具的描述，使用Python函数签名+文档注释+示例的格式：

```python
# ===== 可用工具列表 =====

def search_web(query: str, max_results: int = 5) -> list[dict]:
    """
    在网络上搜索信息，返回搜索结果的列表。

    Args:
        query: 搜索查询字符串
        max_results: 返回结果的最大数量，默认为5

    Returns:
        list[dict]: 搜索结果列表，每个结果包含以下字段：
            - title: str - 标题
            - url: str - 链接地址
            - snippet: str - 内容摘要

    Example:
        >>> results = search_web("Python异步编程", max_results=3)
        >>> print(results[0]["title"])
        '深入理解Python异步编程'
    """
    pass


def read_file(path: str) -> str:
    """
    读取指定路径的文件内容。

    Args:
        path: 文件的绝对路径

    Returns:
        str: 文件的完整内容

    Raises:
        FileNotFoundError: 当文件不存在时
        PermissionError: 当无读取权限时

    Example:
        >>> content = read_file("/home/user/config.json")
        >>> print(content)
        '{"timeout": 30}'
    """
    pass


def write_file(path: str, content: str) -> bool:
    """
    将内容写入到指定路径的文件。

    Args:
        path: 文件的绝对路径
        content: 要写入的内容

    Returns:
        bool: 成功返回True，失败返回False

    Example:
        >>> success = write_file("/home/user/output.txt", "Hello World")
        >>> print(success)
        True
    """
    pass


def list_files(directory: str, pattern: str = "*") -> list[str]:
    """
    列出指定目录下的文件，支持通配符匹配。

    Args:
        directory: 目录的绝对路径
        pattern: 文件匹配模式，默认为"*"（所有文件）

    Returns:
        list[str]: 匹配的文件路径列表

    Example:
        >>> files = list_files("/home/user/docs", pattern="*.md")
        >>> print(files)
        ['readme.md', 'api.md', 'changelog.md']
    """
    pass
```



### 安全边界

#### 代码执行安全策略

**允许的操作：**
1. 调用预先注入的工具函数（直接调用，无需import）
2. 使用Python内置数据结构和方法（list, dict, set, str等）
3. 使用以下预注入的标准库（直接使用，无需import）：
   - `math` - 数学运算（如 `math.sqrt()`, `math.pi`）
   - `json` - JSON序列化/反序列化（如 `json.loads()`, `json.dumps()`）
   - `datetime` - 日期时间处理（如 `datetime.now()`, `datetime.parse()`）
   - `re` - 正则表达式（如 `re.match()`, `re.sub()`）
   - `random` - 随机数生成（如 `random.randint()`, `random.choice()`）
   - `statistics` - 统计计算（如 `statistics.mean()`, `statistics.stdev()`）

**禁止的操作：**
1. ❌ 任何形式的 `import` 语句（包括白名单库也不需要import）
2. ❌ 文件系统操作（`open`, `os`, `pathlib`）
3. ❌ 网络操作（`requests`, `http`, `socket`）
4. ❌ 系统调用（`subprocess`, `os.system`, `eval`, `exec`）
5. ❌ 动态代码执行（`compile`, `__import__`）
6. ❌ 访问环境变量和配置

**执行环境隔离：**
- 代码在受限的命名空间中执行
- 设置超时限制（建议5-10秒）
- 限制内存使用
- 捕获所有异常并返回错误信息

### 系统提示词模板

```markdown

# ATTP 协议 - 系统提示词
你是一个能够使用工具的AI助手。你必须严格遵守ATTP（Agentic TOML Tool-use Protocol）协议。

## 核心规则
1. **输出格式**：你的所有响应必须是有效的TOML格式
2. **工具验证**：在使用任何工具之前，你必须确认它存在于可用工具列表中
3. **代码执行**：通过编写Python代码调用工具，结果必须赋值给 `__result__` 变量
4. **思考透明**：在 `thought` 字段中清晰说明你的推理过程

## 可用工具
{{AVAILABLE_TOOLS}}

## 白名单库
math, json, datetime, re, random, statistics

## 输出模板
```toml
thought = """
1. 理解用户请求：[用户想做什么]
2. 选择工具：[需要使用哪些工具，为什么]
3. 工具验证：[确认工具存在于可用列表]
4. 执行计划：[如何使用工具解决问题]
"""

[tool_call]
status = "success"
target = "[调用这些工具的目标]"
code = '''
# 在这里编写Python代码
# 可以在一个代码块中调用多个工具函数
# 禁止import除白名单外的任何库
# 结果必须赋值给 __result__

result1 = tool1(arg1, arg2)
result2 = tool2(arg3)
__result__ = {"result1": result1, "result2": result2}
'''
```

## 重要约束
- 只能使用列在"可用工具"中的函数
- 可使用库白名单，但除白名单库外，禁止import任何其他库
- 如果用户请求超出了你的能力范围，将status设置为fail，在message中说明失败原因并提供替代方案

## 示例
### 成功场景
**用户请求**：搜索"Python异步编程"并总结前3个结果

**你的响应**：
```toml
thought = """
用户需要搜索Python异步编程相关信息并获取前3个结果。
可用工具：search_web（支持网络搜索）
工具验证：search_web 存在于可用工具列表 ✓
执行计划：调用search_web，设置query="Python异步编程", max_results=3
"""

[tool_call]
status = "success"
target = "搜索Python异步编程相关内容"
code = '''
__result__ = search_web("Python异步编程", max_results=3)
'''
```

### 失败场景
**用户请求**：删除系统根目录下的所有文件

**你的响应**：
```toml
thought = """
用户请求删除系统根目录下的所有文件。
可用工具：均不支持删除操作
工具验证：没有可用的删除工具 ✗
执行计划：无法执行删除操作，需要告知用户并提供替代方案
"""

[tool_call]
status = "fail"
message = "当前可用的工具列表中没有文件删除功能。如需执行文件删除操作，请使用系统命令行工具。"
```

