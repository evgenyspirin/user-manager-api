package services

import (
	"errors"
	"time"
	"user-manager-api/internal/application/ports"
	"user-manager-api/internal/domain/user"
	"user-manager-api/internal/infrastructure/jwt"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrFailedToGenerateToken = errors.New("failed to generate token")
)

type AuthService struct {
	jwtService *jwt.Service
}

func NewAuthService(
	jwtService *jwt.Service,

) ports.Auth {
	return &AuthService{
		jwtService: jwtService,
	}
}

func (as *AuthService) GenerateToken(u *user.User, requestPassword string) (string, error) {
	err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(requestPassword))
	if err != nil {
		return "", ErrInvalidCredentials
	}

	token, err := as.jwtService.GenerateJWT(u.UUID.String(), u.Role, time.Hour)
	if err != nil {
		return "", ErrFailedToGenerateToken
	}

	return token, nil
}
