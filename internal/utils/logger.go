package utils

import (
	"fmt"
	"os"
)

type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

type DefaultLogger struct {
	Level  LogLevel
	Output *os.File
}

func NewLogger(level LogLevel) *DefaultLogger {
	return &DefaultLogger{
		Level:  level,
		Output: os.Stdout,
	}
}

func (l *DefaultLogger) Log(level LogLevel, msg string, args ...interface{}) {
	if level < l.Level {
		return
	}
	fmt.Fprintf(l.Output, "[%s] %s\n", level.String(), fmt.Sprintf(msg, args...))
}

func (l *DefaultLogger) Debug(msg string, args ...interface{}) {
	l.Log(LogLevelDebug, msg, args...)
}

func (l *DefaultLogger) Info(msg string, args ...interface{}) {
	l.Log(LogLevelInfo, msg, args...)
}

func (l *DefaultLogger) Warn(msg string, args ...interface{}) {
	l.Log(LogLevelWarn, msg, args...)
}

func (l *DefaultLogger) Error(msg string, args ...interface{}) {
	l.Log(LogLevelError, msg, args...)
}

type SilentLogger struct{}

func (s *SilentLogger) Debug(msg string, args ...interface{}) {}
func (s *SilentLogger) Info(msg string, args ...interface{})  {}
func (s *SilentLogger) Warn(msg string, args ...interface{})  {}
func (s *SilentLogger) Error(msg string, args ...interface{}) {}
