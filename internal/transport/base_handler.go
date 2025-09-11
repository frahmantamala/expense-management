package transport

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/frahmantamala/expense-management/pkg/logger"
)

// BaseHandler provides common functionality for HTTP handlers
type BaseHandler struct {
	Logger *slog.Logger
}

// NewBaseHandler creates a base handler with logger
func NewBaseHandler(lg *slog.Logger) *BaseHandler {
	if lg == nil {
		lg = logger.LoggerWrapper()
		if lg == nil {
			lg = slog.Default()
		}
	}
	return &BaseHandler{Logger: lg}
}

// WriteJSON writes a JSON response
func (h *BaseHandler) WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.Logger.Error("failed to encode JSON response", "error", err)
	}
}

// WriteError writes an error response
func (h *BaseHandler) WriteError(w http.ResponseWriter, status int, message string) {
	h.Logger.Error("http error", "status", status, "message", message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errorResp := map[string]interface{}{
		"code":    status,
		"message": message,
	}

	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		h.Logger.Error("failed to encode error response", "error", err)
	}
}

// ExtractTokenFromHeader extracts Bearer token from Authorization header
func (h *BaseHandler) ExtractTokenFromHeader(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return ""
	}

	return authHeader[7:]
}
