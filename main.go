package main

import (
	"context"
	"log"
	"migpt-go/internal/common"
	"migpt-go/llm"
	"migpt-go/mi/fsm"
	"net/http"
	"time"

	_ "migpt-go/llm/tools/gen"
	_ "net/http/pprof"
)

func init() {
	log.SetFlags(log.Llongfile | log.Default().Flags())
}

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
		lc, err := llm.NewLLM()
		if err != nil {
			panic(err)
		}
		ctx, cancel := context.WithTimeout(context.TODO(), time.Second*100)
		defer cancel()

		for q := range question {
			_, err = lc.StreamGenerate(ctx, q, answer)
			if err != nil {
				panic(err)
			}

			_, err := lc.Generate(ctx, q)
			if err != nil {
				panic(err)
			}

		}
	}()

	select {}
}
