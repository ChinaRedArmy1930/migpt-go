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
	"migpt-go/mi/spec"
	"os"
	"strings"
	"time"

	"github.com/looplab/fsm"

	"github.com/qdrant/go-client/qdrant"
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

type tdid string
type tkeyword string

// XiaoAiFSM 小爱同学状态机
type XiaoAiFSM struct {
	FSM              *fsm.FSM
	ctx              context.Context
	mina             mina.IMina
	miot             miot.IMiot
	qqueue           chan<- string
	aqueue           <-chan common.Answer
	QdrantClient     *qdrant.Client
	VectorClient     *openai.LLM
	TimeoutStatus    *status.TimeoutStatus
	ConversationHeap mina.Records
	actCommand       map[tdid]map[tkeyword]act
}

type act struct {
	aiid int
	piid int
	siid int
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
		actCommand:       make(map[tdid]map[tkeyword]act),
	}

	//sudo docker run -d  -p 6333:6333   qdrant/qdrant
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:                   "9.134.91.245",
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

	//获取设备的SPEC指令
	{
		//首先获取所有设备的spec
		instances := spec.GetAllSpec(x.ctx)
		model_map := make(map[string]string)
		for _, v := range instances.Instances {
			model_map[v.Model] = v.Type
		}

		miot_devices, err := x.miot.GetMiotDevices(x.ctx)
		if err != nil {
			log.Fatal(err)
		}

		//urn:miot-spec-v2:device:speaker:0000A015:xiaomi-lx01:1
		//xiaomi.wifispeaker.lx01
		device_type := ""
		for _, device := range miot_devices {
			if device.Name == "小爱音箱mini" {
				x.FSM.SetMetadata("did", device.Did)
			}
			x.actCommand[tdid(device.Did)] = make(map[tkeyword]act)
			device_type = model_map[device.Model]
			internal.GetLogger().Debugf(x.ctx, "model => %s, type => %s", device.Model, device_type)
			//找到type后, 找到设备对应的指令
			device_spec := spec.GetDeviceSpecCommmond(device_type)

			//找到智能音响service的指令
			for _, v := range device_spec.Services {
				if strings.Contains(v.Type, common.IntelligentSpeakerService) {
					siid := v.IID
					piid := -1
					for _, property := range v.Properties {
						switch {
						case strings.Contains(property.Type, common.TextContentProperty):
							{
								piid = property.IID
								x.actCommand[tdid(device.Did)][common.PlayingWord] = act{
									piid: piid,
									siid: siid,
								}
								break
							}
						}
					}

					for _, action := range v.Actions {
						switch {
						case strings.Contains(action.Type, common.WakeUpAction):
							tmpAct := x.actCommand[tdid(device.Did)][tkeyword(common.WakeUpWord)]
							tmpAct.aiid = action.IID
							x.actCommand[tdid(device.Did)][tkeyword(common.WakeUpWord)] = tmpAct
						case strings.Contains(action.Type, common.PlayTextAction):
							tmpAct := x.actCommand[tdid(device.Did)][tkeyword(common.PlayTextWord)]
							tmpAct.aiid = action.IID
							x.actCommand[tdid(device.Did)][tkeyword(common.WakeUpWord)] = tmpAct
						case strings.Contains(action.Type, common.PauseAction):
							tmpAct := x.actCommand[tdid(device.Did)][tkeyword(common.PlayTextWord)]
							tmpAct.aiid = action.IID
							x.actCommand[tdid(device.Did)][tkeyword(common.WakeUpWord)] = tmpAct
						}
					}
				}
			}
		}
	}

	go x.GetUserConversation()

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

	did, ok := x.FSM.Metadata("did")
	if !ok {
		//没拿到did
		internal.GetLogger().Errorf(x.ctx, "get did failed")
		return
	}

	ticker := time.NewTicker(time.Second)

	for range ticker.C {
		if x.ConversationHeap.Len() != 0 {
			break
		}
	}

	msg := heap.Pop(&x.ConversationHeap).(*mina.Record).Query
	internal.GetLogger().Infof(x.ctx, "获取到用户问题:%s", msg)
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
				//检测到AI后,让音响静音"
				r, err := x.miot.Controller(x.ctx, "action", device_id,
					x.actCommand[tdid(did.(string))][common.PauseWord].siid,
					x.actCommand[tdid(did.(string))][common.PauseWord].aiid,
				)
				if err != nil {
					internal.GetLogger().Errorf(x.ctx, "pause error => %s", err)
					return
				}

				internal.GetLogger().Errorf(x.ctx, "小爱静音 => %s", r)

				x.mina.Controller(x.ctx, "play", "检测到AI召唤词,进入AI模式", "", device_id)
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
	last_time := time.Now().UnixMilli()
	//第一条消息需要过滤掉
	for range ticker.C {
		conv, err := x.mina.GetUserConversations(x.ctx, 10, time.Now().UnixMilli(), device.Hardware, device.DeviceID)
		if err != nil {
			internal.GetLogger().Warnf(x.ctx, "err:%v", err)
			continue
		}

		for _, v := range conv.Records {
			if v.Time > last_time {
				internal.GetLogger().Warnf(x.ctx, "get conv msg %s", v.Query)
				heap.Push(&x.ConversationHeap, v)
			}
		}

		if x.ConversationHeap.Len() != 0 {
			last_time = (x.ConversationHeap[0]).Time
			_ = last_time //avoid (SA4006) go-staticcheck
		}
	}
}
