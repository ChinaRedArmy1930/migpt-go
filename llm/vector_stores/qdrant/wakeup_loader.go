package qdrant_store

import (
	"context"
	"log"
	"migpt-go/config"
	internal "migpt-go/internal/log"
	"migpt-go/llm/vector_stores"
	"net/url"
	"os"
	"strconv"

	"github.com/qdrant/go-client/qdrant"
	"github.com/tmc/langchaingo/llms/openai"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
)

const (
	QdrantWeakUpConnection = "wakeupConnection"
)

var _ vector_stores.VectorStore = (*WakeUp)(nil)

type WakeUp struct{}

// SimilaritySearch implements vector_stores.VectorStore.
func (w *WakeUp) SimilaritySearch(msg string) {
	panic("unimplemented")
}

// CollectionName implements vector_stores.VectorStore.
func (w *WakeUp) CollectionName() string {
	return QdrantWeakUpConnection
}

// Load implements vector_stores.VectorStore.
func (w *WakeUp) Load() error {
	ctx := context.TODO()
	u, err := url.Parse(config.DefaultConfig.Qdrant.Grpc)
	if err != nil {
		return err
	}

	port, _ := strconv.Atoi(u.Port())
	//sudo docker run -d  -p 6334:6334   qdrant/qdrant
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:                   u.Hostname(),
		Port:                   port,
		SkipCompatibilityCheck: true,
		GrpcOptions: []grpc.DialOption{grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(1024),
			grpc.UseCompressor(gzip.Name),
		)},
	})

	if err != nil {
		log.Fatal(err)
	}

	collections, err := client.ListCollections(ctx)
	if err != nil {
		log.Fatal(err)
	}

	internal.GetLogger().Infof(ctx, "collection %s", collections)

	exist := false

	for _, v := range collections {
		if v == QdrantWeakUpConnection {
			exist = true
			break
		}
	}

	if !exist {
		err = client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: QdrantWeakUpConnection,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     1024,
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	//	使用 embeddings 将文本计算成向量
	opts := []openai.Option{
		openai.WithBaseURL(config.DefaultConfig.LLM.BaseUrl),
		openai.WithToken(os.Getenv("apikey")),
		openai.WithEmbeddingModel(config.DefaultConfig.LLM.EmbeddingModel),
	}

	llm, err := openai.New(opts...)
	if err != nil {
		log.Fatal(err)
	}

	embedings, err := llm.CreateEmbedding(ctx, config.DefaultConfig.Ai.WakeUpKeyWords)
	if err != nil {
		log.Fatal(err)
	}

	points := make([]*qdrant.PointStruct, 0)

	for k, v := range embedings {
		points = append(points, &qdrant.PointStruct{
			Id:      qdrant.NewIDNum(uint64(k + 1)),
			Vectors: qdrant.NewVectors(v...),
		})
	}

	_, err = client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: QdrantWeakUpConnection,
		Wait:           new(bool),
		Points:         points,
	})

	return err
}
