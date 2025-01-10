package miot

import (
	"context"
	"migpt-go/internal/common"
)

type Miot struct {
}

type IMiot interface {
	GetMiotDevices(ctx context.Context) ([]*MiotDevice, error)

	//控制器
	common.Controller
}
