package logger

import (
	"log/slog"
	"os"
)

var defaultLogger *slog.Logger

func Init(env string) {
	var handler slog.Handler

	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

func L() *slog.Logger {
	return defaultLogger
}
