package account

import "context"

type IMIAccount interface {
	// 获取登录信息
	GetLoginAccount(ctx context.Context) (*MIAccount, error)

	// 登录到小米服务
	ServiceLogin(ctx context.Context) (*MiAccountResponse, error)

	//刷新登录token
	RefreshLoginToken(ctx context.Context) error

	//获取登录token
	TakeLoginToken(ctx context.Context) error
}
