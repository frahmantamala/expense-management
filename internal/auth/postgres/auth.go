package auth

import (
	"database/sql"
	"fmt"

	"github.com/frahmantamala/expense-management/internal/auth"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) GetPasswordForUsername(email string) (string, string, error) {
	var passwordHash string
	var userID string
	query := `SELECT id, password_hash FROM users WHERE email = ? AND is_active = true`

	row := r.db.Raw(query, email).Row()
	if err := row.Scan(&userID, &passwordHash); err != nil {
		if err == sql.ErrNoRows {
			return "", "", fmt.Errorf("user not found")
		}
		return "", "", err
	}
	return passwordHash, userID, nil
}

func (r *Repository) GetUserWithPermissions(userID int64) (*auth.User, error) {
	var user auth.User

	// Get user basic info - only the fields available in auth.User
	query := `SELECT id, email FROM users WHERE id = ? AND is_active = true`

	row := r.db.Raw(query, userID).Row()
	if err := row.Scan(&user.ID, &user.Email); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	// Get user permissions
	permQuery := `SELECT p.name 
	             FROM permissions p 
	             JOIN user_permissions up ON p.id = up.permission_id 
	             WHERE up.user_id = ?`

	rows, err := r.db.Raw(permQuery, userID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []string
	for rows.Next() {
		var permName string
		if err := rows.Scan(&permName); err != nil {
			return nil, err
		}
		permissions = append(permissions, permName)
	}

	user.Permissions = permissions
	return &user, nil
}
