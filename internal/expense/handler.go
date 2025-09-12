package expense

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/frahmantamala/expense-management/internal/auth"
	"github.com/frahmantamala/expense-management/internal/transport"
	"github.com/frahmantamala/expense-management/internal/transport/middleware"
	"github.com/frahmantamala/expense-management/pkg/logger"
	"github.com/go-chi/chi"
)

// Handler handles HTTP requests for expenses
type Handler struct {
	*transport.BaseHandler
	Service *Service
}

// NewHandler creates a new expense handler
func NewHandler(service *Service) *Handler {
	lg := logger.LoggerWrapper()
	if lg == nil {
		lg = slog.Default()
	}
	return &Handler{
		BaseHandler: transport.NewBaseHandler(lg),
		Service:     service,
	}
}

// CreateExpense handles POST /expenses
func (h *Handler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok || user == nil {
		h.Logger.Error("CreateExpense: user not found in context")
		h.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var dto CreateExpenseDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		h.Logger.Error("CreateExpense: invalid request body", "error", err)
		h.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	expense, err := h.Service.CreateExpense(user.ID, dto)
	if err != nil {
		h.Logger.Error("CreateExpense: service error", "error", err, "user_id", user.ID)

		// Check if it's a validation error
		if err.Error() == "amount must be positive" ||
			err.Error() == "amount must be at least 10,000 IDR" ||
			err.Error() == "amount must not exceed 50,000,000 IDR" ||
			err.Error() == "description is required" ||
			err.Error() == "category is required" ||
			err.Error() == "expense date is required" ||
			err.Error() == "expense date cannot be in the future" ||
			err.Error() == "description must be less than 500 characters" {
			h.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		h.WriteError(w, http.StatusInternalServerError, "failed to create expense")
		return
	}

	h.Logger.Info("CreateExpense: expense created successfully",
		"expense_id", expense.ID,
		"user_id", user.ID,
		"amount", expense.AmountIDR,
		"status", expense.ExpenseStatus)

	h.WriteJSON(w, http.StatusCreated, expense)
}

// GetExpense handles GET /expenses/:id
func (h *Handler) GetExpense(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok || user == nil {
		h.Logger.Error("GetExpense: user not found in context")
		h.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	expenseIDStr := chi.URLParam(r, "id")
	expenseID, err := strconv.ParseInt(expenseIDStr, 10, 64)
	if err != nil {
		h.Logger.Error("GetExpense: invalid expense ID", "id", expenseIDStr)
		h.WriteError(w, http.StatusBadRequest, "invalid expense ID")
		return
	}

	// Check if user has manager permissions using the middleware helper
	isManager := middleware.HasManagerPermissions(user)

	expense, err := h.Service.GetExpenseByID(expenseID, user.ID, isManager)
	if err != nil {
		h.Logger.Error("GetExpense: service error", "error", err, "expense_id", expenseID, "user_id", user.ID)

		switch err {
		case ErrExpenseNotFound:
			h.WriteError(w, http.StatusNotFound, "expense not found")
		case ErrUnauthorizedAccess:
			h.WriteError(w, http.StatusForbidden, "access denied")
		default:
			h.WriteError(w, http.StatusInternalServerError, "failed to get expense")
		}
		return
	}

	h.WriteJSON(w, http.StatusOK, expense)
}

// GetUserExpenses handles GET /expenses (user's own expenses)
func (h *Handler) GetUserExpenses(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok || user == nil {
		h.Logger.Error("GetUserExpenses: user not found in context")
		h.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse pagination parameters
	limit := 20 // default
	offset := 0 // default

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	expenses, err := h.Service.GetUserExpenses(user.ID, limit, offset)
	if err != nil {
		h.Logger.Error("GetUserExpenses: service error", "error", err, "user_id", user.ID)
		h.WriteError(w, http.StatusInternalServerError, "failed to get expenses")
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"expenses": expenses,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetPendingApprovals handles GET /expenses/pending (manager only)
func (h *Handler) GetPendingApprovals(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok || user == nil {
		h.Logger.Error("GetPendingApprovals: user not found in context")
		h.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse pagination parameters
	limit := 20 // default
	offset := 0 // default

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	expenses, err := h.Service.GetPendingApprovals(limit, offset, user.Permissions)
	if err != nil {
		h.Logger.Error("GetPendingApprovals: service error", "error", err, "user_id", user.ID)

		switch err {
		case ErrUnauthorizedAccess:
			h.WriteError(w, http.StatusForbidden, "manager access required")
		default:
			h.WriteError(w, http.StatusInternalServerError, "failed to get pending approvals")
		}
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"expenses": expenses,
		"limit":    limit,
		"offset":   offset,
	})
}

// ApproveExpense handles PATCH /expenses/:id/approve (manager only)
func (h *Handler) ApproveExpense(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok || user == nil {
		h.Logger.Error("ApproveExpense: user not found in context")
		h.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	expenseIDStr := chi.URLParam(r, "id")
	expenseID, err := strconv.ParseInt(expenseIDStr, 10, 64)
	if err != nil {
		h.Logger.Error("ApproveExpense: invalid expense ID", "id", expenseIDStr)
		h.WriteError(w, http.StatusBadRequest, "invalid expense ID")
		return
	}

	if err := h.Service.ApproveExpense(expenseID, user.ID, user.Permissions); err != nil {
		h.Logger.Error("ApproveExpense: service error", "error", err, "expense_id", expenseID, "manager_id", user.ID)

		switch err {
		case ErrExpenseNotFound:
			h.WriteError(w, http.StatusNotFound, "expense not found")
		case ErrInvalidExpenseStatus:
			h.WriteError(w, http.StatusBadRequest, "expense cannot be approved in current status")
		case ErrUnauthorizedAccess:
			h.WriteError(w, http.StatusForbidden, "manager access required")
		default:
			h.WriteError(w, http.StatusInternalServerError, "failed to approve expense")
		}
		return
	}

	h.Logger.Info("ApproveExpense: expense approved successfully", "expense_id", expenseID, "manager_id", user.ID)
	h.WriteJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

// RejectExpense handles PATCH /expenses/:id/reject (manager only)
func (h *Handler) RejectExpense(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok || user == nil {
		h.Logger.Error("RejectExpense: user not found in context")
		h.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	expenseIDStr := chi.URLParam(r, "id")
	expenseID, err := strconv.ParseInt(expenseIDStr, 10, 64)
	if err != nil {
		h.Logger.Error("RejectExpense: invalid expense ID", "id", expenseIDStr)
		h.WriteError(w, http.StatusBadRequest, "invalid expense ID")
		return
	}

	var dto RejectExpenseDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		h.Logger.Error("RejectExpense: invalid request body", "error", err)
		h.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate the DTO
	if err := dto.Validate(); err != nil {
		h.Logger.Error("RejectExpense: validation error", "error", err)
		h.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.Service.RejectExpense(expenseID, user.ID, dto.Reason, user.Permissions); err != nil {
		h.Logger.Error("RejectExpense: service error", "error", err, "expense_id", expenseID, "manager_id", user.ID)

		switch err {
		case ErrExpenseNotFound:
			h.WriteError(w, http.StatusNotFound, "expense not found")
		case ErrInvalidExpenseStatus:
			h.WriteError(w, http.StatusBadRequest, "expense cannot be rejected in current status")
		case ErrUnauthorizedAccess:
			h.WriteError(w, http.StatusForbidden, "manager access required")
		default:
			h.WriteError(w, http.StatusInternalServerError, "failed to reject expense")
		}
		return
	}

	h.Logger.Info("RejectExpense: expense rejected successfully",
		"expense_id", expenseID,
		"manager_id", user.ID,
		"reason", dto.Reason)

	h.WriteJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}
