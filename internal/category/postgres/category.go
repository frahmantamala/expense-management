package postgres

import (
	"github.com/frahmantamala/expense-management/internal/category"
	"gorm.io/gorm"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) category.Repository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) GetAll() ([]*category.Category, error) {
	var categories []*category.Category
	err := r.db.Order("name ASC").Find(&categories).Error
	return categories, err
}

func (r *CategoryRepository) GetByName(name string) (*category.Category, error) {
	var cat category.Category
	err := r.db.Where("name = ?", name).First(&cat).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &cat, nil
}

func (r *CategoryRepository) Create(cat *category.Category) error {
	return r.db.Create(cat).Error
}

func (r *CategoryRepository) Update(cat *category.Category) error {
	return r.db.Save(cat).Error
}

func (r *CategoryRepository) Delete(id int64) error {
	return r.db.Model(&category.Category{}).Where("id = ?", id).Update("is_active", false).Error
}
