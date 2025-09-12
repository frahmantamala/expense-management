package category

import (
	"time"
)

type Category struct {
	ID          int64     `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"uniqueIndex;not null"`
	Description string    `json:"description" gorm:"not null"`
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Category) TableName() string {
	return "expense_categories"
}

type CategoryResponse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CategoriesResponse struct {
	Categories []CategoryResponse `json:"categories"`
}

func (c *Category) ToResponse() CategoryResponse {
	return CategoryResponse{
		Name:        c.Name,
		Description: c.Description,
	}
}
