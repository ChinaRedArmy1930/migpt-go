package fsm

import (
	"container/heap"
	"context"
	"fmt"
	"log"
	"migpt-go/config"
	"migpt-go/internal/common"
	internal "migpt-go/internal/log"
	"migpt-go/internal/status"
	"migpt-go/mi/base/mina"
	"migpt-go/mi/base/miot"
	"os"
	"time"

	"github.com/looplab/fsm"

	"github.com/qdrant/go-client/qdrant"
	qdrant_client "github.com/qdrant/go-client/qdrant"
	"github.com/tmc/langchaingo/llms/openai"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
)

// 状态定义
const (
	StateIdle       = "idle"
	StateListening  = "listening"
	StateProcessing = "processing"
	StateSpeaking   = "speaking"
	StateSleeping   = "sleeping"
)

// 事件定义
const (
	EventWakeUp           = "wake_up"
	EventVoiceDetected    = "voice_detected"
	EventProcessComplete  = "process_complete"
	EventResponseComplete = "response_complete"
	EventTimeout          = "timeout"
)

const (
	QdrantWeakUpConnection = "weak_up_keyword_connection"
)

// XiaoAiFSM 小爱同学状态机
type XiaoAiFSM struct {
	FSM              *fsm.FSM
	ctx              context.Context
	mina             mina.IMina
	miot             miot.IMiot
	qqueue           chan<- string
	aqueue           <-chan common.Answer
	QdrantClient     *qdrant_client.Client
	VectorClient     *openai.LLM
	TimeoutStatus    *status.TimeoutStatus
	ConversationHeap mina.Records
}

func NewXiaoAi(question chan<- string, answer <-chan common.Answer) *XiaoAiFSM {
	ctx := context.TODO()
	x := &XiaoAiFSM{
		mina:             mina.InitMina(ctx),
		miot:             miot.InitMiot(ctx),
		ctx:              ctx,
		qqueue:           question,
		aqueue:           answer,
		TimeoutStatus:    status.NewTimeoutStatus(time.Second * 20),
		ConversationHeap: make([]*mina.Record, 0),
	}

	//sudo docker run -d  -p 6333:6333   qdrant/qdrant
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:                   "127.0.0.1",
		Port:                   6334,
		SkipCompatibilityCheck: true,
		GrpcOptions: []grpc.DialOption{grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(1024),
			grpc.UseCompressor(gzip.Name),
		)},
	})

	x.QdrantClient = client

	if err != nil {
		log.Fatal(err)
	}
	collections, err := x.QdrantClient.ListCollections(x.ctx)
	if err != nil {
		log.Fatal(err)
	}

	internal.GetLogger().Infof(x.ctx, "collection %s", collections)

	exist := false

	for _, v := range collections {
		if v == QdrantWeakUpConnection {
			exist = true
			break
		}
	}

	if !exist {
		err = client.CreateCollection(ctx, &qdrant_client.CreateCollection{
			CollectionName: QdrantWeakUpConnection,
			VectorsConfig: qdrant_client.NewVectorsConfig(&qdrant_client.VectorParams{
				Size:     1024,
				Distance: qdrant_client.Distance_Cosine,
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

	x.VectorClient = llm

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

	client.Upsert(context.TODO(), &qdrant.UpsertPoints{
		CollectionName: QdrantWeakUpConnection,
		Wait:           new(bool),
		Points:         points,
	})

	x.FSM = fsm.NewFSM(
		StateIdle,
		fsm.Events{
			// 格式：事件名称 -> 源状态 -> 目标状态
			{Name: EventWakeUp, Src: []string{StateIdle}, Dst: StateListening},
			{Name: EventVoiceDetected, Src: []string{StateListening}, Dst: StateProcessing},
			{Name: EventProcessComplete, Src: []string{StateProcessing}, Dst: StateSpeaking},
			{Name: EventResponseComplete, Src: []string{StateSpeaking}, Dst: StateListening},
			{Name: EventTimeout, Src: []string{StateIdle}, Dst: StateSleeping},
		},
		fsm.Callbacks{
			// 状态进入回调
			"enter_state":        func(ctx context.Context, e *fsm.Event) { x.enterState(e) },
			"leave_" + StateIdle: func(ctx context.Context, e *fsm.Event) { x.enterIdle() },
		},
	)

	return x
}

// 状态进入处理函数
func (x *XiaoAiFSM) enterState(e *fsm.Event) {
	internal.GetLogger().Debugf(x.ctx, "curr => %s, next status  => %s", x.FSM.Current(), e.Dst)
	switch e.Dst {
	case StateIdle:
		x.onEnterIdle()
	case StateListening:
		x.onEnterListening()
	case StateProcessing:
		x.onEnterProcessing()
	case StateSpeaking:
		x.onEnterSpeaking()
	case StateSleeping:
		x.onEnterSleeping()
	}
}

func (x *XiaoAiFSM) enterIdle() {
	//进入状态流转以前先准备一下数据

	//获取设备的device_id
	device_list, err := x.mina.GetMiDeviceList(x.ctx)
	if err != nil {
		log.Fatal(err)
	}

	for _, device := range device_list {
		if device.Name == "小爱音箱mini" {
			x.FSM.SetMetadata("device_id", device.DeviceID)
			x.FSM.SetMetadata("device", device)
			break
		}
	}

	go x.GetUserConversation()

	//获取设备的model
	miot_device_list, err := x.miot.GetMiotDevices(x.ctx)
	if err != nil {
		log.Fatal(err)
	}

	for _, device := range miot_device_list {
		if device.Name == "小爱音箱mini" {
			x.FSM.SetMetadata("model", device.Model)
			break
		}
	}

}

func (x *XiaoAiFSM) onEnterIdle() {
	// 启动空闲计时器（比如30秒无操作进入睡眠）
}

func (x *XiaoAiFSM) onEnterListening() {
	internal.GetLogger().Infof(x.ctx, "正在获取用户语句 ...")
	device_id, ok := x.FSM.Metadata("device_id")
	if !ok {
		//没拿到device_id
		internal.GetLogger().Errorf(x.ctx, "get device_id failed")
		return
	}

	ticker := time.NewTicker(time.Second)

	for range ticker.C {
		if x.ConversationHeap.Len() != 0 {
			break
		}
	}

	msg := heap.Pop(&x.ConversationHeap).(*mina.Record).Query
	f := func() {
		x.FSM.SetMetadata("question", msg)
		x.FSM.Event(x.ctx, EventVoiceDetected)
	}

	//通过当前是否在AI模式或者语音分析决定是否需要AI回答, 如果不流转到下个状态则就是ai回答
	if ok, _ := x.TimeoutStatus.Ok(QdrantWeakUpConnection); ok {
		//续期
		x.TimeoutStatus.Renewal(QdrantWeakUpConnection)
		f()

	} else {
		if x.AiWeakUpSimilarCheck(msg) {
			if x.TimeoutStatus.Exist(QdrantWeakUpConnection) {
				x.TimeoutStatus.Renewal(QdrantWeakUpConnection)
			} else {
				x.TimeoutStatus.Add(QdrantWeakUpConnection, func() {
					//播放退出的音效
					internal.GetLogger().Infof(x.ctx, "exit ai mode")
					x.mina.Controller(x.ctx, "play", "退出AI模式", "", device_id)
				})
				x.TimeoutStatus.Start(QdrantWeakUpConnection)
			}
			f()
		}
	}
}

func (x *XiaoAiFSM) AiWeakUpSimilarCheck(msg string) bool {
	//将数据算成向量
	embedding, err := x.VectorClient.CreateEmbedding(x.ctx, []string{msg})
	if err != nil {
		panic(err)
	}

	var limit = uint64(1)

	search_result, err := x.QdrantClient.Query(x.ctx, &qdrant.QueryPoints{
		CollectionName: QdrantWeakUpConnection,
		Query:          qdrant.NewQuery(embedding[0]...),
		Limit:          &limit,
	})

	if err != nil {
		panic(err)
	}

	internal.GetLogger().Infof(x.ctx, "result => %#v", search_result[0])
	return search_result[0].Score > 0.8
}

func (x *XiaoAiFSM) onEnterProcessing() {
	internal.GetLogger().Infof(x.ctx, "进入处理状态，分析用户请求...")
	//调用接口输出
	question, ok := x.FSM.Metadata("question")
	if !ok {
		//没拿到问题
		internal.GetLogger().Errorf(x.ctx, "get question failed")
		return
	}

	x.qqueue <- question.(string)
	ans := ""
	for answer := range x.aqueue {
		if answer.Over {
			answer.Chunk.Reset()
			break
		}

		ans += answer.Chunk.String()
	}

	x.FSM.SetMetadata("answer", ans)

	x.FSM.Event(x.ctx, EventProcessComplete)
	internal.GetLogger().Debugf(x.ctx, "get answer => %s", ans)
}

func (x *XiaoAiFSM) onEnterSpeaking() {
	internal.GetLogger().Infof(x.ctx, "进入回应状态，播放回答...  ")
	// 调用语音合成和播放模块
	d, _ := x.FSM.Metadata("device_id")
	ans, _ := x.FSM.Metadata("answer")
	x.mina.Controller(x.ctx, "play", ans.(string), "", d.(string))
	err := x.FSM.Event(x.ctx, EventResponseComplete)
	if err != nil {
		log.Fatal(err)
	}
}

func (x *XiaoAiFSM) onEnterSleeping() {
	fmt.Println("进入睡眠状态，降低功耗...")
	// 关闭非必要功能
}

func (x *XiaoAiFSM) GetUserConversation() {
	d, ok := x.FSM.Metadata("device")
	if !ok {
		internal.GetLogger().Errorf(x.ctx, "get device failed")
		return
	}
	device, ok := d.(*mina.MinaDevice)
	if !ok {
		log.Fatal("assert failed")
	}

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	last_time := time.Now().UnixNano()
	for range ticker.C {
		conv, err := x.mina.GetUserConversations(x.ctx, 5, last_time, device.Hardware, device.DeviceID)
		if err != nil {
			internal.GetLogger().Warnf(x.ctx, "err:%v", err)
			continue
		}

		internal.GetLogger().Debugf(x.ctx, "conv %d", conv.Records[0].Time)

		for _, v := range conv.Records {
			heap.Push(&x.ConversationHeap, v)
		}

		if x.ConversationHeap.Len() != 0 {
			last_time = heap.Pop(&x.ConversationHeap).(*mina.Record).Time
			_ = last_time //avoid (SA4006) go-staticcheck
		}
	}
}
