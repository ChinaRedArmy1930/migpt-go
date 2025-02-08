package llm

import (
	"time"

	"github.com/tmc/langchaingo/llms"
)

func getCurrentTime() string {
	return time.Now().Local().Format(time.DateTime)
}

var tools = []llms.Tool{
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "getCurrentTime",
			Description: "获取当前时间",
		},
	},
}
