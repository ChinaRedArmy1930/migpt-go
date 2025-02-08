package llm

import (
	"bytes"
	"context"
	"html/template"
	"migpt-go/config"
	"migpt-go/doc"
	"migpt-go/internal/common"
	internal "migpt-go/internal/log"
	"os"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type LangchainProvider struct {
	client *openai.LLM
	model  string

	system_prompts string
	unimplementedLLMProvider
}

// Init implements LLMProvider.
func (l *LangchainProvider) Init(ctx context.Context) error {
	// 加载配置
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		panic(err)
	}

	// 调用自然语言处理模块
	t, err := template.New("标准化提示词模板").Parse(doc.DefaultSystemTemplate)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	data := map[string]string{}
	err = t.Execute(&buf, data)
	if err != nil {
		internal.GetLogger().Warnf(ctx, "build prompt failed => %s", err)
		return err
	}

	llm, err := openai.New(openai.WithBaseURL(cfg.LLM.BaseUrl),
		openai.WithModel(cfg.LLM.Model),
		openai.WithToken(os.Getenv("apikey")))
	if err != nil {
		internal.GetLogger().Warnf(ctx, "new openai failed => %s", err)
		return err
	}

	l.client = llm
	l.system_prompts = buf.String()
	l.model = cfg.LLM.Model

	return nil
}

// Close implements LLMProvider.
func (l *LangchainProvider) Close() error {
	return nil
}

// Generate implements LLMProvider.
func (l *LangchainProvider) Generate(ctx context.Context, prompt string, options ...Option) (string, error) {
	panic("unimplemented")
}

// StreamGenerate implements LLMProvider.
func (l *LangchainProvider) StreamGenerate(ctx context.Context, prompt string, output chan<- common.Answer, over func(t common.Answer) bool, options ...Option) error {
	content := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, l.system_prompts),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	_, err := l.client.GenerateContent(ctx, content, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		if len(chunk) == 0 {
			return nil
		}
		b := strings.Builder{}
		b.Write(chunk)
		output <- common.Answer{
			Chunk: b,
			Over:  false,
		}
		return nil
	}))
	if err != nil {
		internal.GetLogger().Debugf(ctx, "generate content failed:%s", err)
		return err
	}

	output <- common.Answer{Over: true}

	return nil
}

func NewLLM() LLMProvider[common.Answer] {
	return &LangchainProvider{}
}
