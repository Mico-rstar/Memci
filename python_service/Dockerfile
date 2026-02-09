FROM python:3.11-slim

WORKDIR /app

# 安装系统依赖
RUN apt-get update && apt-get install -y --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

# 复制依赖文件并安装
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# 复制 proto 模块和执行器代码
COPY proto/ /app/proto/
COPY proto/__init__.py /app/proto/
COPY python_service/executor.py .

# 暴露端口
EXPOSE 50051

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD python -c "import grpc; channel = grpc.insecure_channel('localhost:50051'); stub = grpc.channel_ready_future(channel); stub.result(timeout=3)" || exit 1

# 启动服务
CMD ["python", "executor.py"]
