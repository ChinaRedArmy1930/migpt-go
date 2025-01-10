package sync

import (
	"context"
	"migpt-go/internal/buffer"
	internal "migpt-go/internal/log"
)

type CallbackSerializer struct {
	done      chan struct{}
	callbacks *buffer.Unbounded
	ctx       context.Context
}

func NewCallbackSerializer(ctx context.Context) *CallbackSerializer {
	cs := &CallbackSerializer{
		done:      make(chan struct{}),
		callbacks: buffer.NewUnbounded(),
	}

	go cs.run(ctx)
	return cs
}

func (cs *CallbackSerializer) TrySchedule(f func(context.Context)) {
	_ = cs.callbacks.Put(f)
}

func (cs *CallbackSerializer) ScheduleOr(f func(context.Context), onFailure func()) {
	if cs.callbacks.Put(f) != nil {
		internal.GetLogger().Fatalf(cs.ctx, "put %#v failed", f)
		onFailure()
	}
}
func (cs *CallbackSerializer) run(ctx context.Context) {
	defer close(cs.done)

	//当ctx被关闭后执行剩余的所有任务
	context.AfterFunc(ctx, func() {
		cs.callbacks.Close()

		for cb := range cs.callbacks.Get() {
			cs.callbacks.Load()
			cb.(func(context.Context))(ctx)
		}
	})

	for ctx.Err() != nil {
		select {
		case <-ctx.Done():
		case cb := <-cs.callbacks.Get():
			cs.callbacks.Load()
			cb.(func(context.Context))(ctx)
		}
	}

}

func (cs *CallbackSerializer) Done() <-chan struct{} {
	return cs.done
}
