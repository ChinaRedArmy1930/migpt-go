package account

import (
	"context"
	"migpt-go/internal/common"
	internal_sync "migpt-go/internal/sync"

	"os"
	"strconv"
	"testing"

	"github.com/google/uuid"
)

func TestMIAccount_GetLoginAccount(t *testing.T) {
	type fields struct {
		Sid          common.Sid
		DeviceId     string
		UserId       int64
		Pass         *MiPass
		ServiceToken string
		Did          string
		PubSub       internal_sync.PubSub
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
			name: "测试获取账号信息",
			fields: fields{
				Sid:      "xiaomiio",
				DeviceId: uuid.New().String(),
				UserId: func() int64 {
					t, _ := strconv.Atoi(os.Getenv("mi_user_id"))
					return int64(t)
				}(),
				Pass:         &MiPass{},
				ServiceToken: "",
				Did:          "习丹丹的小米音箱",
			},
			args: args{
				ctx: context.TODO(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MIAccount{
				Sid:          tt.fields.Sid,
				UserId:       tt.fields.UserId,
				Pass:         tt.fields.Pass,
				ServiceToken: tt.fields.ServiceToken,
				Did:          tt.fields.Did,
			}
			if _, err := m.GetLoginAccount(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("MIAccount.GetLoginAccount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
