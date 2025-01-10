package mina

import (
	"context"
	"migpt-go/internal/common"
)

type IMina interface {
	GetMiDeviceList(ctx context.Context) ([]*MinaDevice, error)
	GetMiDeviceStatus(ctx context.Context, deviceId string) (common.PlayStatus, error)
	GetUserConversations(ctx context.Context, limit int, timestamp int64, hardware string, deviceId string) (*MiConversation, error)
	//Todo 接口可能不支持
	GetSystemBoard(ctx context.Context) error

	//控制器
	common.Controller
}
