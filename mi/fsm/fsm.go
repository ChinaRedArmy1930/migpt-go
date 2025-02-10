package fsm

import (
	"context"
	"fmt"
	"log"
	"migpt-go/config"
	"migpt-go/internal/common"
	internal "migpt-go/internal/log"
	"migpt-go/mi/base/mina"
	"migpt-go/mi/base/miot"
	"os"
	"time"

	chroma_go "github.com/amikos-tech/chroma-go/types"
	"github.com/google/uuid"
	"github.com/looplab/fsm"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores/chroma"
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

// XiaoAiFSM 小爱同学状态机
type XiaoAiFSM struct {
	FSM    *fsm.FSM
	ctx    context.Context
	mina   mina.IMina
	miot   miot.IMiot
	qqueue chan<- string
	aqueue <-chan common.Answer
}

func NewXiaoAi(question chan<- string, answer <-chan common.Answer) *XiaoAiFSM {
	ctx := context.TODO()
	x := &XiaoAiFSM{
		mina:   mina.InitMina(ctx),
		miot:   miot.InitMiot(ctx),
		ctx:    ctx,
		qqueue: question,
		aqueue: answer,
	}

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
			"enter_state": func(ctx context.Context, e *fsm.Event) { x.enterState(e) },
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

// 各个状态的具体处理逻辑
func (x *XiaoAiFSM) onEnterIdle() {
	// 启动空闲计时器（比如30秒无操作进入睡眠）
}

func (x *XiaoAiFSM) onEnterListening() {
	device_list, err := x.mina.GetMiDeviceList(x.ctx)
	if err != nil {
		panic(err)
	}

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for range ticker.C {
		for _, device := range device_list {
			if device.Name == "小爱音箱mini" {
				conv, err := x.mina.GetUserConversations(x.ctx, 1, time.Now().UnixNano(), device.Hardware, device.DeviceID)
				if err != nil {
					internal.GetLogger().Warnf(x.ctx, "err:%v", err)
					continue
				}

				internal.GetLogger().Infof(x.ctx, "conv:%#v", conv.Records[0].Query)

				msg := conv.Records[0].Query

				//判断是否需要进入Ai模式
				if !AiMode(msg) {
					x.FSM.SetMetadata("question", msg)
					x.FSM.SetMetadata("device_id", device.DeviceID)
					x.FSM.Event(x.ctx, EventVoiceDetected)
					return
				}
			}
		}
	}
}

func AiMode(msg string) bool {
	store, errNs := chroma.New(
		chroma.WithChromaURL(os.Getenv("CHROMA_URL")),
		chroma.WithOpenAIAPIKey(os.Getenv("OPENAI_API_KEY")),
		chroma.WithDistanceFunction(chroma_go.COSINE),
		chroma.WithNameSpace(uuid.New().String()),
	)
	if errNs != nil {
		log.Fatalf("new: %v\n", errNs)
	}

	// Add documents to the vector store.
	docs := make([]schema.Document, 0)
	for _, v := range config.DefaultConfig.Ai.WakeUpKeyWords {
		docs = append(docs, schema.Document{PageContent: v})
	}

	_, errAd := store.AddDocuments(context.Background(), docs)
	if errAd != nil {
		log.Fatalf("AddDocument: %v\n", errAd)
	}

	store.SimilaritySearch(context.Background(), msg, 1)
	return true
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
		fmt.Print(answer)
		ans += answer.Chunk.String()
	}

	x.FSM.SetMetadata("answer", ans)

	x.FSM.Event(x.ctx, EventProcessComplete)
	internal.GetLogger().Debugf(x.ctx, "get answer => %s", ans)
}

func (x *XiaoAiFSM) onEnterSpeaking() {
	internal.GetLogger().Infof(x.ctx, "进入回应状态，播放回答...")
	// 调用语音合成和播放模块
	d, _ := x.FSM.Metadata("device_id")
	ans, _ := x.FSM.Metadata("answer")
	x.mina.Controller(x.ctx, "play", ans.(string), "", d.(string))
}

func (x *XiaoAiFSM) onEnterSleeping() {
	fmt.Println("进入睡眠状态，降低功耗...")
	// 关闭非必要功能
}
