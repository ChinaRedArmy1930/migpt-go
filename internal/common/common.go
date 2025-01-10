package common

import "context"

type Response[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data,omitempty"`
	Result  T      `json:"result,omitempty"`
}

type Controller interface {
	RegisterControl(name string, fn ControlFunc)
	Controller(ctx context.Context, op string, args ...interface{}) (string, error)
	Control(context.Context, ControlFunc) ControlFunc
}

type ControlFunc func(context.Context, ...interface{}) (string, error)
