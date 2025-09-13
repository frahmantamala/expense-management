package expense

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/frahmantamala/expense-management/internal"
	"github.com/frahmantamala/expense-management/internal/transport"
	"github.com/frahmantamala/expense-management/pkg/logger"
	"github.com/go-chi/chi"
)

type ServiceAPI interface {
	CreateExpense(req *CreateExpenseDTO, userID int64) (*Expense, error)
	GetExpenseByID(expenseID int64, userID int64, userPermissions []string) (*Expense, error)
	GetExpensesForUser(userID int64, userPermissions []string, params *ExpenseQueryParams) ([]*Expense, error)
	UpdateExpenseStatus(expenseID int64, status string, userID int64, userPermissions []string) (*Expense, error)
	SubmitExpenseForApproval(expenseID int64, userID int64, userPermissions []string) (*Expense, error)
	ApproveExpense(expenseID int64, managerID int64, userPermissions []string) error
	RejectExpense(expenseID int64, managerID int64, reason string, userPermissions []string) error
	RetryPayment(expenseID int64, userPermissions []string) error
}

type Handler struct {
	*transport.BaseHandler
	Service ServiceAPI
}

func NewHandler(service ServiceAPI) *Handler {
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
	user, ok := internal.UserFromContext(r.Context())
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

	expense, err := h.Service.CreateExpense(&dto, user.ID)
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
	user, ok := internal.UserFromContext(r.Context())
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

	expense, err := h.Service.GetExpenseByID(expenseID, user.ID, user.Permissions)
	if err != nil {
		h.Logger.Error("GetExpense: service error", "error", err, "expense_id", expenseID, "user_id", user.ID)
		h.HandleServiceError(w, err)
		return
	}

	h.WriteJSON(w, http.StatusOK, expense)
}

// GetAllExpenses retrieves all expenses for managers/admins
func (h *Handler) GetAllExpenses(w http.ResponseWriter, r *http.Request) {
	user, ok := internal.UserFromContext(r.Context())
	if !ok || user == nil {
		h.Logger.Error("GetAllExpenses: user not found in context")
		h.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse query parameters
	params := &ExpenseQueryParams{}

	// Per page (renamed from limit)
	if perPageStr := r.URL.Query().Get("per_page"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= 100 {
			params.PerPage = pp
		}
	}

	// Page (calculate from offset for backwards compatibility)
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			params.Page = p
		}
	} else if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		// Backwards compatibility: convert offset to page
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			params.Page = (offset / params.PerPage) + 1
		}
	}

	// Search
	params.Search = r.URL.Query().Get("search")

	// Category filter
	params.CategoryID = r.URL.Query().Get("category_id")

	// Status filter
	params.Status = r.URL.Query().Get("status")

	// Sort parameters
	params.SortBy = r.URL.Query().Get("sort_by")
	params.SortOrder = r.URL.Query().Get("sort_order")

	// Set defaults
	params.SetDefaults()

	// Use the enhanced GetExpensesForUser method with query parameters
	expenses, err := h.Service.GetExpensesForUser(user.ID, user.Permissions, params)
	if err != nil {
		h.Logger.Error("GetAllExpenses: service error", "error", err, "user_id", user.ID)
		h.WriteError(w, http.StatusInternalServerError, "failed to retrieve expenses")
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"expenses":    expenses,
		"per_page":    params.PerPage,
		"page":        params.Page,
		"search":      params.Search,
		"category_id": params.CategoryID,
		"status":      params.Status,
		"sort_by":     params.SortBy,
		"sort_order":  params.SortOrder,
	})
}

func (h *Handler) ApproveExpense(w http.ResponseWriter, r *http.Request) {
	user, ok := internal.UserFromContext(r.Context())
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
	user, ok := internal.UserFromContext(r.Context())
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
