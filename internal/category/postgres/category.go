package postgres

import (
	"github.com/frahmantamala/expense-management/internal/category"
	categoryDatamodel "github.com/frahmantamala/expense-management/internal/core/datamodel/category"
	"gorm.io/gorm"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) category.RepositoryAPI {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) GetAll() ([]*categoryDatamodel.ExpenseCategory, error) {
	var categories []*categoryDatamodel.ExpenseCategory
	err := r.db.Order("name ASC").Find(&categories).Error
	return categories, err
}

func (r *CategoryRepository) GetByName(name string) (*categoryDatamodel.ExpenseCategory, error) {
	var cat categoryDatamodel.ExpenseCategory
	err := r.db.Where("name = ?", name).First(&cat).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &cat, nil
}

func (r *CategoryRepository) GetByID(id int64) (*categoryDatamodel.ExpenseCategory, error) {
	var cat categoryDatamodel.ExpenseCategory
	err := r.db.Where("id = ?", id).First(&cat).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &cat, nil
}

func (r *CategoryRepository) Create(cat *categoryDatamodel.ExpenseCategory) error {
	return r.db.Create(cat).Error
}

func (r *CategoryRepository) Update(cat *categoryDatamodel.ExpenseCategory) error {
	return r.db.Save(cat).Error
}

func (r *CategoryRepository) Delete(id int64) error {
	return r.db.Model(&categoryDatamodel.ExpenseCategory{}).Where("id = ?", id).Update("is_active", false).Error
}
