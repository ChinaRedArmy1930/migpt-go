package main

import (
	"bytes"
	"context"
	_ "embed"
	"migpt-go/doc"
	"migpt-go/internal/common"
	internal "migpt-go/internal/log"
	"migpt-go/mi/fsm"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"text/template"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	go func() {
		http.ListenAndServe("0.0.0.0:6060", nil) // 启动 pprof
	}()
	ctx := context.TODO()

	question := make(chan string)
	answer := make(chan common.Answer)
	xiaoai := fsm.NewXiaoAi(question, answer)
	go xiaoai.FSM.Event(ctx, fsm.EventWakeUp) // 唤醒

	go func() {
		callai(ctx, question, answer)
	}()

	select {}
}

func callai(ctx context.Context, q <-chan string, a chan<- common.Answer) error {
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
	llm, err := openai.New(openai.WithBaseURL("https://api.siliconflow.cn/v1"),
		openai.WithModel("deepseek-ai/DeepSeek-V3"),
		openai.WithToken(os.Getenv("apikey")))
	if err != nil {
		internal.GetLogger().Warnf(ctx, "new openai failed => %s", err)
		return err
	}

	for {
		question := <-q
		internal.GetLogger().Debugf(ctx, "get question :%s", question)
		content := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, buf.String()),
			llms.TextParts(llms.ChatMessageTypeHuman, question),
		}

		_, err := llm.GenerateContent(ctx, content, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			if len(chunk) == 0 {
				return nil
			}
			b := strings.Builder{}
			b.Write(chunk)
			a <- common.Answer{
				Chunk: b,
				Over:  false,
			}
			return nil
		}))
		if err != nil {
			internal.GetLogger().Debugf(ctx, "generate content failed:%s", err)
			return err
		}

		a <- common.Answer{Over: true}

	}

	return nil
}
