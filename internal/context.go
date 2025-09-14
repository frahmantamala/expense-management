package internal

import (
	"context"
	"errors"
	"time"
)

type ctxKey string

const (
	ContextUserKey     ctxKey = "userID"
	ContextAuthUserKey ctxKey = "user"
)

var ErrForbidden = errors.New("forbidden")

type User struct {
	ID          int64    `json:"id"`
	Email       string   `json:"email"`
	Permissions []string `json:"permissions,omitempty"`
}

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

func UserFromContext(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value(ContextAuthUserKey).(*User)
	return u, ok
}

func ContextWithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, ContextAuthUserKey, user)
}

func WithTimeout(ctx context.Context, duration time.Duration) (context.Context, context.CancelFunc) {
	if duration <= 0 {
		duration = 5 * time.Second
	}
	return context.WithTimeout(ctx, duration)
}
