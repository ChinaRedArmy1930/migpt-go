package fsm

import (
	"bytes"
	"context"
	"fmt"
	"migpt-go/doc"
	internal "migpt-go/internal/log"
	"migpt-go/mi/base/mina"
	"migpt-go/mi/base/miot"
	"os"
	"text/template"
	"time"

	"github.com/looplab/fsm"
	"github.com/tmc/langchaingo/llms/openai"
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
	FSM  *fsm.FSM
	ctx  context.Context
	mina mina.IMina
	miot miot.IMiot
}

func NewXiaoAi() *XiaoAiFSM {
	ctx := context.TODO()
	x := &XiaoAiFSM{
		mina: mina.InitMina(ctx),
		miot: miot.InitMiot(ctx),
		ctx:  ctx,
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

	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for range ticker.C {
		for _, device := range device_list {
			if device.Name == "小爱音箱mini" {
				conv, err := x.mina.GetUserConversations(x.ctx, 10, time.Now().Unix(), device.Hardware, device.DeviceID)
				if err != nil {
					internal.GetLogger().Warnf(x.ctx, "err:%v", err)
					continue
				}

				internal.GetLogger().Infof(x.ctx, "conv:%#v", conv)

				msg := "测试"
				x.FSM.Event(x.ctx, EventVoiceDetected, msg)
				return
			}
		}
	}
}

func (x *XiaoAiFSM) onEnterProcessing() {
	internal.GetLogger().Infof(x.ctx, "进入处理状态，分析用户请求...")
	// 调用自然语言处理模块
	t, err := template.New("标准化提示词模板").Parse(doc.DefaultSystemTemplate)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	data := map[string]string{}
	err = t.Execute(&buf, data)
	if err != nil {
		internal.GetLogger().Warnf(x.ctx, "build prompt failed => %s", err)
		return
	}

	llm, err := openai.New(openai.WithBaseURL("https://api.siliconflow.cn/v1"),
		openai.WithModel("deepseek-ai/DeepSeek-V3"),
		openai.WithToken(os.Getenv("apikey")))
	if err != nil {
		internal.GetLogger().Warnf(x.ctx, "new openai failed => %s", err)
		return
	}

	result, err := llm.Call(x.ctx, buf.String())
	if err != nil {
		internal.GetLogger().Warnf(x.ctx, "call failed => %s", err)
		return
	}

	fmt.Printf("result => %s", result)

	//调用接口输出
}

func (x *XiaoAiFSM) onEnterSpeaking() {
	fmt.Println("进入回应状态，播放回答...")
	// 调用语音合成和播放模块
}

func (x *XiaoAiFSM) onEnterSleeping() {
	fmt.Println("进入睡眠状态，降低功耗...")
	// 关闭非必要功能
}
