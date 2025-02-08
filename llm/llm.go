package llm

import (
	"context"
	"time"
)

// Option 动态配置（超时、温度等参数）
type Option func(*Options)

// LLMProvider 核心接口，定义通用的 LLM 操作
type LLMProvider[T any] interface {
	// 初始化
	Init(ctx context.Context) error
	// 生成文本
	Generate(ctx context.Context, prompt string, options ...Option) (string, error)
	// 流式生成（可选）
	StreamGenerate(ctx context.Context, prompt string, output chan<- T, over func(t T) bool, options ...Option) error
	// 模型元数据（名称、版本等）
	ModelInfo() ModelMeta
	// 关闭资源（如果有）
	Close() error
}

// 模型元数据
type ModelMeta struct {
	Name    string
	Version string
	Vendor  string
}

type Options struct {
	Temperature float32
	MaxTokens   int
	Timeout     time.Duration
	Stream      bool
	// 扩展字段 ...
}

func (l *unimplementedLLMProvider) Init(ctx context.Context) error { return nil }
func (l *unimplementedLLMProvider) Generate(ctx context.Context, prompt string, options ...Option) (string, error) {
	return "", nil
}
func (l *unimplementedLLMProvider) StreamGenerate(ctx context.Context, prompt string, output chan<- any, over func(t any) bool, options ...Option) error {
	return nil
}
func (l *unimplementedLLMProvider) ModelInfo() ModelMeta { return ModelMeta{} }
func (l *unimplementedLLMProvider) Close() error         { return nil }

// 接口的默认实现
type unimplementedLLMProvider struct{}

var _ LLMProvider[any] = (*unimplementedLLMProvider)(nil)
