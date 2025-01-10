package internal

import (
	"context"
)

type LogLevel int

// Colors
const (
	Reset       = "\033[0m"
	Red         = "\033[31m"
	Green       = "\033[32m"
	Yellow      = "\033[33m"
	Blue        = "\033[34m"
	Magenta     = "\033[35m"
	Cyan        = "\033[36m"
	White       = "\033[37m"
	BlueBold    = "\033[34;1m"
	MagentaBold = "\033[35;1m"
	RedBold     = "\033[31;1m"
	YellowBold  = "\033[33;1m"
)

const (
	Silent LogLevel = iota + 1
	Error
	Warn
	Info
	Debug
	Trace
)

type Ilog interface {
	LogMode(LogLevel) Ilog
	Infof(context.Context, string, ...interface{})
	Warnf(context.Context, string, ...interface{})
	Errorf(context.Context, string, ...interface{})
	Debugf(context.Context, string, ...interface{})
	Tracef(context.Context, string, ...interface{})
	Fatalf(context.Context, string, ...interface{})
}
