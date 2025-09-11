package logger

import (
	"context"
	"log/slog"
)

type ctxKey string

const loggerKey ctxKey = "logger"

// With returns a new context that includes a logger with fields.
func With(ctx context.Context, fields ...any) context.Context {
	l := From(ctx).With(fields...)
	return context.WithValue(ctx, loggerKey, l)
}

// From returns the logger stored in context, or default if missing.
func From(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}
	return LoggerWrapper()
}
