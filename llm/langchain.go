package llm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"log"
	"migpt-go/config"
	"migpt-go/doc"
	"migpt-go/internal/common"
	internal "migpt-go/internal/log"
	llm "migpt-go/llm/tools"
	llmtools "migpt-go/llm/tools"
	"os"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type LangchainProvider struct {
	client *openai.LLM
	model  string
	ctx    context.Context

	system_prompts string
	unimplementedLLMProvider
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
func (l *LangchainProvider) StreamGenerate(ctx context.Context, prompt string, output chan<- common.Answer, options ...Option) (func(t common.Answer) bool, error) {
	content := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, l.system_prompts),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	over := func(a common.Answer) bool { return a.Over }

	resp, err := l.client.GenerateContent(ctx, content, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		select {
		case <-ctx.Done():
			return errors.New("timeout")
		default:
		}

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
	}), llms.WithTools(llmtools.Tools))
	if err != nil {
		internal.GetLogger().Debugf(ctx, "generate content failed:%s", err)
		return over, err
	}
	internal.GetLogger().Infof(l.ctx, "resp %#v", resp.Choices)
	b := strings.Builder{}
	if len(resp.Choices) != 0 && resp.Choices[0] != nil {
		internal.GetLogger().Infof(l.ctx, "choice %v", resp.Choices[0].FuncCall)
		fnName := resp.Choices[0].FuncCall.Name
		args := []byte(resp.Choices[0].FuncCall.Arguments)

		if handler := llm.GetTool(fnName); handler != nil {
			result, err := handler(args)
			if err != nil {
				log.Fatalf("工具调用 %s 失败: %v", fnName, err)
			}
			b.Write([]byte(result))
		} else {
			b.Write(fmt.Appendf(nil, "未知工具调用: %s", fnName))
			output <- common.Answer{Chunk: b}
		}
	}

	output <- common.Answer{Over: true, Chunk: b}

	return over, nil
}

func NewLLM() (LLMProvider[common.Answer], error) {
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

	var l LangchainProvider
	var buf bytes.Buffer
	ctx := context.TODO()
	data := map[string]string{}
	err = t.Execute(&buf, data)
	if err != nil {
		internal.GetLogger().Warnf(ctx, "build prompt failed => %s", err)
		return nil, err
	}

	llm, err := openai.New(openai.WithBaseURL(cfg.LLM.BaseUrl),
		openai.WithModel(cfg.LLM.Model),
		openai.WithToken(os.Getenv("apikey")),
		openai.WithResponseFormat(openai.ResponseFormatJSON),
	)
	if err != nil {
		internal.GetLogger().Warnf(ctx, "new openai failed => %s", err)
		return nil, err
	}

	l.client = llm
	l.system_prompts = buf.String()
	l.model = cfg.LLM.Model

	return &l, nil
}
