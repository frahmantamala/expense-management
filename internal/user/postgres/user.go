package postgres

import (
	"database/sql"
	"strings"

	userDatamodel "github.com/frahmantamala/expense-management/internal/core/datamodel/user"
	"github.com/frahmantamala/expense-management/internal/user"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) user.RepositoryAPI {
	return &Repository{db: db}
}

func (r *Repository) GetByID(userID int64) (*userDatamodel.User, error) {
	var u userDatamodel.User
	var department sql.NullString

	query := `SELECT id, email, name, department, is_active, password_hash, created_at, updated_at
			  FROM users WHERE id = ? AND is_active = true`

	row := r.db.Raw(query, userID).Row()
	if err := row.Scan(&u.ID, &u.Email, &u.Name, &department, &u.IsActive, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if err == sql.ErrNoRows || err == gorm.ErrRecordNotFound {
			return nil, user.ErrNotFound
		}
		return nil, err
	}

	// Handle nullable department field
	u.Department = department.String

	return &u, nil
}

func (r *Repository) GetPermissions(userID int64) ([]string, error) {
	query := `SELECT p.name
			  FROM permissions p
			  JOIN user_permissions up ON p.id = up.permission_id
			  WHERE up.user_id = ?`

	rows, err := r.db.Raw(query, userID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []string
	for rows.Next() {
		var perm string
		if err := rows.Scan(&perm); err != nil {
			return nil, err
		}
		permissions = append(permissions, strings.TrimSpace(perm))
	}
	return permissions, nil
}
