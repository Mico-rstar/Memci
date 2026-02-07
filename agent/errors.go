package agent

import (
	"fmt"
)

// AgentResult represents the final result of an agent run
type AgentResult struct {
	FinalMessage   string        // 最终响应内容
	LastToolResult *ToolResult   // 最后一个工具调用的结果
	Metrics        *Metrics      // 执行指标
	Iterations     int           // 迭代次数
	Success        bool          // 是否成功
	Error          error         // 错误信息
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Target string        // 工具调用目标描述
	Result interface{}   // 工具执行返回值
	Success bool         // 是否执行成功
	Error   error        // 错误信息
}

// Metrics holds agent execution metrics
type Metrics struct {
	TotalIterations     int  // 总迭代次数
	TotalToolCalls      int  // 总工具调用次数
	SuccessfulToolCalls int  // 成功的工具调用次数
	FailedToolCalls     int  // 失败的工具调用次数
	TotalTokensUsed     int  // 总使用的 Token 数量
}

// MaxIterationsError is returned when the agent exceeds max iterations
type MaxIterationsError struct {
	Iterations int
	Message    string
}

func (e *MaxIterationsError) Error() string {
	return fmt.Sprintf("%s (iterations: %d)", e.Message, e.Iterations)
}

// AgentError is a general agent error
type AgentError struct {
	Phase   string // "llm", "tool", "context", "parser"
	Err     error
	Message string
}

func (e *AgentError) Error() string {
	return fmt.Sprintf("agent error in phase '%s': %s", e.Phase, e.Message)
}

func (e *AgentError) Unwrap() error {
	return e.Err
}
