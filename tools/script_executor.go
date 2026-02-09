package tools

import (
	"fmt"
	"time"

	"go.starlark.net/starlark"
	"memci/config"
	"memci/context"
)

// ScriptExecutor 脚本执行器接口
type ScriptExecutor interface {
	Execute(code string, execContext map[string]any) (any, error)
}

// ScriptExecutorFactory 创建脚本执行器
type ScriptExecutorFactory struct {
	cfg *config.AgentConfig
}

// NewScriptExecutorFactory 创建脚本执行器工厂
func NewScriptExecutorFactory(cfg *config.AgentConfig) *ScriptExecutorFactory {
	return &ScriptExecutorFactory{cfg: cfg}
}

// CreateExecutor 创建执行器
func (f *ScriptExecutorFactory) CreateExecutor() (ScriptExecutor, error) {
	switch f.cfg.ScriptExecutor.Type {
	case "grpc":
		return NewPythonGRPCExecutor(
			f.cfg.ScriptExecutor.GRPC.Address,
			f.cfg.ScriptExecutor.GRPC.Timeout,
		)
	case "starlark", "":
		return &StarlarkExecutorAdapter{}, nil
	default:
		return nil, fmt.Errorf("unknown script executor type: %s", f.cfg.ScriptExecutor.Type)
	}
}

// StarlarkExecutorAdapter Starlark 执行器适配器（保持兼容）
type StarlarkExecutorAdapter struct {
	env starlark.StringDict
}

// NewStarlarkExecutorAdapter 创建 Starlark 执行器适配器
func NewStarlarkExecutorAdapter(env starlark.StringDict) *StarlarkExecutorAdapter {
	return &StarlarkExecutorAdapter{env: env}
}

// Execute 执行 Starlark 代码
func (a *StarlarkExecutorAdapter) Execute(code string, context map[string]interface{}) (interface{}, error) {
	// 将 context 转换为 Starlark 环境
	env := starlark.StringDict{}
	for k, v := range context {
		sv, err := goToStarlarkValue(v)
		if err != nil {
			return nil, fmt.Errorf("convert context key %s: %w", k, err)
		}
		env[k] = sv
	}

	// 添加原始环境
	for k, v := range a.env {
		env[k] = v
	}

	executor := NewExecutor(env)
	return executor.Execute(code)
}

// SetEnv 设置 Starlark 环境
func (a *StarlarkExecutorAdapter) SetEnv(env starlark.StringDict) {
	a.env = env
}

// ============ 工具提供者工厂 ============

// NewContextToolsProviderWithExecutor 创建带执行器的工具提供者
func NewContextToolsProviderWithExecutor(
	agentCtx interface{},
	executor ScriptExecutor,
) *ContextToolsProvider {
	// 这里可以扩展以支持不同的执行器类型
	// 目前仍使用原有的实现
	ctx, ok := agentCtx.(*context.AgentContext)
	if !ok {
		panic("agentContext must be *context.AgentContext")
	}
	return NewContextToolsProvider(ctx)
}

// WaitForPythonExecutor 等待 Python 执行器就绪
func WaitForPythonExecutor(address string, timeout time.Duration) error {
	executor, err := NewPythonGRPCExecutor(address, timeout)
	if err != nil {
		return fmt.Errorf("create python executor: %w", err)
	}
	defer executor.Close()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		healthy, err := executor.Health()
		if err == nil && healthy {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("python executor not ready after %v", timeout)
}
