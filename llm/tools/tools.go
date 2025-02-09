package llm

import (
	"encoding/json"
	"reflect"
	"runtime"

	"github.com/tmc/langchaingo/llms"
)

type ToolHandler func(json.RawMessage) (string, error)

var toolRegistry = make(map[string]ToolHandler)

// 自动注册器
func RegisterTool(name string, handler ToolHandler) {
	toolRegistry[name] = handler
}

func GetTool(name string) ToolHandler {

	return toolRegistry[name]
}

// 通过函数指针自动获取名称（需要约定命名规则）
func AutoRegister(handler ToolHandler) {
	name := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
	RegisterTool(name, handler)
}

var Tools = []llms.Tool{
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "GetCurrentTime",
			Description: "获取当前时间",
		},
	},
}
