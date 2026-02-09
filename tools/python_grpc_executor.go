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
	// 不使用 WithBlock()，避免阻塞启动
	conn, err := grpc.Dial(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(10*1024*1024), // 10MB
			grpc.MaxCallSendMsgSize(10*1024*1024), // 10MB
		),
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
func (e *PythonGRPCExecutor) Execute(code string, execContext map[string]any) (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	// 转换上下文
	pbContext := goToProtobufMap(execContext)

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

// ============ 类型转换函数 ============

// goToProtobufMap 将 Go map 转换为 Protobuf map
func goToProtobufMap(m map[string]any) map[string]*executorpb.Value {
	result := make(map[string]*executorpb.Value)
	for k, v := range m {
		result[k] = goToProtobuf(v)
	}
	return result
}

// goToProtobuf 将 Go 类型转换为 Protobuf Value
func goToProtobuf(v any) *executorpb.Value {
	value := &executorpb.Value{}

	switch val := v.(type) {
	case string:
		value.Value = &executorpb.Value_StrValue{StrValue: val}
	case int:
		value.Value = &executorpb.Value_IntValue{IntValue: int64(val)}
	case int64:
		value.Value = &executorpb.Value_IntValue{IntValue: val}
	case float64:
		value.Value = &executorpb.Value_FloatValue{FloatValue: val}
	case bool:
		value.Value = &executorpb.Value_BoolValue{BoolValue: val}
	case nil:
		value.Value = &executorpb.Value_NullValue{NullValue: executorpb.NullValue_NULL_VALUE}
	case []any:
		listValue := &executorpb.ListValue{}
		for _, item := range val {
			listValue.Values = append(listValue.Values, goToProtobuf(item))
		}
		value.Value = &executorpb.Value_ListValue{ListValue: listValue}
	case map[string]any:
		structVal := &executorpb.Struct{}
		for key, item := range val {
			structVal.Fields[key] = goToProtobuf(item)
		}
		value.Value = &executorpb.Value_MapValue{MapValue: structVal}
	default:
		value.Value = &executorpb.Value_StrValue{StrValue: fmt.Sprintf("%v", val)}
	}

	return value
}

// protobufToGo 将 Protobuf Value 转换为 Go 类型
func protobufToGo(v *executorpb.Value) any {
	if v == nil {
		return nil
	}

	switch val := v.Value.(type) {
	case *executorpb.Value_StrValue:
		return val.StrValue
	case *executorpb.Value_IntValue:
		return val.IntValue
	case *executorpb.Value_FloatValue:
		return val.FloatValue
	case *executorpb.Value_BoolValue:
		return val.BoolValue
	case *executorpb.Value_NullValue:
		return nil
	case *executorpb.Value_ListValue:
		result := make([]any, len(val.ListValue.Values))
		for i, item := range val.ListValue.Values {
			result[i] = protobufToGo(item)
		}
		return result
	case *executorpb.Value_MapValue:
		result := make(map[string]any)
		for key, item := range val.MapValue.Fields {
			result[key] = protobufToGo(item)
		}
		return result
	case *executorpb.Value_FunctionValue:
		// 函数引用，返回名称字符串
		return val.FunctionValue.Name
	default:
		return nil
	}
}
