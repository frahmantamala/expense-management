package postgres

import (
	"encoding/json"
	"time"

	"github.com/frahmantamala/expense-management/internal/core/datamodel/payment"
	paymentpkg "github.com/frahmantamala/expense-management/internal/payment"
	"gorm.io/gorm"
)

type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) paymentpkg.RepositoryAPI {
	return &PaymentRepository{
		db: db,
	}
}

func (r *PaymentRepository) Create(p *payment.Payment) error {
	return r.db.Create(p).Error
}

func (r *PaymentRepository) GetByID(id int64) (*payment.Payment, error) {
	var p payment.Payment
	err := r.db.First(&p, id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PaymentRepository) GetByExternalID(externalID string) (*payment.Payment, error) {
	var p payment.Payment
	err := r.db.Where("external_id = ?", externalID).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PaymentRepository) GetByExpenseID(expenseID int64) ([]*payment.Payment, error) {
	var payments []*payment.Payment
	err := r.db.Where("expense_id = ?", expenseID).Order("created_at DESC").Find(&payments).Error
	return payments, err
}

func (r *PaymentRepository) GetLatestByExpenseID(expenseID int64) (*payment.Payment, error) {
	var p payment.Payment
	err := r.db.Where("expense_id = ?", expenseID).Order("created_at DESC").First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PaymentRepository) UpdateStatus(id int64, status string, paymentMethod *string, gatewayResponse json.RawMessage, failureReason *string) error {
	updates := map[string]interface{}{
		"status":       status,
		"processed_at": time.Now(),
	}

	if paymentMethod != nil {
		updates["payment_method"] = *paymentMethod
	}

	if gatewayResponse != nil {
		updates["gateway_response"] = gatewayResponse
	}

	if failureReason != nil {
		updates["failure_reason"] = *failureReason
	}

	return r.db.Model(&payment.Payment{}).Where("id = ?", id).Updates(updates).Error
}

func (r *PaymentRepository) IncrementRetryCount(id int64) error {
	return r.db.Model(&payment.Payment{}).Where("id = ?", id).UpdateColumn("retry_count", gorm.Expr("retry_count + 1")).Error
}
