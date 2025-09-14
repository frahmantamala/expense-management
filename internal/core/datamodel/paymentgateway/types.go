package paymentgateway

import (
	"errors"
)

type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "PENDING"
	PaymentStatusSuccess PaymentStatus = "SUCCESS"
	PaymentStatusFailed  PaymentStatus = "FAILED"
)

type PaymentRequest struct {
	ExternalID string `json:"external_id"`
	Amount     int64  `json:"amount"`
	Currency   string `json:"currency"`
}

func (r *PaymentRequest) Validate() error {
	if r.ExternalID == "" {
		return errors.New("external_id is required")
	}
	if r.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}
	if r.Currency == "" {
		return errors.New("currency is required")
	}
	return nil
}

type PaymentData struct {
	ID         string        `json:"id"`
	ExternalID string        `json:"external_id"`
	Status     PaymentStatus `json:"status"`
}

type PaymentResponse struct {
	Data PaymentData `json:"data"`
}
