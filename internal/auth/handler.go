package auth

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/frahmantamala/expense-management/internal/core/user"
	"github.com/frahmantamala/expense-management/pkg/logger"
)

type Handler struct {
	Service AuthService
	Logger  *slog.Logger
}

func NewHandler(svc AuthService) *Handler {
	lg := logger.LoggerWrapper()
	if lg == nil {
		lg = slog.Default()
	}
	return &Handler{
		Service: svc,
		Logger:  lg,
	}
}

// Login handles POST /auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var dto LoginDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tokens, err := h.Service.Authenticate(dto)
	if err != nil {
		h.Logger.Error("authentication failed", "error", err)

		switch err {
		case ErrInvalidCredentials:
			h.writeError(w, http.StatusUnauthorized, "invalid credentials")
		case ErrUserInactive:
			h.writeError(w, http.StatusUnauthorized, "user is inactive")
		default:
			if _, ok := err.(ValidationError); ok {
				h.writeError(w, http.StatusBadRequest, err.Error())
			} else {
				h.writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}
		return
	}

	h.writeJSON(w, http.StatusOK, tokens)
}

// RefreshToken handles POST /auth/refresh
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var dto RefreshTokenDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := dto.Validate(); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	tokens, err := h.Service.RefreshTokens(dto.RefreshToken)
	if err != nil {
		h.Logger.Error("token refresh failed", "error", err)

		switch err {
		case ErrInvalidToken, ErrTokenExpired:
			h.writeError(w, http.StatusUnauthorized, "invalid refresh token")
		case ErrUserInactive:
			h.writeError(w, http.StatusUnauthorized, "user is inactive")
		default:
			h.writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, tokens)
}

// Logout handles POST /auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	// Extract token from Authorization header
	token := h.extractTokenFromHeader(r)
	if token == "" {
		h.writeError(w, http.StatusUnauthorized, "missing authorization token")
		return
	}

	// Validate token
	_, err := h.Service.ValidateAccessToken(token)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	// In a production system, you might want to:
	// 1. Add the token to a blacklist/revocation list
	// 2. Store revoked tokens in Redis or database
	// 3. Set shorter TTL on tokens

	// For now, just return success since JWT is stateless
	w.WriteHeader(http.StatusNoContent)
}

// AuthMiddleware validates JWT tokens and adds user to context
func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := h.extractTokenFromHeader(r)
		if token == "" {
			h.writeError(w, http.StatusUnauthorized, "missing authorization token")
			return
		}

		claims, err := h.Service.ValidateAccessToken(token)
		if err != nil {
			h.Logger.Error("token validation failed", "error", err)
			h.writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		// Create a minimal user object for context
		var uid int64
		if claims.UserID != "" {
			if parsed, perr := strconv.ParseInt(claims.UserID, 10, 64); perr == nil {
				uid = parsed
			} else {
				h.Logger.Warn("failed to parse user id from token claims", "value", claims.UserID, "error", perr)
			}
		}
		user := &user.User{
			ID:    uid,
			Email: claims.Email,
		}

		// Add user to request context
		ctx := context.WithValue(r.Context(), ContextUserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Helper methods
func (h *Handler) extractTokenFromHeader(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.Logger.Error("failed to encode JSON response", "error", err)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
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
