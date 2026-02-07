package util

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// ReAct 表示推理-行动结构体
// 包含思考过程（think）和工具调用（tool_call）
type ReAct struct {
	Think    string    `json:"think"`    // 思考过程，位于 TOML 之前的内容
	ToolCall *ToolCall `json:"tool_call"` // 工具调用，为 nil 表示无需工具调用
}

// ToolCall 表示工具调用的 TOML 结构
type ToolCall struct {
	Target string `toml:"target" json:"target"` // 工具调用的目标描述
	Code   string `toml:"code" json:"code"`     // Starlark 代码
	Status string `toml:"status,omitempty" json:"status,omitempty"`
}

// toolCallTOML TOML 解析用的包装结构
type toolCallTOML struct {
	ToolCall ToolCall `toml:"tool_call"`
}

// ParseToolCall 解析包含工具调用的字符串
// 提取 think 部分和 TOML 工具调用部分，舍去工具调用后的多余输出
func ParseToolCall(input string) (*ReAct, error) {
	// 查找 [tool_call] 的位置
	toolCallStart := strings.Index(input, "[tool_call]")
	if toolCallStart == -1 {
		// 没有找到工具调用，返回只有 think 的结构
		return &ReAct{
			Think:    strings.TrimSpace(input),
			ToolCall: nil,
		}, nil
	}

	// 提取 think 部分（[tool_call] 之前的内容）
	think := input[:toolCallStart]
	think = strings.TrimSpace(think)
	// 移除可能存在的 ```toml 后缀
	think = strings.TrimSuffix(think, "```toml")
	think = strings.TrimSuffix(think, "```")
	think = strings.TrimSpace(think)

	// 提取 TOML 内容（从 [tool_call] 开始）
	tomlContent := input[toolCallStart:]

	// 如果有 ``` 结束标记，截取到该位置
	codeBlockEnd := strings.Index(tomlContent, "```")
	if codeBlockEnd != -1 {
		tomlContent = tomlContent[:codeBlockEnd+3]
	} else {
		// 没有 ``` 结束，尝试找到 TOML 结尾
		// 策略：找到 code 字段的三引号结束位置
		tripleQuotePattern := regexp.MustCompile(`code\s*=\s*'''[\s\S]*?'''`)
		tripleQuoteMatch := tripleQuotePattern.FindStringIndex(tomlContent)
		if tripleQuoteMatch != nil && tripleQuoteMatch[1] > 0 {
			// 截取到三引号结束后的内容
			remaining := tomlContent[tripleQuoteMatch[1]:]
			// 查找下一个 ```
			nextBacktick := strings.Index(remaining, "```")
			if nextBacktick != -1 {
				tomlContent = tomlContent[:tripleQuoteMatch[1]+nextBacktick+3]
			} else {
				// 没有 ```，直接截取到 TOML 结束
				tomlContent = tomlContent[:tripleQuoteMatch[1]]
			}
		}
	}

	// 移除可能的 ```toml 前缀和 ``` 后缀
	tomlContent = strings.TrimPrefix(tomlContent, "```toml")
	tomlContent = strings.TrimPrefix(tomlContent, "```")
	tomlContent = strings.TrimSuffix(tomlContent, "```")
	tomlContent = strings.TrimSpace(tomlContent)

	// 解析 TOML
	var parsed toolCallTOML
	metadata, err := toml.Decode(tomlContent, &parsed)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	// 检查是否有未解析的字段
	if len(metadata.Undecoded()) > 0 {
		return nil, fmt.Errorf("undecoded fields in TOML: %v", metadata.Undecoded())
	}

	return &ReAct{
		Think:    think,
		ToolCall: &parsed.ToolCall,
	}, nil
}

// ParseToolCallStrict 严格模式解析，要求必须包含工具调用
func ParseToolCallStrict(input string) (*ReAct, error) {
	react, err := ParseToolCall(input)
	if err != nil {
		return nil, err
	}

	if react.ToolCall == nil {
		return nil, fmt.Errorf("no tool call found in input")
	}

	return react, nil
}

// ExtractCodeFromMarkdown 从 Markdown 代码块中提取纯代码
// 处理 ```toml ... ``` 和 ``` ... ``` 格式
func ExtractCodeFromMarkdown(input string) string {
	// 移除 ```toml 和 ``` 包裹
	pattern := regexp.MustCompile("```(?:toml)?\n?(.*?)\n?```")
	matches := pattern.FindStringSubmatch(input)

	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	return input
}

// CleanThought 清理思考内容，去除多余的空白和换行
func CleanThought(think string) string {
	// 将多个连续空白字符替换为单个空格
	pattern := regexp.MustCompile(`\s+`)
	cleaned := pattern.ReplaceAllString(think, " ")
	return strings.TrimSpace(cleaned)
}
