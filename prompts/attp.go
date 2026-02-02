package prompts

import (
	_ "embed"
	"strings"
)

//go:embed attp_sys_prompt.md
var attpSystemPromptFile string

// ATTP系统提示词模板（从文件加载）
var ATTP_SYSTEM_PROMPT_TEMPLATE = attpSystemPromptFile

// Mock工具列表
const MOCK_TOOLS_DESCRIPTION string = `
# ===== 可用工具列表 =====

def get_current_date() -> str:
    """
    获取当前日期。

    Returns:
        str: 当前日期，格式为 "YYYY-MM-DD"，例如 "2026-02-02"

    Example:
        >>> today = get_current_date()
        >>> print(today)
        '2026-02-02'
    """
    pass


def get_weather(city: str, date: str) -> dict:
    """
    获取指定城市在指定日期的天气预报。

    Args:
        city: 城市名称，如 "北京"、"上海"、"广州"
        date: 日期字符串，格式为 "YYYY-MM-DD"

    Returns:
        dict: 包含天气信息的字典，包括以下字段：
            - temperature: int - 温度（摄氏度）
            - condition: str - 天气状况（如"晴"、"多云"、"雨"）
            - wind: str - 风力描述
            - humidity: int - 湿度百分比

    Example:
        >>> weather = get_weather("广州", "2026-02-04")
        >>> print(weather["temperature"])
        25
        >>> print(weather["condition"])
        '多云'
    """
    pass


def calculate(expression: str) -> float:
    """
    计算数学表达式的值。

    Args:
        expression: 数学表达式字符串，支持加减乘除和括号

    Returns:
        float: 表达式的计算结果

    Example:
        >>> result = calculate("2 * (3 + 4)")
        >>> print(result)
        14.0
    """
    pass


def search_web(query: str, max_results: int = 5) -> list[dict]:
    """
    在网络上搜索信息。

    Args:
        query: 搜索查询字符串
        max_results: 返回结果的最大数量，默认为5

    Returns:
        list[dict]: 搜索结果列表，每个结果包含以下字段：
            - title: str - 标题
            - url: str - 链接地址
            - snippet: str - 内容摘要

    Example:
        >>> results = search_web("Python异步编程", max_results=3)
        >>> print(results[0]["title"])
        '深入理解Python异步编程'
    """
    pass
`

// BuildATTPSystemPrompt 构建完整的ATTP系统提示词
func BuildATTPSystemPrompt(toolsDescription string) string {
	if toolsDescription == "" {
		toolsDescription = MOCK_TOOLS_DESCRIPTION
	}
	return strings.Replace(ATTP_SYSTEM_PROMPT_TEMPLATE, "{{AVAILABLE_TOOLS}}", toolsDescription, 1)
}
