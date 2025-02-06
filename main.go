package main

import (
	"context"
	_ "embed"
	"migpt-go/mi/fsm"
)

func main() {
	//首先得到最近的一条消息
	ctx := context.TODO()

	xiaoai := fsm.NewXiaoAi()
	xiaoai.FSM.Event(ctx, fsm.EventWakeUp) // 唤醒

}
