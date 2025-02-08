package llm

import (
	"github.com/tmc/langchaingo/llms"
)

var Tools = []llms.Tool{
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "getCurrentTime",
			Description: "获取当前时间",
		},
	},
}
