package postgres

import (
	"time"

	expenseDatamodel "github.com/frahmantamala/expense-management/internal/core/datamodel/expense"
	"github.com/frahmantamala/expense-management/internal/expense"
	"gorm.io/gorm"
)

type ExpenseRepository struct {
	db *gorm.DB
}

func NewExpenseRepository(db *gorm.DB) expense.RepositoryAPI {
	return &ExpenseRepository{db: db}
}

func (r *ExpenseRepository) Create(exp *expenseDatamodel.Expense) error {
	return r.db.Create(exp).Error
}

func (r *ExpenseRepository) GetByID(id int64) (*expenseDatamodel.Expense, error) {
	var exp expenseDatamodel.Expense
	err := r.db.Where("id = ?", id).First(&exp).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, expense.ErrExpenseNotFound
		}
		return nil, err
	}
	return &exp, nil
}

func (r *ExpenseRepository) GetByUserID(userID int64, params *expense.ExpenseQueryParams) ([]*expenseDatamodel.Expense, error) {
	var expenses []*expenseDatamodel.Expense
	query := r.db.Model(&expenseDatamodel.Expense{}).Where("user_id = ?", userID)

	// Apply common query filters
	query = r.applyQueryFilters(query, params)

	err := query.Find(&expenses).Error
	return expenses, err
}

func (r *ExpenseRepository) GetAllExpenses(params *expense.ExpenseQueryParams) ([]*expenseDatamodel.Expense, error) {
	var expenses []*expenseDatamodel.Expense
	query := r.db.Model(&expenseDatamodel.Expense{})

	// Apply common query filters
	query = r.applyQueryFilters(query, params)

	err := query.Find(&expenses).Error
	return expenses, err
}

// Helper method to apply common query filters
func (r *ExpenseRepository) applyQueryFilters(query *gorm.DB, params *expense.ExpenseQueryParams) *gorm.DB {
	// Search functionality
	if params.Search != "" {
		searchPattern := "%" + params.Search + "%"
		query = query.Where("description ILIKE ? OR category ILIKE ?", searchPattern, searchPattern)
	}

	// Category filter
	if params.CategoryID != "" {
		query = query.Where("category = ?", params.CategoryID)
	}

	// Status filter
	if params.Status != "" {
		query = query.Where("expense_status = ?", params.Status)
	}

	// Sorting
	orderClause := "created_at DESC" // Default fallback
	switch params.SortBy {
	case "createdAt":
		orderClause = "created_at"
		if params.SortOrder == "desc" {
			orderClause += " DESC"
		} else {
			orderClause += " ASC"
		}
	case "submittedAt":
		orderClause = "submitted_at"
		if params.SortOrder == "desc" {
			orderClause += " DESC"
		} else {
			orderClause += " ASC"
		}
	case "amount":
		orderClause = "amount_idr"
		if params.SortOrder == "desc" {
			orderClause += " DESC"
		} else {
			orderClause += " ASC"
		}
	}

	return query.Order(orderClause).
		Limit(params.PerPage).
		Offset(params.GetOffset())
}

func (r *ExpenseRepository) Update(exp *expenseDatamodel.Expense) error {
	exp.UpdatedAt = time.Now()
	return r.db.Save(exp).Error
}

func (r *ExpenseRepository) UpdateStatus(id int64, status string, processedAt time.Time) error {
	return r.db.Model(&expenseDatamodel.Expense{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"expense_status": status,
			"processed_at":   processedAt,
			"updated_at":     time.Now(),
		}).Error
}

func (r *ExpenseRepository) UpdatePaymentInfo(id int64, paymentStatus, paymentID, paymentExternalID string, paidAt *time.Time) error {
	updates := map[string]interface{}{
		"payment_status":      paymentStatus,
		"payment_id":          paymentID,
		"payment_external_id": paymentExternalID,
		"updated_at":          time.Now(),
	}

	if paidAt != nil {
		updates["paid_at"] = *paidAt
	}

	return r.db.Model(&expenseDatamodel.Expense{}).
		Where("id = ?", id).
		Updates(updates).Error
}
