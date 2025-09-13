package transport

import (
	"encoding/json"
	"log/slog"
	"net/http"

	errors "github.com/frahmantamala/expense-management/internal"
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

// HandleError handles structured AppError responses
func (h *BaseHandler) HandleError(w http.ResponseWriter, err error) {
	if appErr, ok := errors.IsAppError(err); ok {
		h.Logger.Error("application error",
			"type", appErr.Type,
			"code", appErr.Code,
			"message", appErr.Message,
			"status", appErr.StatusCode,
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(appErr.StatusCode)

		if encodeErr := json.NewEncoder(w).Encode(map[string]interface{}{
			"error": appErr,
		}); encodeErr != nil {
			h.Logger.Error("failed to encode error response", "error", encodeErr)
		}
		return
	}

	// Fallback for non-AppError errors
	h.Logger.Error("internal error", "error", err)
	h.WriteError(w, http.StatusInternalServerError, "Internal server error")
}

// HandleServiceError provides common error handling for service layer errors
func (h *BaseHandler) HandleServiceError(w http.ResponseWriter, err error) {
	// Map common Go errors to AppErrors
	switch err.Error() {
	case "record not found", "sql: no rows in result set":
		h.HandleError(w, errors.ErrExpenseNotFound)
		return
	}

	// Handle structured errors
	h.HandleError(w, err)
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
