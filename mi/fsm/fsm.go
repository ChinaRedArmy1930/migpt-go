package fsm

import (
	"context"
	"fmt"

	"github.com/looplab/fsm"
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
	FSM *fsm.FSM
	// 可以在此添加其他上下文信息
}

func NewXiaoAi() *XiaoAiFSM {
	x := &XiaoAiFSM{}

	x.FSM = fsm.NewFSM(
		StateIdle,
		fsm.Events{
			// 格式：事件名称 -> 源状态 -> 目标状态
			{Name: EventWakeUp, Src: []string{StateIdle, StateSleeping}, Dst: StateListening},
			{Name: EventVoiceDetected, Src: []string{StateListening}, Dst: StateProcessing},
			{Name: EventProcessComplete, Src: []string{StateProcessing}, Dst: StateSpeaking},
			{Name: EventResponseComplete, Src: []string{StateSpeaking}, Dst: StateIdle},
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
	fmt.Println("进入待机状态，等待唤醒...")
	// 启动空闲计时器（比如30秒无操作进入睡眠）
}

func (x *XiaoAiFSM) onEnterListening() {
	fmt.Println("进入聆听状态，开始录音...")
	// 启动语音检测逻辑
}

func (x *XiaoAiFSM) onEnterProcessing() {
	fmt.Println("进入处理状态，分析用户请求...")
	// 调用自然语言处理模块
}

func (x *XiaoAiFSM) onEnterSpeaking() {
	fmt.Println("进入回应状态，播放回答...")
	// 调用语音合成和播放模块
}

func (x *XiaoAiFSM) onEnterSleeping() {
	fmt.Println("进入睡眠状态，降低功耗...")
	// 关闭非必要功能
}
