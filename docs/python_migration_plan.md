# Starlark 到 Python 解释器迁移方案

## 当前架构分析

### 现有组件
```
LLM Response (Starlark 代码)
        ↓
Starlark Executor (go.starlark.net)
        ↓
Go Builtin 函数 (ContextToolsProvider)
        ↓
ContextManager → ContextSystem
```

### 限制
- Starlark 是 Python 的子集，缺少很多特性
- LLM 熟悉 Python 但不熟悉 Starlark 语法限制
- 限制包括：无类、无异常、无装饰器、有限的标准库

## 迁移方案设计

## 推荐实现：方案四（容器化 Python + gRPC）

### 详细设计

#### 1. 目录结构
```
tools/
├── python_executor.go       # Python 解释器封装
├── python_type_conv.go     # Go ↔ Python 类型转换
├── go_wrapper.go           # Go 函数包装器
├── starlark_executor.go    # 保留，用于兼容
└── context_tools.go        # 工具注册（保持不变）
```

#### 2. PythonExecutor 接口
```go
// 保持与 Starlark Executor 相同的接口
type ScriptExecutor interface {
    Execute(code string) (interface{}, error)
}

type PythonExecutor struct {
    interpreter *python.PyInterpreter
    env         *python.PyDict
}

func NewPythonExecutor() (*PythonExecutor, error) {
    // 初始化 Python 解释器
    python.Py_Initialize()

    // 创建执行环境
    env := python.PyDict_New()

    return &PythonExecutor{
        interpreter: python.PyInterpreter_New(),
        env:         env,
    }
}

func (e *PythonExecutor) Execute(code string) (interface{}, error) {
    // 执行 Python 代码
    result, err := e.interpreter.Eval(code, e.env, e.env)
    if err != nil {
        return nil, fmt.Errorf("python execution failed: %w", err)
    }

    // 提取 __result__ (与 Starlark 保持一致)
    resultObj := e.env.GetItem("__result__")
    if resultObj == nil {
        return nil, nil
    }

    return pythonToGo(resultObj), nil
}
```

#### 3. Go 函数包装
```go
// 注册 Go 函数到 Python 环境
func (e *PythonExecutor) RegisterGoFunc(name string, fn interface{}) error {
    wrappedFunc := wrapGoFunc(fn)
    return e.env.SetItem(name, wrappedFunc)
}

// 包装 Go 函数为 Python 可调用对象
func wrapGoFunc(fn interface{}) *python.PyObject {
    return python.NewCFunction(func(args *python.PyTuple, kwargs python.PyDict) (*python.PyObject, error) {
        // 解析参数
        goArgs, err := pythonTupleToGoSlice(args)
        if err != nil {
            return nil, err
        }

        goKwargs, err := pythonDictToGoMap(kwargs)
        if err != nil {
            return nil, err
        }

        // 反射调用 Go 函数
        fnValue := reflect.ValueOf(fn)
        var callArgs []reflect.Value

        // 处理参数
        if goKwargs != nil {
            // 有命名参数的情况
            // ... 需要根据函数签名处理
        }

        results := fnValue.Call(callArgs)

        // 转换结果
        if results.Kind() == reflect.Interface && !results.IsNil() {
            return goToPython(results.Interface())
        }

        python.Py_IncRef(python.Py_None())
        return python.Py_None(), nil
    })
}
```

#### 4. 工具注册
```go
// 修改 ContextToolsProvider 支持两种执行器
type ContextToolsProvider struct {
    agentContext *context.AgentContext
    executor     ScriptExecutor
    usePython    bool  // 配置项，选择使用 Python
}

func (p *ContextToolsProvider) RegisterTools() interface{} {
    if p.usePython {
        return p.registerPythonTools()
    }
    return p.registerStarlarkTools()
}

func (p *ContextToolsProvider) registerPythonTools() *python.PyDict {
    env := python.PyDict_New()

    // 注册工具函数
    p.registerPythonTool(env, "update_page", p.updatePagePython)
    p.registerPythonTool(env, "expand_details", p.expandDetailsPython)
    // ... 其他工具

    return env
}
```

#### 5. 类型转换
```go
// Go → Python
func goToPython(v interface{}) *python.PyObject {
    switch val := v.(type) {
    case string:
        return python.PyUnicode_FromString(val)
    case int:
        return python.PyLong_FromLong(int64(val))
    case float64:
        return python.PyFloat_FromDouble(val)
    case bool:
        if val {
            python.Py_IncRef(python.Py_True())
            return python.Py_True()
        }
        python.Py_IncRef(python.Py_False())
        return python.Py_False()
    case map[string]interface{}:
        return mapToPythonDict(val)
    case []interface{}:
        return sliceToPythonList(val)
    case nil:
        python.Py_IncRef(python.Py_None())
        return python.Py_None()
    default:
        // 其他类型转为字符串
        return python.PyUnicode_FromString(fmt.Sprintf("%v", val))
    }
}

// Python → Go
func pythonToGo(obj *python.PyObject) interface{} {
    if obj == nil {
        return nil
    }

    // 检查类型并转换
    if python.PyUnicode_Check(obj) {
        return python.PyUnicode_AsUTF8String(obj)
    }
    if python.PyLong_Check(obj) {
        return int(python.PyLong_AsLong(obj))
    }
    if python.PyFloat_Check(obj) {
        return python.PyFloat_AsDouble(obj)
    }
    if python.PyBool_Check(obj) {
        return obj == python.Py_True()
    }
    if python.PyDict_Check(obj) {
        return pythonDictToMap(obj)
    }
    if python.PyList_Check(obj) {
        return pythonListToSlice(obj)
    }
    if obj == python.Py_None() {
        return nil
    }

    // 其他类型转为字符串
    return python.PyObject_Str(obj)
}
```

---

## 配置选项

### 在 config.toml 中添加配置
```toml
[agent.script_executor]
type = "python"  # "starlark" 或 "python"
enabled = true
```

### 启动时选择
```go
func NewContextToolsProvider(agentCtx *context.AgentContext, usePython bool) *ContextToolsProvider {
    return &ContextToolsProvider{
        agentContext: agentCtx,
        usePython:    usePython,
    }
}
```

---

## 迁移路径

### 阶段一：并行支持（当前）
- 保持 Starlark 解释器
- 新增 Python 解释器
- 通过配置选择使用哪个

### 阶段二：默认 Python
- Python 解释器成为默认
- Starlark 保留用于向后兼容

### 阶段三：完全迁移
- 移除 Starlark 依赖
- 纯 Python 实现

---

## 依赖项

### go-python
```go
import "github.com/stretchr/testify/assert"
import "github.com/go-python/gpython/py"
```

### 安装
```bash
go get github.com/go-python/gpython
```

---

## 代码示例

### Python 风格的工具调用
```python
# 使用完整的 Python 语法
result = create_detail_page(
    name="用户想法",
    description="今天的一个想法",
    detail="用户建议使用更自然的对话方式",
    parent_index="usr-10"
)

# 使用类和装饰器
class PageBuilder:
    def __init__(self, prefix):
        self.prefix = prefix

    def create(self, name, detail):
        return create_detail_page(
            name=f"{self.prefix}-{name}",
            description=f"{self.prefix}的子页面",
            detail=detail,
            parent_index=""
        )

builder = PageBuilder("usr")
builder.create("设计想法", "考虑使用树形结构")

# 使用异常处理
try:
    result = get_page("usr-1")
except Exception as e:
    result = f"Error: {e}"

# 使用标准库
import json
data = json.dumps({"pages": ["usr-1", "usr-2"]})
```

### 与 Starlark 对比
```python
# Starlark (受限)
result = create_detail_page("name", "desc", "detail", "parent_index")

# Python (完整功能)
result = create_detail_page(
    name="页面名称",
    description="页面描述",
    detail="详细内容",
    parent_index="usr-1"
)

# 使用 Python 特性
pages = []
for i in range(3):
    pages.append(create_detail_page(f"page{i}", "", "", parent_index))
```

---

## 测试策略

### 单元测试
```go
func TestPythonExecutor(t *testing.T) {
    executor, _ := NewPythonExecutor()

    // 测试简单表达式
    result, _ := executor.Execute("__result__ = 42")
    assert.Equal(t, 42, result)

    // 测试函数调用
    executor.RegisterGoFunc("add", func(a, b int) int { return a + b })
    result, _ = executor.Execute("__result__ = add(1, 2)")
    assert.Equal(t, 3, result)
}
```

### 兼容性测试
```go
// 确保相同的输入在两种解释器下产生相同结果
func testCompatibility(t *testing.T) {
    starlarkExec := NewStarlarkExecutor()
    pythonExec := NewPythonExecutor()

    code := `__result__ = create_detail_page("test", "desc", "detail", "parent")`

    starlarkResult, _ := starlarkExec.Execute(code)
    pythonResult, _ := pythonExec.Execute(code)

    assert.Equal(t, starlarkResult, pythonResult)
}
```

---

## 注意事项

### 1. 沙箱安全
- 限制 Python 标准库访问（os, subprocess, eval 等）
- 设置执行超时
- 内存限制

### 2. 错误处理
- Python 异常到 Go 错误的转换
- 提供详细的错误信息

### 3. 性能
- Python 解释器初始化成本
- 考虑复用解释器实例
- 可选：预编译常用脚本

### 4. 部署
- 需要确保目标环境有 Python
- 可选：静态链接 libpython
- 考虑提供无 Python 的回退方案

---

## 方案对比

| 方案 | 优势 | 劣势 | 推荐度 |
|------|------|------|--------|
| go-python | 成熟稳定，社区活跃 | 需要 cgo，依赖 libpython | ⭐⭐⭐ |
| gopy | 类型安全，双向调用 | 代码生成复杂，适合反向调用 | ⭐⭐ |
| cpython C API | 完全控制 | 复杂，易出错，需要处理引用计数 | ⭐ |
| **容器化 Python + gRPC** | **完全隔离，部署灵活，无需 cgo** | **网络延迟，架构复杂** | **⭐⭐⭐⭐⭐** |

---

## 总结

**推荐方案：** 使用容器化 Python + gRPC（方案四）

**核心优势：**
1. 完整 Python 语法，LLM 更熟悉
2. 安全隔离，Python 代码在独立容器运行
3. 无需 cgo，部署简单
4. 可独立扩展 Python 服务
5. 依赖管理简单

---

## 方案四：容器化 Python + gRPC 详细设计

### 架构图

```
┌─────────────────┐     gRPC      ┌─────────────────┐
│   Go Service    │◄──────────────►│  Python Service │
│                 │                 │   (Container)   │
│  - Agent        │                 │                 │
│  - ContextMgr   │                 │  - Python Exec  │
│  - gRPC Client  │                 │  - Type Conv    │
└─────────────────┘                 └─────────────────┘
        ↑                                      │
        │                                      ↓
   LLM Response                        Python Code
   (Python code)                        Execution
        │
        ↓
   ┌─────────┐
   │   LLM   │
   └─────────┘
```

### 1. Protobuf 接口定义

**proto/executor.proto:**
```protobuf
syntax = "proto3";

package executor;

option go_package = "memci/proto";

// Python 执行服务
service PythonExecutor {
  // 执行 Python 代码
  rpc Execute(ExecuteRequest) returns (ExecuteResponse);

  // 健康检查
  rpc Health(HealthRequest) returns (HealthResponse);
}

// 执行请求
message ExecuteRequest {
  string code = 1;           // Python 代码
  map<string, Value> context = 2;  // 可用的变量和函数
  int32 timeout = 3;         // 超时时间（秒）
}

// 执行响应
message ExecuteResponse {
  bool success = 1;
  Value result = 2;
  string error = 3;
  repeated string logs = 4;  // 执行日志
}

// 通用值类型
message Value {
  oneof value {
    string str_value = 1;
    int64 int_value = 2;
    double float_value = 3;
    bool bool_value = 4;
    NullValue null_value = 5;
    ListValue list_value = 6;
    Struct map_value = 7;
    FunctionValue function_value = 8;  // Go 函数引用
  }
}

enum NullValue {
  NULL_VALUE = 0;
}

message ListValue {
  repeated Value values = 1;
}

message Struct {
  map<string, Value> fields = 1;
}

// Go 函数引用
message FunctionValue {
  string name = 1;  // 函数名称
}

// 健康检查
message HealthRequest {}

message HealthResponse {
  bool healthy = 1;
  string version = 2;
}
```

### 2. Python gRPC 服务实现

**python_service/executor.py:**
```python
import grpc
from concurrent import futures
import executor_pb2
import executor_pb2_grpc
import sys
import io
import contextlib

# Python 执行器服务
class PythonExecutorServicer(executor_pb2_grpc.PythonExecutorServicer):
    def __init__(self):
        self.go_functions = {}  # name -> callable

    def register_go_function(self, name: str, func):
        """注册 Go 函数"""
        self.go_functions[name] = func

    def Execute(self, request, context):
        """执行 Python 代码"""
        try:
            # 捕获输出
            output = io.StringIO()

            # 准备执行环境（限制的内置函数）
            exec_globals = {
                '__builtins__': {
                    'print': lambda *args, **kwargs: print(*args, file=output, **kwargs),
                    'len': len, 'range': range, 'str': str, 'int': int,
                    'float': float, 'bool': bool, 'list': list,
                    'dict': dict, 'tuple': tuple, 'set': set,
                }
            }

            # 注册 Go 函数
            for name, func in self.go_functions.items():
                exec_globals[name] = func

            # 注入上下文变量
            for key, value in request.context.items():
                exec_globals[key] = protobuf_to_python(value)

            # 执行代码
            exec(request.code, exec_globals)

            # 提取结果
            result = exec_globals.get('__result__', None)

            return executor_pb2.ExecuteResponse(
                success=True,
                result=python_to_protobuf(result),
                logs=[output.getvalue()]
            )

        except Exception as e:
            return executor_pb2.ExecuteResponse(
                success=False,
                error=str(e)
            )

    def Health(self, request, context):
        return executor_pb2.HealthResponse(
            healthy=True,
            version="1.0.0"
        )

def protobuf_to_python(value):
    """将 Protobuf Value 转换为 Python 对象"""
    which = value.WhichOneof("value")
    if which == "str_value":
        return value.str_value
    elif which == "int_value":
        return value.int_value
    elif which == "float_value":
        return value.float_value
    elif which == "bool_value":
        return value.bool_value
    elif which == "null_value":
        return None
    elif which == "list_value":
        return [protobuf_to_python(v) for v in value.list_value.values]
    elif which == "map_value":
        return {k: protobuf_to_python(v) for k, v in value.map_value.fields.items()}
    return None

def python_to_protobuf(value):
    """将 Python 对象转换为 Protobuf Value"""
    result = executor_pb2.Value()

    if value is None:
        result.null_value = executor_pb2.NULL_VALUE
    elif isinstance(value, str):
        result.str_value = value
    elif isinstance(value, int):
        result.int_value = value
    elif isinstance(value, float):
        result.float_value = value
    elif isinstance(value, bool):
        result.bool_value = value
    elif isinstance(value, list):
        list_value = result.list_value
        for item in value:
            list_value.values.append(python_to_protobuf(item))
    elif isinstance(value, dict):
        struct = result.map_value.fields
        for k, v in value.items():
            struct[k] = python_to_protobuf(v)
    else:
        result.str_value = str(value)

    return result

def serve(port=50051):
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    executor_pb2_grpc.add_PythonExecutorServicer_to_server(
        PythonExecutorServicer(), server
    )
    server.add_insecure_port(f'[::]:{port}')
    server.start()
    print(f"Python Executor Server started on port {port}")
    server.wait_for_termination()

if __name__ == '__main__':
    serve()
```

### 3. Go gRPC 客户端实现

**tools/python_grpc_executor.go:**
```go
package tools

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	executorpb "memci/proto"
)

// PythonGRPCExecutor Python 执行器（gRPC 客户端）
type PythonGRPCExecutor struct {
	conn    *grpc.ClientConn
	client  executorpb.PythonExecutorClient
	timeout time.Duration
}

// NewPythonGRPCExecutor 创建新的 Python gRPC 执行器
func NewPythonGRPCExecutor(address string, timeout time.Duration) (*PythonGRPCExecutor, error) {
	conn, err := grpc.Dial(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to python service: %w", err)
	}

	return &PythonGRPCExecutor{
		conn:    conn,
		client:  executorpb.NewPythonExecutorClient(conn),
		timeout: timeout,
	}, nil
}

// Execute 执行 Python 代码
func (e *PythonGRPCExecutor) Execute(code string, context map[string]interface{}) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	// 转换上下文
	pbContext := goToProtobufMap(context)

	// 执行请求
	req := &executorpb.ExecuteRequest{
		Code:    code,
		Context: pbContext,
		Timeout: int32(e.timeout.Seconds()),
	}

	resp, err := e.client.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("grpc execute failed: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("python execution failed: %s", resp.Error)
	}

	// 转换结果
	return protobufToGo(resp.Result), nil
}

// Close 关闭连接
func (e *PythonGRPCExecutor) Close() error {
	return e.conn.Close()
}

// Health 健康检查
func (e *PythonGRPCExecutor) Health() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := e.client.Health(ctx, &executorpb.HealthRequest{})
	if err != nil {
		return false, err
	}
	return resp.Healthy, nil
}
```

### 4. Docker 配置

**python_service/Dockerfile:**
```dockerfile
FROM python:3.11-slim

WORKDIR /app

# 安装依赖
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# 复制代码
COPY executor_pb2.py .
COPY executor_pb2_grpc.py .
COPY executor.py .

# 暴露端口
EXPOSE 50051

# 启动服务
CMD ["python", "executor.py"]
```

**python_service/requirements.txt:**
```
grpcio==1.60.0
grpcio-tools==1.60.0
protobuf==4.25.1
```

**docker-compose.yml:**
```yaml
version: '3.8'

services:
  memci-go:
    build: .
    ports:
      - "8080:8080"
    depends_on:
      - python-executor
    environment:
      - PYTHON_EXECUTOR_ADDR=python-executor:50051
      - SCRIPT_EXECUTOR_TYPE=grpc

  python-executor:
    build: ./python_service
    ports:
      - "50051:50051"
    restart: unless-stopped
```

### 5. 配置更新

**config.toml:**
```toml
[agent.script_executor]
type = "grpc"  # "starlark", "python" (嵌入), 或 "grpc" (容器)

[agent.script_executor.grpc]
address = "localhost:50051"
timeout = 30  # 秒
```

### 6. 部署流程

**生成 Protobuf 代码：**
```bash
# 生成 Python 代码
python -m grpc_tools.protoc \
    --proto_path=. \
    --python_out=. \
    --grpc_python_out=. \
    proto/executor.proto

# 生成 Go 代码
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/executor.proto
```

**构建和启动：**
```bash
# 构建并启动所有服务
docker-compose up -d

# 单独启动 Python 服务
cd python_service
docker build -t memci-python-executor .
docker run -p 50051:50051 memci-python-executor
```

### 7. 实施步骤

1. 创建 proto 目录和 executor.proto 文件
2. 生成 Go 和 Python 的 gRPC 代码
3. 实现 Python gRPC 服务
4. 实现 Go gRPC 客户端
5. 创建 Python 服务的 Dockerfile
6. 创建 docker-compose.yml
7. 更新配置文件
8. 编写测试
9. 部署和调试
