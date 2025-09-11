package payment

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/frahmantamala/expense-management/internal/auth"
	"github.com/frahmantamala/expense-management/internal/expense"
	"github.com/frahmantamala/expense-management/internal/transport"
	"github.com/go-chi/chi"
)

type ExpenseService interface {
	RetryPayment(expenseID int64, userPermissions []string) error
}

type Handler struct {
	transport.BaseHandler
	ExpenseService ExpenseService
	Logger         *slog.Logger
}

func NewHandler(expenseService ExpenseService, logger *slog.Logger) *Handler {
	return &Handler{
		ExpenseService: expenseService,
		Logger:         logger,
	}
}

func (h *Handler) RetryPayment(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok || user == nil {
		h.Logger.Error("RetryPayment: user not found in context")
		h.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	expenseIDStr := chi.URLParam(r, "id")
	expenseID, err := strconv.ParseInt(expenseIDStr, 10, 64)
	if err != nil {
		h.Logger.Error("RetryPayment: invalid expense ID", "id", expenseIDStr)
		h.WriteError(w, http.StatusBadRequest, "invalid expense ID")
		return
	}

	if err := h.ExpenseService.RetryPayment(expenseID, user.Permissions); err != nil {
		h.Logger.Error("RetryPayment: service error", "error", err, "expense_id", expenseID, "user_id", user.ID)

		switch err {
		case expense.ErrExpenseNotFound:
			h.WriteError(w, http.StatusNotFound, "expense not found")
		case expense.ErrInvalidExpenseStatus:
			h.WriteError(w, http.StatusBadRequest, "payment retry not allowed for current expense status")
		case expense.ErrUnauthorizedAccess:
			h.WriteError(w, http.StatusForbidden, "manager access required")
		default:
			h.WriteError(w, http.StatusInternalServerError, "failed to retry payment")
		}
		return
	}

	h.Logger.Info("RetryPayment: payment retry initiated", "expense_id", expenseID, "user_id", user.ID)
	h.WriteJSON(w, http.StatusOK, map[string]string{"status": "payment retry initiated"})
}
