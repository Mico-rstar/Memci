import grpc
from concurrent import futures
import sys
import io
import contextlib
import logging

# 添加项目根目录到路径
sys.path.insert(0, '/home/rene/projs/Memci')

from proto import executor_pb2
from proto import executor_pb2_grpc

# 配置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)


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


class PythonExecutorServicer(executor_pb2_grpc.PythonExecutorServicer):
    """Python 执行器服务"""

    def __init__(self):
        self.go_functions = {}  # name -> callable

    def register_go_function(self, name: str, func):
        """注册 Go 函数（将在后续实现中通过反向调用完成）"""
        self.go_functions[name] = func
        logger.info(f"Registered Go function: {name}")

    def Execute(self, request, context):
        """执行 Python 代码"""
        try:
            # 捕获输出
            output = io.StringIO()

            # 准备执行环境（限制的内置函数，安全考虑）
            exec_globals = {
                '__builtins__': {
                    'print': lambda *args, **kwargs: print(*args, file=output, **kwargs),
                    'len': len,
                    'range': range,
                    'str': str,
                    'int': int,
                    'float': float,
                    'bool': bool,
                    'list': list,
                    'dict': dict,
                    'tuple': tuple,
                    'set': set,
                    'sum': sum,
                    'min': min,
                    'max': max,
                    'abs': abs,
                    'all': all,
                    'any': any,
                    'enumerate': enumerate,
                    'zip': zip,
                    'sorted': sorted,
                    'reversed': reversed,
                    'map': map,
                    'filter': filter,
                    # 排除危险函数：eval, exec, open, __import__, etc.
                }
            }

            # 注册 Go 函数
            for name, func in self.go_functions.items():
                exec_globals[name] = func

            # 注入上下文变量
            for key, value in request.context.items():
                exec_globals[key] = protobuf_to_python(value)

            # 执行代码
            logger.info(f"Executing code: {request.code[:100]}...")
            exec(request.code, exec_globals)

            # 提取结果
            result = exec_globals.get('__result__', None)

            logger.info(f"Execution successful, result type: {type(result)}")

            return executor_pb2.ExecuteResponse(
                success=True,
                result=python_to_protobuf(result),
                logs=[output.getvalue()]
            )

        except Exception as e:
            logger.error(f"Execution failed: {str(e)}", exc_info=True)
            return executor_pb2.ExecuteResponse(
                success=False,
                error=str(e)
            )

    def Health(self, request, context):
        """健康检查"""
        return executor_pb2.HealthResponse(
            healthy=True,
            version="1.0.0"
        )


def serve(port=50051):
    """启动 gRPC 服务器"""
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    executor_pb2_grpc.add_PythonExecutorServicer_to_server(
        PythonExecutorServicer(), server
    )
    server.add_insecure_port(f'[::]:{port}')
    server.start()
    logger.info(f"Python Executor Server started on port {port}")
    server.wait_for_termination()


if __name__ == '__main__':
    serve()
