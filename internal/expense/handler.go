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

type Handler struct {
	*transport.BaseHandler
	Service *Service
}

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
		h.HandleServiceError(w, err)
		return
	}

	h.Logger.Info("CreateExpense: expense created successfully",
		"expense_id", expense.ID,
		"user_id", user.ID,
		"amount", expense.AmountIDR,
		"status", expense.ExpenseStatus)

	h.WriteJSON(w, http.StatusCreated, expense)
}

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

	isManager := middleware.HasManagerPermissions(user)

	expense, err := h.Service.GetExpenseByID(expenseID, user.ID, isManager)
	if err != nil {
		h.Logger.Error("GetExpense: service error", "error", err, "expense_id", expenseID, "user_id", user.ID)
		h.HandleServiceError(w, err)
		return
	}

	h.WriteJSON(w, http.StatusOK, expense)
}

func (h *Handler) GetUserExpenses(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok || user == nil {
		h.Logger.Error("GetUserExpenses: user not found in context")
		h.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := 20
	offset := 0

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

// GetAllExpenses retrieves all expenses for managers/admins
func (h *Handler) GetAllExpenses(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok || user == nil {
		h.Logger.Error("GetAllExpenses: user not found in context")
		h.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := 20
	offset := 0

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

	expenses, err := h.Service.GetAllExpenses(limit, offset, user.Permissions)
	if err != nil {
		h.Logger.Error("GetAllExpenses: service error", "error", err, "user_id", user.ID)

		switch err {
		case ErrUnauthorizedAccess:
			h.WriteError(w, http.StatusForbidden, "manager access required")
		default:
			h.WriteError(w, http.StatusInternalServerError, "failed to get all expenses")
		}
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"expenses": expenses,
		"limit":    limit,
		"offset":   offset,
	})
}

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
