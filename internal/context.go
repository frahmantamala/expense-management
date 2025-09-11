package internal

import (
	"context"
	"time"
)

type ctxKey string

const ContextUserKey ctxKey = "userID"

func UserIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if userID, ok := ctx.Value(ContextUserKey).(string); ok {
		return userID
	}
	return ""
}

func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ContextUserKey, userID)
}

// WithTimeout returns a context with timeout, defaulting to 5 seconds if duration is zero or negative.
func WithTimeout(ctx context.Context, duration time.Duration) (context.Context, context.CancelFunc) {
	if duration <= 0 {
		duration = 5 * time.Second
	}
	return context.WithTimeout(ctx, duration)
}
