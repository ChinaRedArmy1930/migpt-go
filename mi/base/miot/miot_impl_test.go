package miot

import (
	"context"
	"migpt-go/mi/base/account"
	"reflect"
	"testing"
)

func TestMiotAccount_GetMiotDevices(t *testing.T) {
	type fields struct {
		MIAccount account.MIAccount
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*MiotDevice
		wantErr bool
	}{
		{
			name:   "TestMiotAccount_GetMiotDevices",
			fields: fields{},
			args: args{
				ctx: context.TODO(),
			},
			want:    []*MiotDevice{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := InitMiot(tt.args.ctx)
			_, err := m.GetMiotDevices(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("MiotAccount.GetMiotDevices() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
func Test_encodeMiIOT(t *testing.T) {
	type args struct {
		method    string
		uri       string
		data      map[string]interface{}
		ssecurity string
	}
	tests := []struct {
		name string
		args args
		want MiIOTRequest
	}{
		{
			name: "Test_encodeMiIOT",
			args: args{
				method:    "POST",
				uri:       "/home/device_list",
				ssecurity: "123",
			},
			want: MiIOTRequest{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := signMiotParams(tt.args.uri, tt.args.data, tt.args.ssecurity); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("encodeMiIOT() = %v, want %v", got, tt.want)
			}
		})
	}
}
