package logging

import (
	"log/slog"
	"os"
)

const (
	DEBUG LoggerLevel = "DEBUG"
	INFO  LoggerLevel = "INFO"
	WARN  LoggerLevel = "WARN"
	ERROR LoggerLevel = "ERROR"

	TEXT LoggerKind = "TEXT"
	JSON LoggerKind = "JSON"
)

type LoggerLevel string
type LoggerKind string

func NewLogger(level LoggerLevel, kind LoggerKind) (*slog.Logger, error) {
	writer := os.Stdout

	var slogLogLevel slog.Level
	switch level {
	case DEBUG:
		slogLogLevel = slog.LevelDebug
	case INFO:
		slogLogLevel = slog.LevelInfo
	case WARN:
		slogLogLevel = slog.LevelWarn
	case ERROR:
		slogLogLevel = slog.LevelError
	}

	var handler slog.Handler
	switch kind {
	case TEXT:
		handler = slog.NewTextHandler(writer, &slog.HandlerOptions{Level: slogLogLevel})
	case JSON:
		handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: slogLogLevel})
	}

	return slog.New(handler), nil
}

func IsValidLoggerLevel(level string) bool {
	switch level {
	case string(DEBUG), string(INFO), string(WARN), string(ERROR):
		return true
	}
	return false
}

func IsValidLoggerKind(kind string) bool {
	switch kind {
	case string(TEXT), string(JSON):
		return true
	}
	return false
}
