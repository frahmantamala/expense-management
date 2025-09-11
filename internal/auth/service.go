package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type UserRepository interface {
	GetPasswordForUsername(username string) (passwordHash string, userID string, err error)
}

// Service is the main auth service with dependencies
type Service struct {
	userRepo       UserRepository
	tokenGenerator TokenGenerator
	bcryptCost     int
}

func NewService(userRepo UserRepository, tokenGen TokenGenerator) *Service {
	return &Service{
		userRepo:       userRepo,
		tokenGenerator: tokenGen,
		bcryptCost:     bcrypt.DefaultCost,
	}
}

func NewJWTTokenGenerator(accessSecret, refreshSecret string) *JWTTokenGenerator {
	return &JWTTokenGenerator{
		AccessTokenSecret:  []byte(accessSecret),
		RefreshTokenSecret: []byte(refreshSecret),
		AccessTokenTTL:     15 * time.Minute,   // Short-lived access token
		RefreshTokenTTL:    24 * 7 * time.Hour, // 7 days refresh token
	}
}

func (s *Service) Authenticate(dto LoginDTO) (AuthTokens, error) {
	// Validate input
	if err := dto.Validate(); err != nil {
		return AuthTokens{}, err
	}

	// Get user credentials
	storedHash, userID, err := s.userRepo.GetPasswordForUsername(dto.Email)
	if err != nil {
		return AuthTokens{}, ErrInvalidCredentials
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(dto.Password)); err != nil {
		return AuthTokens{}, ErrInvalidCredentials
	}

	// Generate tokens with email included
	accessToken, err := s.tokenGenerator.GenerateAccessToken(userID, dto.Email)
	if err != nil {
		return AuthTokens{}, err
	}

	refreshToken, err := s.tokenGenerator.GenerateRefreshToken(userID, dto.Email)
	if err != nil {
		return AuthTokens{}, err
	}

	return AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *Service) RefreshTokens(refreshToken string) (AuthTokens, error) {
	// Validate refresh token
	claims, err := s.tokenGenerator.ValidateToken(refreshToken)
	if err != nil {
		return AuthTokens{}, err
	}

	// Generate new tokens with email from existing claims
	accessToken, err := s.tokenGenerator.GenerateAccessToken(claims.UserID, claims.Email)
	if err != nil {
		return AuthTokens{}, err
	}

	newRefreshToken, err := s.tokenGenerator.GenerateRefreshToken(claims.UserID, claims.Email)
	if err != nil {
		return AuthTokens{}, err
	}

	return AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *Service) ValidateAccessToken(tokenString string) (*Claims, error) {
	return s.tokenGenerator.ValidateToken(tokenString)
}

func (j *JWTTokenGenerator) GenerateAccessToken(userID string, email string) (string, error) {
	expiresAt := time.Now().Add(j.AccessTokenTTL)

	claims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(j.AccessTokenSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (j *JWTTokenGenerator) GenerateRefreshToken(userID string, email string) (string, error) {
	expiresAt := time.Now().Add(j.RefreshTokenTTL)

	claims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(j.RefreshTokenSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (j *JWTTokenGenerator) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Check signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Try access token secret first, then refresh token secret
		if claims, ok := token.Claims.(*Claims); ok {
			// For refresh tokens, use refresh secret
			if time.Until(claims.ExpiresAt.Time) > j.AccessTokenTTL {
				return j.RefreshTokenSecret, nil
			}
		}
		return j.AccessTokenSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func (s *Service) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func GenerateRandomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
