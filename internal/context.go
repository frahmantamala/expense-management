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

// User represents a minimal user structure for context
// This avoids import cycles by not importing the full auth.User
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

// UserFromContext retrieves the authenticated user from the context
func UserFromContext(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value(ContextAuthUserKey).(*User)
	return u, ok
}

// ContextWithUser adds a user to the context
func ContextWithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, ContextAuthUserKey, user)
}

// WithTimeout returns a context with timeout, defaulting to 5 seconds if duration is zero or negative.
func WithTimeout(ctx context.Context, duration time.Duration) (context.Context, context.CancelFunc) {
	if duration <= 0 {
		duration = 5 * time.Second
	}
	return context.WithTimeout(ctx, duration)
}
