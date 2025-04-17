package logging

import (
	"fmt"
	"log/slog"
)

// LoggerWrapper is a wrapper around slog.Logger, which is a structured logger. It is used to log messages in the application.
type LoggerWrapper struct {
	*slog.Logger
}

func NewLoggerWrapper(logger *slog.Logger) *LoggerWrapper {
	return &LoggerWrapper{logger}
}

func (l *LoggerWrapper) Infofln(msg string, args ...any) {
	formattedMsg := fmt.Sprintf(msg, args...)
	l.Logger.Info(formattedMsg)
}

func (l *LoggerWrapper) Errorfln(msg string, args ...any) {
	formattedMsg := fmt.Sprintf(msg, args...)
	l.Logger.Error(formattedMsg)
}

func (l *LoggerWrapper) Debugfln(msg string, args ...any) {
	formattedMsg := fmt.Sprintf(msg, args...)
	l.Logger.Debug(formattedMsg)
}
