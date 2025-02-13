package internal

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"sync"

	"github.com/fatih/color"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var log Ilog

const newline = "\n"
const suffix = "%s, "

type lockedWriter struct {
	sync.Mutex
	writer io.Writer
}

func (w *lockedWriter) Write(p []byte) (n int, err error) {
	w.Lock()
	defer w.Unlock()
	return w.writer.Write(p)
}

func GetLogger() Ilog {
	f := sync.OnceFunc(func() {
		var writer io.Writer
		writer = zapcore.AddSync(&lumberjack.Logger{
			Filename:   "logs/app.log",
			MaxSize:    100, // MB
			MaxBackups: 3,
			MaxAge:     30, // days
		})

		if os.Getenv("stdout_log") == "true" {
			writer = zapcore.Lock(zapcore.AddSync(&lockedWriter{writer: os.Stdout}))
		}

		log = New(writer, Config{
			LogLevel: Debug,
			Colorful: true,
		})
	})

	if log == nil {
		f()
	}

	return log
}

func init() {

}

type Config struct {
	LogLevel LogLevel
	Colorful bool
}

type logger struct {
	io.Writer
	Config
}

func fileWithLineNum() string {
	pcs := [13]uintptr{}
	len := runtime.Callers(3, pcs[:])
	if runtime.Callers(3, pcs[:]) > 0 {
		frames := runtime.CallersFrames(pcs[:len])
		frame, _ := frames.Next()
		return string(strconv.AppendInt(append([]byte(frame.File), ':'), int64(frame.Line), 10))

	}

	return ""
}

// Fatalf implements Ilog.
func (l *logger) Fatalf(ctx context.Context, format string, a ...any) {
	str := fmt.Sprintf(suffix+format+newline, append([]interface{}{fileWithLineNum()}, a...)...)
	panic(str)
}

// Debug implements Ilog.
func (l *logger) Debugf(ctx context.Context, format string, a ...any) {
	if l.LogLevel >= Debug {
		str := fmt.Sprintf(suffix+format+newline, append([]interface{}{fileWithLineNum()}, a...)...)
		color.New(color.FgGreen).Fprint(l.Writer, str)
	}
}

// Error implements Ilog.
func (l *logger) Errorf(ctx context.Context, format string, a ...any) {
	if l.LogLevel >= Error {
		str := fmt.Sprintf(suffix+format+newline, append([]interface{}{fileWithLineNum()}, a...)...)
		color.New(color.FgRed).Fprint(l.Writer, str)
	}
}

// Info implements Ilog.
func (l *logger) Infof(ctx context.Context, format string, a ...any) {
	if l.LogLevel >= Info {
		str := fmt.Sprintf(suffix+format+newline, append([]interface{}{fileWithLineNum()}, a...)...)
		color.New(color.FgGreen).Fprint(l.Writer, str)
	}
}

// Trace implements Ilog.
func (l *logger) Tracef(ctx context.Context, format string, a ...any) {
	if l.LogLevel >= Trace {
		str := fmt.Sprintf(suffix+format+newline, append([]interface{}{fileWithLineNum()}, a...)...)
		color.New(color.FgBlue).Fprint(l.Writer, str)
	}
}

// Warn implements Ilog.
func (l *logger) Warnf(ctx context.Context, format string, a ...any) {
	if l.LogLevel >= Warn {
		str := fmt.Sprintf(suffix+format+newline, append([]interface{}{fileWithLineNum()}, a...)...)
		color.New(color.FgMagenta).Fprint(l.Writer, str)
	}
}

// LogMode implements Ilog.
func (l *logger) LogMode(level LogLevel) Ilog {
	newlogger := *l
	newlogger.LogLevel = level
	return &newlogger
}

func New(writer io.Writer, config Config) Ilog {
	color.NoColor = !config.Colorful

	return &logger{
		Writer: writer,
		Config: config,
	}
}
