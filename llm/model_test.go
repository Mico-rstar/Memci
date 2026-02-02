package llm

import (
	"fmt"
	"memci/config"
	"memci/logger"
	"memci/message"
	"memci/prompts"
	"memci/test"
	"memci/tools"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcess(t *testing.T) {
	cfg := config.LoadConfig("../config/config.toml")
	lg := logger.NewNoOpLogger()

	msgs := message.NewMessageList()
	msgs.AddMessage(message.User, "你好")

	model := NewModel(cfg, lg, ModelQwenFlash, tools.ToolList{})
	rsp, err := model.Process(*msgs)
	require.NoError(t, err)
	fmt.Println(rsp)

	require.NotNil(t, rsp.Content)
	require.Equal(t, message.Assistant, rsp.Role)

	
}

func TestToolCall(t *testing.T) {
	cfg := config.LoadConfig("../config/config.toml")
	lg := logger.NewNoOpLogger()

	msgs := message.
		NewMessageList().
		AddMessage(message.User, "我在东莞，现在的天气怎么样")

	model := NewModel(cfg, lg, ModelQwenFlash, tools.ToolList{
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
	rsp, err := model.Process(*msgs)
	require.NoError(t, err)
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
	model := NewModel(cfg, lg, ModelQwenPlus, tools.ToolList{
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
	rsp, err := model.Process(*msgs)
	require.NoError(t, err)

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

	model := NewModel(cfg, lg, ModelQwenMax, tools.ToolList{})
	rsp, err := model.Process(*msgs)
	require.NoError(t, err)

	fmt.Println(rsp.Content)
	require.NotNil(t, rsp)
}

// TestATTPProtocol 测试ATTP协议
func TestATTPProtocol(t *testing.T) {
	cfg := config.LoadConfig("../config/config.toml")
	lg := logger.NewNoOpLogger()

	// 构建ATTP系统提示词
	systemPrompt := prompts.BuildATTPSystemPrompt("")

	// 测试单个工具调用
	t.Run("SingleTool_GetCurrentDate", func(t *testing.T) {
		msgs := message.
			NewMessageList().
			AddMessage(message.System, systemPrompt).
			AddMessage(message.User, "今天是几号？")

		model := NewModel(cfg, lg, ModelQwenPlus, tools.ToolList{})
		rsp, err := model.Process(*msgs)
		require.NoError(t, err)

		require.NotNil(t, rsp)
		fmt.Println("=== Response ===")
		fmt.Println(rsp.Content)

		// 验证响应包含必要的TOML结构
		require.Contains(t, rsp.Content, "thought", "响应应包含thought字段")
		require.Contains(t, rsp.Content, "tool_verification", "响应应包含tool_verification字段")
		require.Contains(t, rsp.Content, "tool_call", "响应应包含tool_call字段")
	})

	// 测试多工具调用
	t.Run("MultiTool_GetFutureWeather", func(t *testing.T) {
		msgs := message.
			NewMessageList().
			AddMessage(message.System, systemPrompt).
			AddMessage(message.User, "广州后天的天气怎么样？")

		model := NewModel(cfg, lg, ModelQwenPlus, tools.ToolList{})
		rsp, err := model.Process(*msgs)
		require.NoError(t, err)

		require.NotNil(t, rsp)
		fmt.Println("=== Response ===")
		fmt.Println(rsp.Content)

		// 验证响应同时使用了get_current_date和get_weather
		require.Contains(t, rsp.Content, "get_current_date", "应调用get_current_date获取当前日期")
		require.Contains(t, rsp.Content, "get_weather", "应调用get_weather查询天气")
	})

	// 测试数学计算
	t.Run("SingleTool_Calculate", func(t *testing.T) {
		msgs := message.
			NewMessageList().
			AddMessage(message.System, systemPrompt).
			AddMessage(message.User, "帮我计算 2 * (3 + 4) 的结果")

		model := NewModel(cfg, lg, ModelQwenPlus, tools.ToolList{})
		rsp, err := model.Process(*msgs)
		require.NoError(t, err)

		require.NotNil(t, rsp)
		fmt.Println("=== Response ===")
		fmt.Println(rsp.Content)

		require.Contains(t, rsp.Content, "calculate", "应调用calculate进行计算")
	})

	// 测试网络搜索
	t.Run("SingleTool_SearchWeb", func(t *testing.T) {
		msgs := message.
			NewMessageList().
			AddMessage(message.System, systemPrompt).
			AddMessage(message.User, "搜索一下Python异步编程的最新资料")

		model := NewModel(cfg, lg, ModelQwenPlus, tools.ToolList{})
		rsp, err := model.Process(*msgs)
		require.NoError(t, err)

		require.NotNil(t, rsp)
		fmt.Println("=== Response ===")
		fmt.Println(rsp.Content)

		require.Contains(t, rsp.Content, "search_web", "应调用search_web进行搜索")
	})
}

// TestATTPDataset 运行ATTP测试数据集
func TestATTPDataset(t *testing.T) {
	
	cfg := config.LoadConfig("../config/config.toml")
	lg := logger.NewNoOpLogger()
	systemPrompt := prompts.BuildATTPSystemPrompt("")

	for _, tc := range test.ATTPDataset {
		t.Run(tc.ID+"_"+tc.Category, func(t *testing.T) {
			t.Parallel() // 并行运行子测试

			// 构建消息列表
			msgs := message.NewMessageList()
			msgs.AddCachedMessage(message.System, systemPrompt)
			for _, msg := range tc.Input {
				msgs.AddMessage(msg.Role, msg.Content.String())
			}

			// 调用模型
			model := NewModel(cfg, lg, ModelQwenPlus, tools.ToolList{})
			rsp, err := model.Process(*msgs)
			require.NoError(t, err)

			require.NotNil(t, rsp)

			fmt.Printf("\n=== %s: %s ===\n", tc.ID, tc.Description)
			fmt.Printf("Category: %s\n", tc.Category)
			fmt.Printf("User: %s\n", tc.Input[0].Content.String())
			fmt.Printf("Response:\n%s\n", rsp.Content.String())

			// 运行验证函数
			if tc.Validate != nil {
				valid := tc.Validate(rsp.Content.String())
				require.True(t, valid, "响应验证失败")
			}
		})
	}
}
