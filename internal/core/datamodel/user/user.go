package user

import "time"

type User struct {
	ID           int64     `gorm:"primaryKey"`
	Email        string    `gorm:"column:email;uniqueIndex;not null"`
	Name         string    `gorm:"column:name;not null"`
	PasswordHash string    `gorm:"column:password_hash;not null"`
	Department   string    `gorm:"column:department"`
	IsActive     bool      `gorm:"column:is_active;default:true"`
	CreatedAt    time.Time `gorm:"column:created_at;default:now()"`
	UpdatedAt    time.Time `gorm:"column:updated_at;default:now()"`
}

type Permission struct {
	ID          int64     `gorm:"primaryKey"`
	Name        string    `gorm:"column:name;uniqueIndex;not null"`
	Description string    `gorm:"column:description"`
	CreatedAt   time.Time `gorm:"column:created_at;default:now()"`
}

type UserPermission struct {
	ID           int64     `gorm:"primaryKey"`
	UserID       int64     `gorm:"column:user_id;not null"`
	PermissionID int64     `gorm:"column:permission_id;not null"`
	GrantedBy    *int64    `gorm:"column:granted_by"`
	CreatedAt    time.Time `gorm:"column:created_at;default:now()"`
}
