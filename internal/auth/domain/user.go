package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
}

type TokenRepository interface {
	StoreRefreshToken(ctx context.Context, userID, refreshToken string, ttl time.Duration) error
	GetRefreshToken(ctx context.Context, refreshToken string) (string, error)
	DeleteRefreshToken(ctx context.Context, refreshToken string) error
}

type TokenManager interface {
	GenerateAccessToken(userID, role string, ttl time.Duration) (string, error)
	ValidateAccessToken(tokenString string) (userID, role string, err error)
	GenerateRefreshToken() (string, error)
}

type EventPublisher interface {
	PublishUserCreated(ctx context.Context, user *User) error
}
