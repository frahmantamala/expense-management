package postgres

import (
	"time"

	"github.com/frahmantamala/expense-management/internal/expense"
	"gorm.io/gorm"
)

type ExpenseRepository struct {
	db *gorm.DB
}

func NewExpenseRepository(db *gorm.DB) expense.Repository {
	return &ExpenseRepository{db: db}
}

func (r *ExpenseRepository) Create(exp *expense.Expense) error {
	return r.db.Create(exp).Error
}

func (r *ExpenseRepository) GetAllExpenses(limit, offset int) ([]*expense.Expense, error) {
	var expenses []*expense.Expense
	err := r.db.Order("submitted_at DESC").Limit(limit).Offset(offset).Find(&expenses).Error
	return expenses, err
}

func (r *ExpenseRepository) GetByID(id int64) (*expense.Expense, error) {
	var exp expense.Expense
	err := r.db.Where("id = ?", id).First(&exp).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, expense.ErrExpenseNotFound
		}
		return nil, err
	}
	return &exp, nil
}

func (r *ExpenseRepository) GetByUserID(userID int64, limit, offset int) ([]*expense.Expense, error) {
	var expenses []*expense.Expense
	err := r.db.Where("user_id = ?", userID).
		Order("submitted_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&expenses).Error
	return expenses, err
}

func (r *ExpenseRepository) Update(exp *expense.Expense) error {
	exp.UpdatedAt = time.Now()
	return r.db.Save(exp).Error
}

func (r *ExpenseRepository) UpdateStatus(id int64, status string, processedAt time.Time) error {
	return r.db.Model(&expense.Expense{}).
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

	return r.db.Model(&expense.Expense{}).
		Where("id = ?", id).
		Updates(updates).Error
}
