package category

import (
	"time"

	categoryDatamodel "github.com/frahmantamala/expense-management/internal/core/datamodel/category"
)

type Category struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (c *Category) IsActiveCategory() bool {
	return c.IsActive
}

func (c *Category) ToResponse() CategoryResponse {
	return CategoryResponse{
		Name:        c.Name,
		Description: c.Description,
	}
}

func (c *Category) Activate() {
	c.IsActive = true
	c.UpdatedAt = time.Now()
}

func (c *Category) Deactivate() {
	c.IsActive = false
	c.UpdatedAt = time.Now()
}

func NewCategory(name, description string) *Category {
	now := time.Now()
	return &Category{
		Name:        name,
		Description: description,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func ToDataModel(c *Category) *categoryDatamodel.ExpenseCategory {
	return &categoryDatamodel.ExpenseCategory{
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		IsActive:    c.IsActive,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

func FromDataModel(c *categoryDatamodel.ExpenseCategory) *Category {
	return &Category{
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		IsActive:    c.IsActive,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}
