package payment

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/frahmantamala/expense-management/internal/core/datamodel/payment"
	paymentgatewaytypes "github.com/frahmantamala/expense-management/internal/core/datamodel/paymentgateway"
	"github.com/frahmantamala/expense-management/internal/paymentgateway"
)

type RepositoryAPI interface {
	Create(p *payment.Payment) error
	GetByID(id int64) (*payment.Payment, error)
	GetByExternalID(externalID string) (*payment.Payment, error)
	GetByExpenseID(expenseID int64) ([]*payment.Payment, error)
	GetLatestByExpenseID(expenseID int64) (*payment.Payment, error)
	UpdateStatus(id int64, status string, paymentMethod *string, gatewayResponse json.RawMessage, failureReason *string) error
	IncrementRetryCount(id int64) error
}

type PaymentService struct {
	logger     *slog.Logger
	repository RepositoryAPI
	gateway    *paymentgateway.Client
}

func NewPaymentService(logger *slog.Logger, repository RepositoryAPI, gateway *paymentgateway.Client) *PaymentService {
	return &PaymentService{
		logger:     logger,
		repository: repository,
		gateway:    gateway,
	}
}

func (s *PaymentService) CreatePayment(expenseID int64, externalID string, amountIDR int64) (*payment.Payment, error) {
	// Check if external_id already exists for idempotency
	existingPayment, err := s.repository.GetByExternalID(externalID)
	if err == nil && existingPayment != nil {
		s.logger.Warn("external_id already exists, rejecting duplicate payment creation",
			"external_id", externalID,
			"existing_payment_id", existingPayment.ID,
			"existing_expense_id", existingPayment.ExpenseID,
			"new_expense_id", expenseID)
		return nil, fmt.Errorf("external_id %s already exists", externalID)
	}

	paymentEntity := NewPayment(expenseID, externalID, amountIDR)

	err = s.repository.Create(paymentEntity)
	if err != nil {
		s.logger.Error("failed to create payment record", "error", err, "expense_id", expenseID)
		return nil, fmt.Errorf("failed to create payment record: %w", err)
	}

	s.logger.Info("payment record created", "payment_id", paymentEntity.ID, "expense_id", expenseID, "external_id", externalID)
	return paymentEntity, nil
}

func (s *PaymentService) ProcessPayment(req *PaymentRequest) (*PaymentResponse, error) {

	paymentRecord, err := s.repository.GetByExternalID(req.ExternalID)
	if err != nil {
		s.logger.Error("payment record not found", "external_id", req.ExternalID, "error", err)
		return nil, fmt.Errorf("payment record not found: %w", err)
	}

	gatewayReq := &paymentgatewaytypes.PaymentRequest{
		ExternalID: req.ExternalID,
		Amount:     req.Amount,
		Currency:   "IDR",
	}

	gatewayResp, err := s.gateway.ProcessPayment(gatewayReq)
	if err != nil {
		s.logger.Error("payment gateway error", "error", err, "external_id", req.ExternalID)

		failureReason := err.Error()
		updateErr := s.repository.UpdateStatus(paymentRecord.ID, StatusFailed, nil, nil, &failureReason)
		if updateErr != nil {
			s.logger.Error("failed to update payment status after gateway error", "error", updateErr, "payment_id", paymentRecord.ID)
		}

		return nil, fmt.Errorf("payment processing failed: %w", err)
	}

	status := MapExternalStatus(string(gatewayResp.Data.Status))
	respBody, _ := json.Marshal(gatewayResp)

	paymentResp := &PaymentResponse{
		Data: PaymentData{
			ID:         gatewayResp.Data.ID,
			ExternalID: gatewayResp.Data.ExternalID,
			Status:     status,
		},
	}

	err = s.repository.UpdateStatus(paymentRecord.ID, status, nil, respBody, nil)
	if err != nil {
		s.logger.Error("failed to update payment status", "error", err, "payment_id", paymentRecord.ID)
	}

	s.logger.Info("payment successfully",
		"payment_id", paymentResp.Data.ID,
		"external_id", paymentResp.Data.ExternalID,
		"status", paymentResp.Data.Status)

	return paymentResp, nil
}

func (s *PaymentService) RetryPayment(req *PaymentRequest) (*PaymentResponse, error) {
	s.logger.Info("retrying payment", "external_id", req.ExternalID, "amount", req.Amount)

	payment, err := s.repository.GetByExternalID(req.ExternalID)
	if err != nil {
		s.logger.Error("payment record not found for retry", "external_id", req.ExternalID, "error", err)
		return nil, fmt.Errorf("payment record not found: %w", err)
	}

	err = s.repository.IncrementRetryCount(payment.ID)
	if err != nil {
		s.logger.Error("failed to increment retry count", "error", err, "payment_id", payment.ID)
	}

	return s.ProcessPayment(req)
}

func (s *PaymentService) GetPaymentByExpenseID(expenseID int64) (*payment.Payment, error) {
	return s.repository.GetLatestByExpenseID(expenseID)
}

func (s *PaymentService) GetPaymentByExternalID(externalID string) (*payment.Payment, error) {
	return s.repository.GetByExternalID(externalID)
}

func (s *PaymentService) UpdatePaymentStatus(paymentID int64, status string, paymentMethod *string, gatewayResponse json.RawMessage, failureReason *string) error {
	return s.repository.UpdateStatus(paymentID, status, paymentMethod, gatewayResponse, failureReason)
}
