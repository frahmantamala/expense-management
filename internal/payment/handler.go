package payment

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	errors "github.com/frahmantamala/expense-management/internal"
	"github.com/frahmantamala/expense-management/internal/auth"
	"github.com/frahmantamala/expense-management/internal/transport"
)

type ExpenseServiceAPI interface {
	RetryPayment(expenseID int64, userPermissions []string) error
}

type Handler struct {
	transport.BaseHandler
	ExpenseService ExpenseServiceAPI
	PaymentService ServiceAPI
	Logger         *slog.Logger
}

func NewHandler(expenseService ExpenseServiceAPI, paymentService ServiceAPI, logger *slog.Logger) *Handler {
	return &Handler{
		ExpenseService: expenseService,
		PaymentService: paymentService,
		Logger:         logger,
	}
}

// RetryPayment handles POST /api/v1/payment/retry
func (h *Handler) RetryPayment(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok || user == nil {
		h.Logger.Error("RetryPayment: user not found in context")
		h.HandleError(w, errors.NewUnauthorizedError("authentication required", errors.ErrCodeInvalidToken))
		return
	}

	var req PaymentRetryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Error("RetryPayment: failed to parse request body", "error", err)
		h.HandleError(w, errors.NewValidationError("invalid request body", errors.ErrCodeValidationFailed))
		return
	}

	if err := req.Validate(); err != nil {
		h.Logger.Error("RetryPayment: validation error", "error", err)
		h.HandleServiceError(w, err)
		return
	}

	// Parse expense ID
	expenseID, err := strconv.ParseInt(req.ExpenseID, 10, 64)
	if err != nil {
		h.Logger.Error("RetryPayment: invalid expense ID", "expense_id", req.ExpenseID)
		h.HandleError(w, errors.NewValidationError("invalid expense ID", errors.ErrCodeValidationFailed))
		return
	}

	if err := h.ExpenseService.RetryPayment(expenseID, user.Permissions); err != nil {
		h.Logger.Error("RetryPayment: service error", "error", err, "expense_id", expenseID, "external_id", req.ExternalID, "user_id", user.ID)
		h.HandleServiceError(w, err)
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
