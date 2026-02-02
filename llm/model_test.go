package llm

import (
	"fmt"
	"memci/config"
	"memci/logger"
	"memci/message"
	"memci/tools"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcess(t *testing.T) {
	cfg := config.LoadConfig("../config/config.toml")
	lg := logger.NewNoOpLogger()

	msgs := message.NewMessageList()
	msgs.AddMessage(message.User, "你好")

	model := NewModel(*cfg, lg, ModelQwenFlash, tools.ToolList{})
	rsp := model.Process(*msgs)

	require.NotNil(t, rsp.Content)
	require.Equal(t, message.Assistant, rsp.Role)

	fmt.Println(rsp)
	
}

func TestToolCall(t *testing.T) {
	cfg := config.LoadConfig("../config/config.toml")
	lg := logger.NewNoOpLogger()

	msgs := message.
		NewMessageList().
		AddMessage(message.User, "我在东莞，现在的天气怎么样")

	model := NewModel(*cfg, lg, ModelQwenFlash, tools.ToolList{
		Tools: []tools.FunctionTool{
			{
				Name:        "get_weather",
				Description: "获取某地天气",
				Strict:      true,
				Parameters: tools.
					NewParameterBuilder().
					AddField(
						tools.Field{
							Name:        "city",
							Description: "要获取天气信息的城市",
							Required:    true,
							Type:        "string",
						},
					).
					Build(),
			},
		},
	})
	rsp := model.Process(*msgs)
	require.NotNil(t, rsp)
	require.Equal(t, "get_weather", rsp.ToolCalls[0].Function.Name)
	fmt.Println(rsp.ToolCalls[0].Function.Arguments)
}

// 验证编码式工具调用
func TestToolCallCode(t *testing.T) {
	cfg := config.LoadConfig("../config/config.toml")
	lg := logger.NewNoOpLogger()

	msgs := message.
		NewMessageList().
		AddMessage(message.User, "我在广州，后天天气怎么样？")

	dct := `
你可以编写python代码调用下列工具来回答用户的问题。

**可用的工具函数：**

1. get_current_date() -> str
   获取当前日期，返回格式为 "YYYY-MM-DD" 的字符串。
   示例：today = get_current_date()  # 返回 "2026-02-02"

2. get_weather(city: str, date: str) -> dict
   获取指定城市在指定日期的天气预报。
   参数：
     - city: 城市名称，如 "北京"、"上海"、"广州"
     - date: 日期字符串，格式为 "YYYY-MM-DD"
   返回：包含天气信息的字典，包括温度、天气状况、风力等。
   示例：get_weather("广州", "2026-02-04")

**重要说明：**
- 日期格式必须是 "YYYY-MM-DD"
- 可以使用 get_current_date() 获取当前日期作为参考
- 只调用工具获取数据，不要在代码中做额外输出
`
	model := NewModel(*cfg, lg, ModelQwenPlus, tools.ToolList{
		Tools: []tools.FunctionTool{
			{
				Name:        "call_tool",
				Description: dct,
				Strict:      true,
				Parameters: tools.
					NewParameterBuilder().
					AddField(
						tools.Field{
							Name:        "code",
							Description: "调用工具的python代码",
							Required:    true,
							Type:        "string",
						},
					).
					Build(),
			},
		},
	})
	rsp := model.Process(*msgs)

	require.NotNil(t, rsp)
	require.Equal(t, "call_tool", rsp.ToolCalls[0].Function.Name)
	fmt.Println(rsp.ToolCalls[0].Function.Arguments)

}

func TestSystemCodingToolCall(t *testing.T) {
	cfg := config.LoadConfig("../config/config.toml")
	lg := logger.NewNoOpLogger()

	systemPrompt := `
你可以编写python代码调用下列工具来回答用户的问题。

**可用的工具函数：**

1. get_current_date() -> str
   获取当前日期，返回格式为 "YYYY-MM-DD" 的字符串。
   示例：today = get_current_date()  # 返回 "2026-02-02"

2. get_weather(city: str, date: str) -> dict
   获取指定城市在指定日期的天气预报。
   参数：
     - city: 城市名称，如 "北京"、"上海"、"广州"
     - date: 日期字符串，格式为 "YYYY-MM-DD"
   返回：包含天气信息的字典，包括温度、天气状况、风力等。
   示例：get_weather("广州", "2026-02-04")

**重要说明：**
- 日期格式必须是 "YYYY-MM-DD"
- 可以使用 get_current_date() 获取当前日期作为参考
- 只调用工具获取数据，不要在代码中做额外输出
`

	msgs := message.
		NewMessageList().
		AddMessage(message.System, systemPrompt).
		AddMessage(message.User, "我在广州，后天天气怎么样？")

	model := NewModel(*cfg, lg, ModelQwenMax, tools.ToolList{})
	rsp := model.Process(*msgs)

	fmt.Println(rsp.Content)
	require.NotNil(t, rsp)
}
