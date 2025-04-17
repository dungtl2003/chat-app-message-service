package logging

import (
	"fmt"
	"log/slog"
	"os"
)

func NewLogger(level string, kind string) (*slog.Logger, error) {
	writer := os.Stdout

	var slogLogLevel slog.Level
	switch level {
	case "DEBUG":
		slogLogLevel = slog.LevelDebug
	case "INFO":
		slogLogLevel = slog.LevelInfo
	case "WARN":
		slogLogLevel = slog.LevelWarn
	case "ERROR":
		slogLogLevel = slog.LevelError
	default:
		return nil, fmt.Errorf("invalid log level")
	}

	var handler slog.Handler
	switch kind {
	case "TEXT":
		handler = slog.NewTextHandler(writer, &slog.HandlerOptions{Level: slogLogLevel})
	case "JSON":
		handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: slogLogLevel})
	default:
		return nil, fmt.Errorf("invalid log kind")
	}

	return slog.New(handler), nil
}
