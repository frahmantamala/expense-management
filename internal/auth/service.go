package auth

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Service struct {
	userRepo          RepositoryAPI
	tokenGenerator    TokenGeneratorAPI
	permissionChecker PermissionChecker
	rbacAuthorization *RBACAuthorization
	bcryptCost        int
	logger            *slog.Logger
}

func NewService(userRepo RepositoryAPI, tokenGen TokenGeneratorAPI, bcryptCost int, logger *slog.Logger) *Service {
	permChecker := NewPermissionChecker()
	return &Service{
		userRepo:          userRepo,
		tokenGenerator:    tokenGen,
		permissionChecker: permChecker,
		rbacAuthorization: NewRBACAuthorization(permChecker.(*DefaultPermissionChecker), logger),
		bcryptCost:        bcryptCost,
		logger:            logger,
	}
}

func NewJWTTokenGenerator(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) *JWTTokenGenerator {
	return &JWTTokenGenerator{
		AccessTokenSecret:  []byte(accessSecret),
		RefreshTokenSecret: []byte(refreshSecret),
		AccessTokenTTL:     accessTTL,
		RefreshTokenTTL:    refreshTTL,
	}
}

func (s *Service) Authenticate(dto LoginDTO) (AuthTokens, error) {
	if err := dto.Validate(); err != nil {
		return AuthTokens{}, err
	}

	storedHash, userID, err := s.userRepo.GetPasswordForUsername(dto.Email)
	if err != nil {
		return AuthTokens{}, ErrInvalidCredentials
	}

	if err := VerifyPassword(storedHash, dto.Password); err != nil {
		return AuthTokens{}, ErrInvalidCredentials
	}

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

	claims, err := s.tokenGenerator.ValidateToken(refreshToken)
	if err != nil {
		return AuthTokens{}, err
	}

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

func (s *Service) GetUserWithPermissions(userID int64) (*User, error) {
	return s.userRepo.GetUserWithPermissions(userID)
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
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		if claims, ok := token.Claims.(*Claims); ok {
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
	return HashPassword(password, s.bcryptCost)
}

func (s *Service) PermissionChecker() PermissionChecker {
	return s.permissionChecker
}

func (s *Service) RBACAuthorization() *RBACAuthorization {
	return s.rbacAuthorization
}
