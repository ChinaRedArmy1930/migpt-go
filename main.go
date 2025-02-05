package main

import (
	"context"
	internal "migpt-go/internal/log"
	"migpt-go/mi/base/mina"
	"time"
)

func main() {
	//首先得到最近的一条消息
	ctx := context.TODO()
	mina := mina.InitMina(ctx)
	device_list, err := mina.GetMiDeviceList(ctx)
	if err != nil {
		panic(err)
	}

	ticker := time.NewTicker(time.Second * 10)

	for range ticker.C {
		//	k := math.Floor(math.Log2(float64(len(device_list)))) + 1
		for _, device := range device_list {
			if device.Name == "小爱音箱mini" {
				conv, err := mina.GetUserConversations(ctx, 10, time.Now().Unix(), device.Hardware, device.DeviceID)
				if err != nil {
					internal.GetLogger().Warnf(ctx, "err:%v", err)
					continue
				}

				internal.GetLogger().Infof(ctx, "conv:%#v", conv)
			}
		}

	}
}
