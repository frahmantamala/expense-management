package user

import (
	"log/slog"
	"net/http"

	"github.com/frahmantamala/expense-management/internal/auth"
	"github.com/frahmantamala/expense-management/internal/transport"
	"github.com/frahmantamala/expense-management/pkg/logger"
)

type ServiceAPI interface {
	GetByID(userID int64) (*User, error)
	GetPermissions(userID int64) ([]string, error)
}

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

// GetCurrentUser handles GET /users/me
func (h *Handler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	h.Logger.Info("GetCurrentUser: starting request")

	user, ok := auth.UserFromContext(r.Context())
	if !ok || user == nil {
		h.Logger.Error("GetCurrentUser: user not found in context", "ok", ok, "user_nil", user == nil)
		h.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	h.Logger.Info("GetCurrentUser: user found in context", "user_id", user.ID, "email", user.Email)

	u, err := h.Service.GetByID(user.ID)
	if err != nil {
		h.Logger.Error("GetCurrentUser: service GetByID failed", "user_id", user.ID, "error", err)
		h.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.Logger.Info("GetCurrentUser: service returned user", "user_id", u.ID, "email", u.Email, "name", u.Name)

	h.Logger.Info("GetCurrentUser: sending response", "user_id", u.ID, "email", u.Email, "name", u.Name)

	h.WriteJSON(w, http.StatusOK, u)
}
