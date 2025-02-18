package qdrant_store

import (
	"context"
	"testing"
)

func TestLoadDocument(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test file",
			args: args{
				path: "/data/workspace/migpt-go/doc/documents/",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := LoadDocument(context.TODO(), tt.args.path); (err != nil) != tt.wantErr {
				t.Errorf("LoadDocument() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
