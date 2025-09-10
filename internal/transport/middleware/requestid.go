package middleware

import (
	"net/http"

	"github.com/frahmantamala/expense-management/pkg/logger"

	"github.com/google/uuid"
)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.NewString()
		}

		// inject into context
		ctx := logger.With(r.Context(), "traceID", traceID)

		// propagate back to response
		w.Header().Set("X-Trace-ID", traceID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
