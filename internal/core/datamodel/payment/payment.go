package payment

import (
	"encoding/json"
	"time"
)

type Payment struct {
	ID              int64           `gorm:"primaryKey"`
	ExpenseID       int64           `gorm:"column:expense_id;not null"`
	ExternalID      string          `gorm:"column:external_id;not null;uniqueIndex"`
	AmountIDR       int64           `gorm:"column:amount_idr;not null"`
	Status          string          `gorm:"column:status;default:pending"`
	PaymentMethod   *string         `gorm:"column:payment_method"`
	GatewayResponse json.RawMessage `gorm:"column:gateway_response;type:jsonb"`
	FailureReason   *string         `gorm:"column:failure_reason"`
	RetryCount      int             `gorm:"column:retry_count;default:0"`
	ProcessedAt     *time.Time      `gorm:"column:processed_at"`
	CreatedAt       time.Time       `gorm:"column:created_at;default:now()"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;default:now()"`
}
