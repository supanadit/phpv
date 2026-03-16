package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/log"
)

type Logger struct {
	logger  *log.Logger
	verbose bool
	quiet   bool
	mu      sync.Mutex
}

var (
	defaultLogger *Logger
	loggerOnce    sync.Once
)

func GetLogger() *Logger {
	loggerOnce.Do(func() {
		defaultLogger = NewLogger()
	})
	return defaultLogger
}

func NewLogger() *Logger {
	logger := log.New(os.Stderr)
	logger.SetPrefix("")
	logger.SetReportTimestamp(false)
	logger.SetFormatter(log.TextFormatter)
	logger.SetLevel(log.InfoLevel)

	return &Logger{
		logger:  logger,
		verbose: false,
		quiet:   false,
	}
}

func (l *Logger) SetVerbose(verbose bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.verbose = verbose
	if verbose {
		l.logger.SetLevel(log.DebugLevel)
	} else {
		l.logger.SetLevel(log.InfoLevel)
	}
}

func (l *Logger) SetQuiet(quiet bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.quiet = quiet
	if quiet {
		l.logger.SetOutput(io.Discard)
	} else {
		l.logger.SetOutput(os.Stderr)
	}
}

func (l *Logger) IsVerbose() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.verbose
}

func (l *Logger) IsQuiet() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.quiet
}

func (l *Logger) Debug(args ...interface{}) {
	if l.IsVerbose() {
		msg := dimText(fmt.Sprint(args...))
		l.logger.Debug(msg)
	}
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.IsVerbose() {
		msg := fmt.Sprintf(format, args...)
		l.logger.Debug(dimText(msg))
	}
}

func (l *Logger) Info(args ...interface{}) {
	if l.IsQuiet() {
		return
	}
	msg := infoText(fmt.Sprint(args...))
	l.logger.Info(msg)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	if l.IsQuiet() {
		return
	}
	msg := fmt.Sprintf(format, args...)
	l.logger.Info(infoText(msg))
}

func (l *Logger) Warn(args ...interface{}) {
	if l.IsQuiet() {
		return
	}
	msg := warnText(fmt.Sprint(args...))
	l.logger.Warn(msg)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	if l.IsQuiet() {
		return
	}
	msg := fmt.Sprintf(format, args...)
	l.logger.Warn(warnText(msg))
}

func (l *Logger) Error(args ...interface{}) {
	msg := errText(fmt.Sprint(args...))
	l.logger.Error(msg)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.logger.Error(errText(msg))
}

func (l *Logger) Success(args ...interface{}) {
	if l.IsQuiet() {
		return
	}
	msg := successText(fmt.Sprint(args...))
	l.logger.Info(msg)
}

func (l *Logger) Successf(format string, args ...interface{}) {
	if l.IsQuiet() {
		return
	}
	msg := fmt.Sprintf(format, args...)
	l.logger.Info(successText(msg))
}

func (l *Logger) Dim(args ...interface{}) {
	if l.IsQuiet() {
		return
	}
	msg := dimText(fmt.Sprint(args...))
	l.logger.Info(msg)
}

func (l *Logger) Dimf(format string, args ...interface{}) {
	if l.IsQuiet() {
		return
	}
	msg := fmt.Sprintf(format, args...)
	l.logger.Info(dimText(msg))
}

func (l *Logger) Print(args ...interface{}) {
	if l.IsQuiet() {
		return
	}
	fmt.Print(args...)
}

func (l *Logger) Printf(format string, args ...interface{}) {
	if l.IsQuiet() {
		return
	}
	fmt.Printf(format, args...)
}

func (l *Logger) Println(args ...interface{}) {
	if l.IsQuiet() {
		return
	}
	fmt.Println(args...)
}

func (l *Logger) With(key string, value interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	newLogger := &Logger{
		logger:  l.logger.With(key, value),
		verbose: l.verbose,
		quiet:   l.quiet,
	}
	return newLogger
}

func dimText(s string) string {
	return DimStyle.Render(s)
}

func infoText(s string) string {
	return InfoStyle.Render(s)
}

func warnText(s string) string {
	return WarningStyle.Render(s)
}

func errText(s string) string {
	return ErrorStyle.Render(s)
}

func successText(s string) string {
	return SuccessStyle.Render(s)
}

func init() {
	GetLogger()
}

func Debug(args ...interface{}) {
	GetLogger().Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	GetLogger().Debugf(format, args...)
}

func Info(args ...interface{}) {
	GetLogger().Info(args...)
}

func Infof(format string, args ...interface{}) {
	GetLogger().Infof(format, args...)
}

func Warn(args ...interface{}) {
	GetLogger().Warn(args...)
}

func Warnf(format string, args ...interface{}) {
	GetLogger().Warnf(format, args...)
}

func Error(args ...interface{}) {
	GetLogger().Error(args...)
}

func Errorf(format string, args ...interface{}) {
	GetLogger().Errorf(format, args...)
}

func Success(args ...interface{}) {
	GetLogger().Success(args...)
}

func Successf(format string, args ...interface{}) {
	GetLogger().Successf(format, args...)
}

func Dim(args ...interface{}) {
	GetLogger().Dim(args...)
}

func Dimf(format string, args ...interface{}) {
	GetLogger().Dimf(format, args...)
}

func Print(args ...interface{}) {
	GetLogger().Print(args...)
}

func Printf(format string, args ...interface{}) {
	GetLogger().Printf(format, args...)
}

func Println(args ...interface{}) {
	GetLogger().Println(args...)
}

func IsVerbose() bool {
	return GetLogger().IsVerbose()
}

func IsQuiet() bool {
	return GetLogger().IsQuiet()
}

func SetVerbose(verbose bool) {
	GetLogger().SetVerbose(verbose)
}

func SetQuiet(quiet bool) {
	GetLogger().SetQuiet(quiet)
}

func IndentLines(text string, indent string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}
