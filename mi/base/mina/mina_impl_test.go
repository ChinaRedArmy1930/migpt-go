package mina

import (
	"context"
	"migpt-go/internal/common"
	"migpt-go/mi/base/account"
	"reflect"
	"testing"
	"time"
)

func TestMinaAccount_GetMiDeviceList(t *testing.T) {
	ctx := context.Background()
	type fields struct {
		MIAccount  account.MIAccount
		MinaDevice MinaDevice
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "测试获取数据列表",
			fields: fields{},
			args: args{
				ctx: ctx,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := InitMina(ctx)
			if _, err := m.GetMiDeviceList(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("MinaAccount.GetMiDeviceList() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMinaAccount_GetMiDeviceStatus(t *testing.T) {
	type fields struct {
		MIAccount  account.MIAccount
		MinaDevice MinaDevice
	}
	type args struct {
		ctx context.Context
		id  string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    common.PlayStatus
		wantErr bool
	}{
		{
			name:   "测试获取数据列表",
			fields: fields{},
			args: args{
				ctx: context.TODO(),
				id:  "7d6d4d83-734f-4490-94ed-5035cec3ffa8",
			},
			wantErr: false,
			want:    common.Stopped,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := InitMina(tt.args.ctx)
			got, err := m.GetMiDeviceStatus(tt.args.ctx, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("MinaAccount.GetMiDeviceStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MinaAccount.GetMiDeviceStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMinaAccount_GetUserConversations(t *testing.T) {
	type fields struct {
		MIAccount  account.MIAccount
		DeviceList []*MinaDevice
	}
	type args struct {
		ctx       context.Context
		limit     int
		timestamp int64
		hardware  string
		deviceId  string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "TestMinaAccount_GetUserConversations",
			fields: fields{},
			args: args{
				ctx:       context.TODO(),
				limit:     10,
				timestamp: time.Now().AddDate(-1, 0, 0).Unix(),
				hardware:  "LX01",
				deviceId:  "f0572b8c-8836-42a9-b065-a52837e0d4f4",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := InitMina(tt.args.ctx)
			if _, err := m.GetUserConversations(tt.args.ctx, tt.args.limit, tt.args.timestamp, tt.args.hardware, tt.args.deviceId); (err != nil) != tt.wantErr {
				t.Errorf("MinaAccount.GetUserConversations() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMinaAccount_GetSystemBoard(t *testing.T) {
	type fields struct {
		MIAccount  account.MIAccount
		DeviceList []*MinaDevice
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "TestMinaAccount_GetSystemBoard",
			fields: fields{},
			args: args{
				ctx: context.TODO(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := InitMina(tt.args.ctx)
			if err := m.GetSystemBoard(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("MinaAccount.GetSystemBoard() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
