package category

import "time"

type ExpenseCategory struct {
	ID          int64     `gorm:"primaryKey"`
	Name        string    `gorm:"column:name;uniqueIndex;not null"`
	Description string    `gorm:"column:description"`
	IsActive    bool      `gorm:"column:is_active;default:true"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`
}
