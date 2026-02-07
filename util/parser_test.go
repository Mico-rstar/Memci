package util

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestParseToolCall(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantThink   string
		wantTarget  string
		wantCode    string
		wantHasCall bool
		wantErr     bool
	}{
		{
			name: "标准工具调用",
			input: strings.Join([]string{
				"我应该调用create_detail_page创建一页来记录我的思考",
				"```toml",
				"[tool_call]",
				`target = "创建新的详情页"`,
				"code = '''",
				"__result__ = create_detail_page(",
				`    name="新想法",`,
				`    description="工具调用的思考",`,
										`    detail="详细内容...",`,
										`    parent_index="page:001"`,
										")",
										"'''",
										"```",
										"工具调用后的多余输出应该舍去",
									}, "\n"),
			wantThink:   "我应该调用create_detail_page创建一页来记录我的思考",
			wantTarget:  "创建新的详情页",
			wantCode: strings.Join([]string{
				"__result__ = create_detail_page(",
				`    name="新想法",`,
				`    description="工具调用的思考",`,
				`    detail="详细内容...",`,
				`    parent_index="page:001"`,
				")",
			}, "\n"),
			wantHasCall: true,
			wantErr:     false,
		},
		{
			name: "无 Markdown 包裹的工具调用",
			input: strings.Join([]string{
				"我需要查询信息",
				"",
				"[tool_call]",
				`target = "查询 Page"`,
										`code = '''__result__ = get_page("page:001")'''`,
									}, "\n"),
			wantThink:   "我需要查询信息",
			wantTarget:  "查询 Page",
			wantCode:    `__result__ = get_page("page:001")`,
			wantHasCall: true,
			wantErr:     false,
		},
		{
			name:        "无工具调用",
			input:       "这只是一个普通的消息，不包含任何工具调用",
			wantThink:   "这只是一个普通的消息，不包含任何工具调用",
			wantHasCall: false,
			wantErr:     false,
		},
		{
			name: "复杂代码块",
			input: strings.Join([]string{
				"创建多个页面",
				"",
				"[tool_call]",
				`target = "批量创建"`,
				"code = '''",
				"# 创建第一个页面",
										`page1 = create_detail_page("首页", "描述", "内容", "parent1")`,
										"",
										"# 创建第二个页面",
										`page2 = create_detail_page("次页", "描述", "内容", "parent2")`,
										"",
										"__result__ = [page1, page2]",
										"'''",
										"多余内容",
									}, "\n"),
			wantThink:  "创建多个页面",
			wantTarget: "批量创建",
			wantCode: strings.Join([]string{
				"# 创建第一个页面",
				`page1 = create_detail_page("首页", "描述", "内容", "parent1")`,
				"",
				"# 创建第二个页面",
				`page2 = create_detail_page("次页", "描述", "内容", "parent2")`,
				"",
				"__result__ = [page1, page2]",
			}, "\n"),
			wantHasCall: true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseToolCall(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseToolCall() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.Think != tt.wantThink {
					t.Errorf("ParseToolCall() Think = %q, want %q", got.Think, tt.wantThink)
				}

				hasCall := got.ToolCall != nil
				if hasCall != tt.wantHasCall {
					t.Errorf("ParseToolCall() ToolCall present = %v, want %v", hasCall, tt.wantHasCall)
				}

				if tt.wantHasCall && got.ToolCall != nil {
					if got.ToolCall.Target != tt.wantTarget {
						t.Errorf("ParseToolCall() Target = %q, want %q", got.ToolCall.Target, tt.wantTarget)
					}
					// 标准化代码块（去除首尾空行）后比较
					gotCode := normalizeCode(got.ToolCall.Code)
					wantCode := normalizeCode(tt.wantCode)
					if gotCode != wantCode {
						t.Errorf("ParseToolCall() Code =\n%q\nwant\n%q", gotCode, wantCode)
					}
				}
			}
		})
	}
}

func TestParseToolCallStrict(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "有工具调用",
			input: strings.Join([]string{
				"思考",
				"[tool_call]",
				`target = "测试"`,
				`code = "test"`,
			}, "\n"),
			wantErr: false,
		},
		{
			name:    "无工具调用",
			input:   "只是普通文本",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseToolCallStrict(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseToolCallStrict() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}


func TestExtractCodeFromMarkdown(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "toml 代码块",
			input: "```toml\ncode here\n```",
			want:  "code here",
		},
		{
			name:  "普通代码块",
			input: "```\ncode here\n```",
			want:  "code here",
		},
		{
			name:  "无代码块",
			input: "plain text",
			want:  "plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractCodeFromMarkdown(tt.input); got != tt.want {
				t.Errorf("ExtractCodeFromMarkdown() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCleanThought(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "多余空白",
			input: "  hello   world  \n\n  test  ",
			want:  "hello world test",
		},
		{
			name:  "正常文本",
			input: "hello world",
			want:  "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CleanThought(tt.input); got != tt.want {
				t.Errorf("CleanThought() = %q, want %q", got, tt.want)
			}
		})
	}
}

// 辅助函数：标准化代码块
func normalizeCode(code string) string {
	lines := strings.Split(code, "\n")
	// 去除首尾空行
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}


func TestParToolCall(t *testing.T) {
	data, err := os.ReadFile("./test/data.md")
	if err != nil {
		t.Fatal(err)
	}
	ract, err := ParseToolCall(string(data))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(ract.Think)
	fmt.Println()
	fmt.Println(ract.ToolCall.Code)
}