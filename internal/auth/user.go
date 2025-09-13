package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

//go:generate mockgen -destination=./mock/mock_service_api.go -package=mock github.com/frahmantamala/expense-management/internal/auth ServiceAPI
type ServiceAPI interface {
	Authenticate(dto LoginDTO) (AuthTokens, error)
	RefreshTokens(refreshToken string) (AuthTokens, error)
	ValidateAccessToken(tokenString string) (*Claims, error)
	GetUserWithPermissions(userID int64) (*User, error)
	HashPassword(password string) (string, error)
}

//go:generate mockgen -destination=./mock/mock_repository_api.go -package=mock github.com/frahmantamala/expense-management/internal/auth RepositoryAPI
type RepositoryAPI interface {
	GetPasswordForUsername(username string) (passwordHash string, userID string, err error)
	GetUserWithPermissions(userID int64) (*User, error)
}

//go:generate mockgen -destination=./mock/mock_token_generator_api.go -package=mock github.com/frahmantamala/expense-management/internal/auth TokenGeneratorAPI
type TokenGeneratorAPI interface {
	GenerateAccessToken(userID string, email string) (token string, err error)
	GenerateRefreshToken(userID string, email string) (token string, err error)
	ValidateToken(tokenString string) (*Claims, error)
}

// User represents a user entity with business logic
type User struct {
	ID          int64    `json:"id"`
	Email       string   `json:"email"`
	Permissions []string `json:"permissions,omitempty"`
}

// Domain logic methods

// HasPermission checks if user has a specific permission
func (u *User) HasPermission(permission string) bool {
	for _, p := range u.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// HasAnyPermission checks if user has any of the specified permissions
func (u *User) HasAnyPermission(permissions []string) bool {
	for _, userPerm := range u.Permissions {
		for _, requiredPerm := range permissions {
			if userPerm == requiredPerm {
				return true
			}
		}
	}
	return false
}

// IsManager checks if user has manager-level permissions
func (u *User) IsManager() bool {
	managerPerms := []string{"approve_expenses", "reject_expenses", "admin"}
	return u.HasAnyPermission(managerPerms)
}

// IsAdmin checks if user has admin permissions
func (u *User) IsAdmin() bool {
	return u.HasPermission("admin")
}

// AuthInfo is the internal domain model used by services and converters.
type AuthInfo struct {
	UserID    string
	Token     string
	ExpiresAt time.Time
}

type AuthResponseV1 struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

type UserInfo struct {
	ID          int64     `db:"id"`
	Email       string    `db:"email"`
	Name        string    `db:"name"`
	Department  string    `db:"department"`
	IsActive    bool      `db:"is_active"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
	Permissions []string  `db:"-"`
}

type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Claims represents JWT token claims
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

type JWTTokenGenerator struct {
	AccessTokenSecret  []byte
	RefreshTokenSecret []byte
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrUserInactive       = errors.New("user is inactive")
)

func (a AuthInfo) ToV1() AuthResponseV1 {
	return AuthResponseV1{
		ID:    a.UserID,
		Token: a.Token,
	}
}

// Authentication domain methods

// VerifyPassword verifies a password against a hash
func VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// HashPassword hashes a password with the given cost
func HashPassword(password string, cost int) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// GenerateRandomToken generates a random token
func GenerateRandomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
