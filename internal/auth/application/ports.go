// Package application holds the auth use cases, split into commands (state
// changes) and queries (reads). Infrastructure implements the ports below;
// application code never imports gorm, redis or kafka.
package application

import (
	"context"
	"time"

	"go-boilerplate/internal/auth/domain"
	"go-boilerplate/internal/golibs/jwtutil"
)

type UserRepository interface {
	// Create assigns user.ID. Returns domain.ErrEmailAlreadyExists on conflict.
	Create(ctx context.Context, user *domain.User) error
	// GetByID / GetByEmail return domain.ErrUserNotFound when absent.
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

type RefreshTokenStore interface {
	Save(ctx context.Context, token, userID string, ttl time.Duration) error
	// Consume atomically returns and deletes the token (single use), or
	// domain.ErrInvalidToken when it does not exist.
	Consume(ctx context.Context, token string) (userID string, err error)
}

// TokenSigner is satisfied by *jwtutil.HS256.
type TokenSigner interface {
	Sign(userID, role string, ttl time.Duration) (string, error)
	Verify(token string) (*jwtutil.Claims, error)
}

type EventPublisher interface {
	PublishUserCreated(ctx context.Context, user *domain.User) error
}

// TokenPair is what a successful sign-in returns.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}
