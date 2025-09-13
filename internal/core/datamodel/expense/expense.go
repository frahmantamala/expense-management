package expense

import "time"

type Expense struct {
	ID              int64
	UserID          int64
	AmountIDR       int64
	Description     string
	Category        string
	ReceiptURL      *string
	ReceiptFileName *string
	ExpenseStatus   string
	ExpenseDate     time.Time
	SubmittedAt     time.Time
	ProcessedAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ExpenseCategory struct {
	ID          int64
	Name        string
	Description string
	IsActive    bool
	CreatedAt   time.Time
}
