package auth

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/frahmantamala/expense-management/internal/transport"
	"github.com/frahmantamala/expense-management/pkg/logger"
)

type Handler struct {
	*transport.BaseHandler
	Service ServiceAPI
}

func NewHandler(svc ServiceAPI) *Handler {
	lg := logger.LoggerWrapper()
	if lg == nil {
		lg = slog.Default()
	}
	return &Handler{
		BaseHandler: transport.NewBaseHandler(lg),
		Service:     svc,
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var dto LoginDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		h.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tokens, err := h.Service.Authenticate(dto)
	if err != nil {
		h.Logger.Error("authentication failed", "error", err)

		switch err {
		case ErrInvalidCredentials:
			h.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		case ErrUserInactive:
			h.WriteError(w, http.StatusUnauthorized, "user is inactive")
		default:
			if _, ok := err.(ValidationError); ok {
				h.WriteError(w, http.StatusBadRequest, err.Error())
			} else {
				h.WriteError(w, http.StatusInternalServerError, "internal server error")
			}
		}
		return
	}

	h.WriteJSON(w, http.StatusOK, tokens)
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var dto RefreshTokenDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		h.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := dto.Validate(); err != nil {
		h.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	tokens, err := h.Service.RefreshTokens(dto.RefreshToken)
	if err != nil {
		h.Logger.Error("token refresh failed", "error", err)

		switch err {
		case ErrInvalidToken, ErrTokenExpired:
			h.WriteError(w, http.StatusUnauthorized, "invalid refresh token")
		case ErrUserInactive:
			h.WriteError(w, http.StatusUnauthorized, "user is inactive")
		default:
			h.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	h.WriteJSON(w, http.StatusOK, tokens)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token := h.ExtractTokenFromHeader(r)
	if token == "" {
		h.WriteError(w, http.StatusUnauthorized, "missing authorization token")
		return
	}

	// Validate token
	_, err := h.Service.ValidateAccessToken(token)
	if err != nil {
		h.WriteError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := h.ExtractTokenFromHeader(r)
		if token == "" {
			h.Logger.Error("auth middleware: missing authorization token")
			h.WriteError(w, http.StatusUnauthorized, "missing authorization token")
			return
		}

		tokenPrefix := token
		if len(token) > 20 {
			tokenPrefix = token[:20]
		}
		h.Logger.Info("auth middleware: validating token", "token_prefix", tokenPrefix)

		claims, err := h.Service.ValidateAccessToken(token)
		if err != nil {
			h.Logger.Error("token validation failed", "error", err, "token_prefix", tokenPrefix)
			h.WriteError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		h.Logger.Info("auth middleware: token validated successfully", "user_id", claims.UserID, "email", claims.Email)

		var uid int64
		if claims.UserID != "" {
			if parsed, perr := strconv.ParseInt(claims.UserID, 10, 64); perr == nil {
				uid = parsed
			} else {
				h.Logger.Warn("failed to parse user id from token claims", "value", claims.UserID, "error", perr)
			}
		}

		coreUser, err := h.Service.GetUserWithPermissions(uid)
		if err != nil {
			h.Logger.Error("auth middleware: failed to load user permissions", "user_id", uid, "error", err)
			h.WriteError(w, http.StatusUnauthorized, "user not found")
			return
		}

		h.Logger.Info("auth middleware: adding user to context", "user_id", uid, "email", claims.Email)

		ctx := context.WithValue(r.Context(), ContextUserKey, coreUser)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
