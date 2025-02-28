package main

import (
	"context"
	"fmt"
	"log"
	"migpt-go/config"
	"migpt-go/internal/common"
	internal "migpt-go/internal/log"
	"migpt-go/llm"
	"migpt-go/mi/fsm"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	_ "migpt-go/llm/tools/gen"
	"migpt-go/llm/vector_stores"
	qdrant_store "migpt-go/llm/vector_stores/qdrant"
	_ "net/http/pprof"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

func init() {
	log.SetFlags(log.Llongfile | log.Default().Flags())
}

func main() {
	go func() {
		http.ListenAndServe("0.0.0.0:6060", nil) // 启动 pprof
	}()
	ctx := context.TODO()

	//load所有数据到数据库
	for _, v := range []vector_stores.VectorStore{&qdrant_store.KnowledgeHub{}, &qdrant_store.WakeUp{}} {
		if err := v.Load(); err != nil {
			log.Fatal(fmt.Errorf("load %s error => %s", v.CollectionName(), err))
		}
	}

	question := make(chan string)
	answer := make(chan common.Answer)
	xiaoai := fsm.NewXiaoAi(question, answer)
	go xiaoai.FSM.Event(ctx, fsm.EventWakeUp) // 唤醒

	go func() {
		for q := range question {
			//此处决策，是获取知识库还是走api
			internal.GetLogger().Infof(ctx, "get question %s", q)
			if false {
				internal.GetLogger().Infof(ctx, "ask ai")
				AskAi(q, answer)
			} else {
				internal.GetLogger().Infof(ctx, "search dochub")
				AskDocHub(q, answer)
			}
		}
	}()

	select {}
}

func AskAi(q string, answer chan<- common.Answer) {
	lc, err := llm.NewLLM()
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*100)
	defer cancel()

	_, err = lc.StreamGenerate(ctx, q, answer)
	if err != nil {
		panic(err)
	}

	_, err = lc.Generate(ctx, q)
	if err != nil {
		panic(err)
	}
}

func AskDocHub(q string, answer chan<- common.Answer) {
	llm, err := openai.New(
		openai.WithBaseURL(config.DefaultConfig.LLM.BaseUrl),
		openai.WithToken(os.Getenv("apikey")),
		openai.WithEmbeddingModel(config.DefaultConfig.LLM.EmbeddingModel),
		openai.WithModel(config.DefaultConfig.LLM.),
	)
	if err != nil {
		log.Fatal(err)
	}

	u, err := url.Parse(config.DefaultConfig.Qdrant.Http)
	if err != nil {
		log.Fatal(err)
	}

	embed, err := embeddings.NewEmbedder(llm)
	if err != nil {
		log.Fatal(err)
	}

	qd, err := qdrant.New(
		qdrant.WithURL(*u),
		qdrant.WithCollectionName((&qdrant_store.KnowledgeHub{}).CollectionName()),
		qdrant.WithEmbedder(embed),
	)
	if err != nil {
		log.Fatal(err)
	}

	retrieverQaChain := chains.NewConversationalRetrievalQA(
		chains.LoadStuffQA(llm),
		chains.LoadCondenseQuestionGenerator(llm),
		vectorstores.ToRetriever(qd, 1),
		memory.NewConversationBuffer())

	result, err := chains.Run(context.TODO(), retrieverQaChain, q)
	if err != nil {
		log.Fatal(err)
	}

	internal.GetLogger().Infof(context.TODO(), "get dochub ans %s", result)
	
	sb := strings.Builder{}
	sb.Write([]byte(result))
	answer <- common.Answer{
		Chunk: sb,
		Over:  true,
	}
}
