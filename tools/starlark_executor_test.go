package tools

import (
	"testing"

	"go.starlark.net/starlark"
)

// TestStarlarkValueToGo 测试 Starlark 到 Go 的类型转换
func TestStarlarkValueToGo(t *testing.T) {
	tests := []struct {
		name     string
		input    starlark.Value
		expected interface{}
	}{
		{
			name:     "string",
			input:    starlark.String("hello"),
			expected: "hello",
		},
		{
			name:     "int",
			input:    starlark.MakeInt(42),
			expected: int64(42),
		},
		{
			name:     "float",
			input:    starlark.Float(3.14),
			expected: 3.14,
		},
		{
			name:     "bool true",
			input:    starlark.Bool(true),
			expected: true,
		},
		{
			name:     "bool false",
			input:    starlark.Bool(false),
			expected: false,
		},
		{
			name:     "none",
			input:    starlark.None,
			expected: nil,
		},
		{
			name: "list",
			input: starlark.NewList([]starlark.Value{
				starlark.String("a"),
				starlark.MakeInt(1),
			}),
			expected: []interface{}{"a", int64(1)},
		},
		{
			name: "dict",
			input: func() *starlark.Dict {
				d := starlark.NewDict(2)
				d.SetKey(starlark.String("key1"), starlark.String("value1"))
				d.SetKey(starlark.String("key2"), starlark.MakeInt(2))
				return d
			}(),
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": int64(2),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := starlarkValueToGo(tt.input)
			switch expected := tt.expected.(type) {
			case int64:
				if res, ok := result.(int64); !ok || res != expected {
					t.Errorf("starlarkValueToGo(%v) = %v (%T), want %v (%T)", tt.input, result, result, expected, expected)
				}
			case []interface{}:
				if res, ok := result.([]interface{}); !ok || !equalSlices(res, expected) {
					t.Errorf("starlarkValueToGo(%v) = %v (%T), want %v (%T)", tt.input, result, result, expected, expected)
				}
			case map[string]interface{}:
				if res, ok := result.(map[string]interface{}); !ok || !equalMaps(res, expected) {
					t.Errorf("starlarkValueToGo(%v) = %v (%T), want %v (%T)", tt.input, result, result, expected, expected)
				}
			default:
				if result != expected {
					t.Errorf("starlarkValueToGo(%v) = %v (%T), want %v (%T)", tt.input, result, result, expected, expected)
				}
			}
		})
	}
}

// TestGoToStarlarkValue 测试 Go 到 Starlark 的类型转换
func TestGoToStarlarkValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected starlark.Value
	}{
		{
			name:     "string",
			input:    "hello",
			expected: starlark.String("hello"),
		},
		{
			name:     "int",
			input:    42,
			expected: starlark.MakeInt(42),
		},
		{
			name:     "float",
			input:    3.14,
			expected: starlark.Float(3.14),
		},
		{
			name:     "bool",
			input:    true,
			expected: starlark.Bool(true),
		},
		{
			name:     "nil",
			input:    nil,
			expected: starlark.None,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := goToStarlarkValue(tt.input)
			if err != nil {
				t.Fatalf("goToStarlarkValue(%v) error = %v", tt.input, err)
			}
			if result.String() != tt.expected.String() {
				t.Errorf("goToStarlarkValue(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestExecutor 测试执行器
func TestExecutor(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		env      starlark.StringDict
		expected interface{}
		wantErr  bool
	}{
		{
			name: "simple string result",
			code: `__result__ = "hello world"`,
			env:  starlark.StringDict{},
			expected: "hello world",
			wantErr: false,
		},
		{
			name: "simple int result",
			code: `__result__ = 42`,
			env:  starlark.StringDict{},
			expected: int64(42),
			wantErr: false,
		},
		{
			name: "function call",
			code: `
def add(a, b):
    return a + b

__result__ = add(1, 2)
`,
			env:      starlark.StringDict{},
			expected: int64(3),
			wantErr:  false,
		},
		{
			name: "list result",
			code: `__result__ = [1, 2, 3]`,
			env:  starlark.StringDict{},
			expected: []interface{}{int64(1), int64(2), int64(3)},
			wantErr: false,
		},
		{
			name: "dict result",
			code: `__result__ = {"key": "value", "num": 42}`,
			env:  starlark.StringDict{},
			expected: map[string]interface{}{
				"key": "value",
				"num": int64(42),
			},
			wantErr: false,
		},
		{
			name: "no result",
			code: `x = 42`,
			env:  starlark.StringDict{},
			expected: nil,
			wantErr: false,
		},
		{
			name: "syntax error",
			code: `__result__ = `,
			env:  starlark.StringDict{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := NewExecutor(tt.env)
			result, err := exec.Execute(tt.code)

			if (err != nil) != tt.wantErr {
				t.Errorf("Executor.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				switch expected := tt.expected.(type) {
				case int64:
					if res, ok := result.(int64); !ok || res != expected {
						t.Errorf("Executor.Execute() = %v (%T), want %v (%T)", result, result, expected, expected)
					}
				case []interface{}:
					if res, ok := result.([]interface{}); !ok || !equalSlices(res, expected) {
						t.Errorf("Executor.Execute() = %v (%T), want %v (%T)", result, result, expected, expected)
					}
				case map[string]interface{}:
					if res, ok := result.(map[string]interface{}); !ok || !equalMaps(res, expected) {
						t.Errorf("Executor.Execute() = %v (%T), want %v (%T)", result, result, expected, expected)
					}
				default:
					if result != expected {
						t.Errorf("Executor.Execute() = %v (%T), want %v (%T)", result, result, expected, expected)
					}
				}
			}
		})
	}
}

// 辅助函数

func equalSlices(a, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalMaps(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func TestExecCode(t *testing.T) {
	code := `
# 在 usr-1 下创建新的详情页记录用户的要求
new_index = create_detail_page(
    name="natural_tone",
    description="用户要求使用自然的语气交流",
    detail="用户要求以后用自然的语气和他交流。",
    parent_index="usr-1"
)

__result__ = "已创建笔记页面"
	`

	exec := NewExecutor(starlark.StringDict{})
	result, err := exec.Execute(code)
	if err != nil {
		t.Errorf("Executor.Execute() error = %v", err)
	}
	if result != "已创建笔记页面 usr-1-natural_tone" {
		t.Errorf("Executor.Execute() = %v, want '已创建笔记页面 usr-1-natural_tone'", result)
	}
}

func TestExecCode1(t *testing.T) {
	code := `
# 在 usr-1 下创建新的详情页记录用户的要求
print("usr-1-natural_tone")
	`

	exec := NewExecutor(starlark.StringDict{})
	_, err := exec.Execute(code)
	if err != nil {
		t.Errorf("Executor.Execute() error = %v", err)
	}

}
