package commands

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"go-boilerplate/internal/auth/application"
	"go-boilerplate/internal/auth/domain"
	"go-boilerplate/internal/golibs/crypt"

	"go.uber.org/zap"
)

type RegisterCommand struct {
	Email    string
	Password string
	Name     string
}

type LoginCommand struct {
	Email    string
	Password string
}

type AuthResult struct {
	User *domain.User
	application.TokenPair
}

type TokenTTL struct {
	Access  time.Duration
	Refresh time.Duration
}

// AuthCommandHandler handles every use case that issues tokens.
type AuthCommandHandler struct {
	Users   application.UserRepository
	Refresh application.RefreshTokenStore
	Signer  application.TokenSigner
	Events  application.EventPublisher // optional
	TTL     TokenTTL
	Logger  *zap.Logger
	Now     func() time.Time // injectable for tests
}

func (h *AuthCommandHandler) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

func (h *AuthCommandHandler) Register(ctx context.Context, cmd RegisterCommand) (*AuthResult, error) {
	if err := domain.ValidateRegistration(cmd.Email, cmd.Password, cmd.Name); err != nil {
		return nil, err
	}

	hash, err := crypt.HashPassword(cmd.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	user := domain.NewUser(cmd.Email, hash, cmd.Name, h.now())
	// No pre-check by email: the unique index is the source of truth, so two
	// concurrent registrations cannot both succeed.
	if err := h.Users.Create(ctx, user); err != nil {
		return nil, err
	}

	tokens, err := h.issueTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	if h.Events != nil {
		// Best effort: the account exists either way. A transactional outbox
		// would make this reliable; see docs when events become critical.
		if err := h.Events.PublishUserCreated(ctx, user); err != nil && h.Logger != nil {
			h.Logger.Warn("publish user created", zap.String("user_id", user.ID), zap.Error(err))
		}
	}
	return &AuthResult{User: user, TokenPair: *tokens}, nil
}

func (h *AuthCommandHandler) Login(ctx context.Context, cmd LoginCommand) (*AuthResult, error) {
	user, err := h.Users.GetByEmail(ctx, domain.NormalizeEmail(cmd.Email))
	if errors.Is(err, domain.ErrUserNotFound) {
		return nil, domain.ErrInvalidCredentials // do not reveal which part was wrong
	}
	if err != nil {
		return nil, err
	}
	if !crypt.CheckPasswordHash(cmd.Password, user.PasswordHash) {
		return nil, domain.ErrInvalidCredentials
	}

	tokens, err := h.issueTokens(ctx, user)
	if err != nil {
		return nil, err
	}
	return &AuthResult{User: user, TokenPair: *tokens}, nil
}

// RefreshTokens rotates a refresh token: the old one is consumed atomically,
// so replaying it (or two concurrent refreshes) fails.
func (h *AuthCommandHandler) RefreshTokens(ctx context.Context, refreshToken string) (*application.TokenPair, error) {
	if refreshToken == "" {
		return nil, domain.ErrInvalidToken
	}
	userID, err := h.Refresh.Consume(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	user, err := h.Users.GetByID(ctx, userID)
	if errors.Is(err, domain.ErrUserNotFound) {
		return nil, domain.ErrInvalidToken
	}
	if err != nil {
		return nil, err
	}
	return h.issueTokens(ctx, user)
}

func (h *AuthCommandHandler) issueTokens(ctx context.Context, user *domain.User) (*application.TokenPair, error) {
	access, err := h.Signer.Sign(user.ID, user.Role, h.TTL.Access)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	refresh := hex.EncodeToString(b)
	if err := h.Refresh.Save(ctx, refresh, user.ID, h.TTL.Refresh); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}
	return &application.TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}
