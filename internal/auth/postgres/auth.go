package auth

import (
	"database/sql"
	"fmt"

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
