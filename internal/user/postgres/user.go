package user

// import (
// 	"context"
// 	"database/sql"
// 	"fmt"

// 	"github.com/jmoiron/sqlx"
// )

// type Repository interface {
// 	HasPermission(ctx context.Context, userID int64, permission string) (bool, error)
// 	GetByID(ctx context.Context, id int64) (*User, error)
// }

// func NewPostgresRepo(db *sqlx.DB) Repository {
// 	return &pgRepo{db: db}
// }

// type pgRepo struct {
// 	db *sqlx.DB
// }

// func (p *pgRepo) HasPermission(ctx context.Context, userID int64, permission string) (bool, error) {
// 	var exists bool
// 	query := `
// SELECT EXISTS(
//   SELECT 1 FROM user_permissions up
//   JOIN permissions p ON up.permission_id = p.id
//   WHERE up.user_id = $1 AND p.name = $2
// )
// `
// 	if err := p.db.GetContext(ctx, &exists, query, userID, permission); err != nil {
// 		return false, fmt.Errorf("haspermission query: %w", err)
// 	}
// 	return exists, nil
// }

// func (p *pgRepo) GetByID(ctx context.Context, id int64) (*User, error) {
// 	var u User
// 	if err := p.db.GetContext(ctx, &u, "SELECT id, email, name, department, is_active, created_at, updated_at FROM users WHERE id = $1", id); err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, sql.ErrNoRows
// 		}
// 		return nil, err
// 	}
// 	return &u, nil
// }
