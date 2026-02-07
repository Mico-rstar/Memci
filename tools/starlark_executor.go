package tools

import (
	"fmt"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// Executor Starlark 代码执行器
type Executor struct {
	thread *starlark.Thread
	env    starlark.StringDict
}

// NewExecutor 创建新的 Starlark 执行器
func NewExecutor(env starlark.StringDict) *Executor {
	return &Executor{
		thread: &starlark.Thread{
			Name: "tool-execution",
		},
		env: env,
	}
}

// Execute 执行 Starlark 代码并返回 __result__ 的值
func (e *Executor) Execute(code string) (interface{}, error) {
	// 1. 执行代码
	resultEnv, err := starlark.ExecFileOptions(syntax.LegacyFileOptions(), e.thread, "", code, e.env)
	if err != nil {
		return nil, fmt.Errorf("starlark execution failed: %w", err)
	}

	// 2. 提取 __result__
	resultValue, ok := resultEnv["__result__"]
	if !ok {
		// 没有设置 __result__，返回 nil
		return nil, nil
	}

	// 3. 转换为 Go 类型
	return starlarkValueToGo(resultValue), nil
}

// ExecuteWithEnv 执行 Starlark 代码并返回完整的环境
func (e *Executor) ExecuteWithEnv(code string) (starlark.StringDict, error) {
	return starlark.ExecFileOptions(syntax.LegacyFileOptions(), e.thread, "", code, e.env)
}

// ============ 类型转换函数 ============

// starlarkValueToGo 将 Starlark 值转换为 Go 类型
func starlarkValueToGo(v starlark.Value) interface{} {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case starlark.String:
		return string(val)
	case starlark.Int:
		i, ok := val.Int64()
		if !ok {
			// 如果超出 int64 范围，返回字符串表示
			return val.String()
		}
		return i
	case starlark.Float:
		return float64(val)
	case starlark.Bool:
		return bool(val)
	case *starlark.Dict:
		return dictToMap(val)
	case *starlark.List:
		return listToSlice(val)
	case starlark.Tuple:
		return tupleToSlice(val)
	case starlark.NoneType:
		return nil
	default:
		// 其他类型返回字符串表示
		return v.String()
	}
}

// dictToMap 将 Starlark Dict 转换为 Go map[string]interface{}
func dictToMap(d *starlark.Dict) map[string]interface{} {
	if d == nil {
		return nil
	}
	result := make(map[string]interface{}, d.Len())
	for _, item := range d.Items() {
		// 键必须是字符串类型
		var key string
		if k, ok := item[0].(starlark.String); ok {
			key = k.GoString()
		} else {
			// 非字符串键，转为字符串
			key = item[0].String()
		}
		result[key] = starlarkValueToGo(item[1])
	}
	return result
}

// listToSlice 将 Starlark List 转换为 Go []interface{}
func listToSlice(l *starlark.List) []interface{} {
	if l == nil {
		return nil
	}
	result := make([]interface{}, l.Len())
	for i := 0; i < l.Len(); i++ {
		result[i] = starlarkValueToGo(l.Index(i))
	}
	return result
}

// tupleToSlice 将 Starlark Tuple 转换为 Go []interface{}
func tupleToSlice(t starlark.Tuple) []interface{} {
	result := make([]interface{}, t.Len())
	for i := 0; i < t.Len(); i++ {
		result[i] = starlarkValueToGo(t.Index(i))
	}
	return result
}

// ============ Go 到 Starlark 的类型转换 ============

// goToStarlarkValue 将 Go 类型转换为 Starlark 值
func goToStarlarkValue(v interface{}) (starlark.Value, error) {
	if v == nil {
		return starlark.None, nil
	}

	switch val := v.(type) {
	case string:
		return starlark.String(val), nil
	case int:
		return starlark.MakeInt(val), nil
	case int64:
		return starlark.MakeInt64(val), nil
	case float64:
		return starlark.Float(val), nil
	case bool:
		return starlark.Bool(val), nil
	case map[string]interface{}:
		return mapToDict(val)
	case []interface{}:
		return sliceToList(val)
	case []string:
		return stringSliceToList(val)
	default:
		return nil, fmt.Errorf("unsupported type: %T", v)
	}
}

// mapToDict 将 Go map[string]interface{} 转换为 Starlark Dict
func mapToDict(m map[string]interface{}) (*starlark.Dict, error) {
	dict := starlark.NewDict(len(m))
	for key, val := range m {
		sv, err := goToStarlarkValue(val)
		if err != nil {
			return nil, fmt.Errorf("convert key %s: %w", key, err)
		}
		if err := dict.SetKey(starlark.String(key), sv); err != nil {
			return nil, fmt.Errorf("set key %s: %w", key, err)
		}
	}
	return dict, nil
}

// sliceToList 将 Go []interface{} 转换为 Starlark List
func sliceToList(s []interface{}) (*starlark.List, error) {
	elements := make([]starlark.Value, len(s))
	for i, val := range s {
		sv, err := goToStarlarkValue(val)
		if err != nil {
			return nil, fmt.Errorf("convert index %d: %w", i, err)
		}
		elements[i] = sv
	}
	return starlark.NewList(elements), nil
}

// stringSliceToList 将 Go []string 转换为 Starlark List
func stringSliceToList(s []string) (*starlark.List, error) {
	elements := make([]starlark.Value, len(s))
	for i, val := range s {
		elements[i] = starlark.String(val)
	}
	return starlark.NewList(elements), nil
}
