package expense

import "time"

type Expense struct {
	ID              int64      `gorm:"primaryKey"`
	UserID          int64      `gorm:"column:user_id;not null"`
	AmountIDR       int64      `gorm:"column:amount_idr;not null"`
	Description     string     `gorm:"not null"`
	Category        string     `gorm:"column:category"`
	ReceiptURL      *string    `gorm:"column:receipt_url"`
	ReceiptFileName *string    `gorm:"column:receipt_filename"`
	ExpenseStatus   string     `gorm:"column:expense_status;default:pending_approval"`
	ExpenseDate     time.Time  `gorm:"column:expense_date;type:date"`
	SubmittedAt     time.Time  `gorm:"column:submitted_at"`
	ProcessedAt     *time.Time `gorm:"column:processed_at"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

type ExpenseCategory struct {
	ID          int64     `gorm:"primaryKey"`
	Name        string    `gorm:"column:name;not null"`
	Description string    `gorm:"column:description"`
	IsActive    bool      `gorm:"column:is_active;default:true"`
	CreatedAt   time.Time `gorm:"column:created_at;default:now()"`
}
