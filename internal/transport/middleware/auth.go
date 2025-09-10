package middleware

import (
	"net/http"

	"github.com/frahmantamala/expense-management/pkg/logger"
)

func UserContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Example only — real app: parse JWT, session, etc.
		userID := r.Header.Get("X-User-ID")

		ctx := logger.With(r.Context(), "userID", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
