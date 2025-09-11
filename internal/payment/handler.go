package payment

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/frahmantamala/expense-management/internal/auth"
	"github.com/frahmantamala/expense-management/internal/expense"
	"github.com/frahmantamala/expense-management/internal/transport"
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

// RetryPayment handles POST /api/v1/payment/retry
func (h *Handler) RetryPayment(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok || user == nil {
		h.Logger.Error("RetryPayment: user not found in context")
		h.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req PaymentRetryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Error("RetryPayment: failed to parse request body", "error", err)
		h.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		h.Logger.Error("RetryPayment: validation error", "error", err)
		h.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Parse expense ID
	expenseID, err := strconv.ParseInt(req.ExpenseID, 10, 64)
	if err != nil {
		h.Logger.Error("RetryPayment: invalid expense ID", "expense_id", req.ExpenseID)
		h.WriteError(w, http.StatusBadRequest, "invalid expense ID")
		return
	}

	if err := h.ExpenseService.RetryPayment(expenseID, user.Permissions); err != nil {
		h.Logger.Error("RetryPayment: service error", "error", err, "expense_id", expenseID, "external_id", req.ExternalID, "user_id", user.ID)

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

	h.Logger.Info("RetryPayment: payment retry initiated",
		"expense_id", expenseID,
		"external_id", req.ExternalID,
		"user_id", user.ID)

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "payment retry initiated",
		"expense_id":  req.ExpenseID,
		"external_id": req.ExternalID,
	})
}
