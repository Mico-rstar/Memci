package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// formatToolResult formats a tool execution result for inclusion in context
func formatToolResult(result *ToolResult) string {
	var builder strings.Builder

	builder.WriteString("## Tool Execution Result\n\n")
	builder.WriteString(fmt.Sprintf("**Target**: %s\n\n", result.Target))

	if result.Success {
		builder.WriteString("**Status**: Success\n\n")
		builder.WriteString("**Result**:\n\n")

		// Format result based on type
		switch v := result.Result.(type) {
		case string:
			builder.WriteString(v)
		case map[string]interface{}:
			jsonBytes, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				builder.WriteString(fmt.Sprintf("%v", v))
			} else {
				builder.WriteString(string(jsonBytes))
			}
		case []interface{}:
			jsonBytes, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				builder.WriteString(fmt.Sprintf("%v", v))
			} else {
				builder.WriteString(string(jsonBytes))
			}
		default:
			builder.WriteString(fmt.Sprintf("%v", v))
		}
	} else {
		builder.WriteString("**Status**: Failed\n\n")
		if result.Error != nil {
			builder.WriteString(fmt.Sprintf("**Error**: %v\n", result.Error))
		}
	}

	return builder.String()
}

// formatFinalResponse formats the final agent response for output
func formatFinalResponse(content string, iterations int, metrics *Metrics) string {
	var builder strings.Builder

	builder.WriteString(content)
	builder.WriteString("\n\n---\n\n")
	builder.WriteString(fmt.Sprintf("**Iterations**: %d\n", iterations))
	builder.WriteString(fmt.Sprintf("**Tool Calls**: %d (%d success, %d failed)\n",
		metrics.TotalToolCalls,
		metrics.SuccessfulToolCalls,
		metrics.FailedToolCalls))

	return builder.String()
}
