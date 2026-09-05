package usecase

import (
	"context"
	"fmt"
	"time"

	"go-boilerplate/internal/auth/domain"
	"go-boilerplate/internal/golibs/crypt"

	"github.com/google/uuid"
)

type AuthUsecase struct {
	userRepo  domain.UserRepository
	tokenRepo domain.TokenRepository
	jwtMgr    domain.TokenManager
	eventPub  domain.EventPublisher
}

func NewAuthUsecase(
	userRepo domain.UserRepository,
	tokenRepo domain.TokenRepository,
	jwtMgr domain.TokenManager,
	eventPub domain.EventPublisher,
) *AuthUsecase {
	return &AuthUsecase{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		jwtMgr:    jwtMgr,
		eventPub:  eventPub,
	}
}

func (u *AuthUsecase) Register(ctx context.Context, email, password, name string) (*domain.User, string, string, error) {
	if existing, _ := u.userRepo.GetByEmail(ctx, email); existing != nil {
		return nil, "", "", domain.ErrEmailAlreadyExists
	}

	hashedPassword, err := crypt.HashPassword(password)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to hash password: %w", err)
	}

	user := &domain.User{
		ID:           uuid.New().String(),
		Email:        email,
		PasswordHash: hashedPassword,
		Name:         name,
		Role:         "user",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		return nil, "", "", err
	}

	accessToken, refreshToken, err := u.generateTokenPair(ctx, user.ID, user.Role)
	if err != nil {
		return nil, "", "", err
	}

	if u.eventPub != nil {
		_ = u.eventPub.PublishUserCreated(ctx, user)
	}

	return user, accessToken, refreshToken, nil
}

func (u *AuthUsecase) Login(ctx context.Context, email, password string) (*domain.User, string, string, error) {
	user, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil || user == nil {
		return nil, "", "", domain.ErrInvalidCredentials
	}

	if !crypt.CheckPasswordHash(password, user.PasswordHash) {
		return nil, "", "", domain.ErrInvalidCredentials
	}

	accessToken, refreshToken, err := u.generateTokenPair(ctx, user.ID, user.Role)
	if err != nil {
		return nil, "", "", err
	}

	return user, accessToken, refreshToken, nil
}

func (u *AuthUsecase) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	userID, err := u.tokenRepo.GetRefreshToken(ctx, refreshToken)
	if err != nil || userID == "" {
		return "", "", domain.ErrInvalidToken
	}

	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", "", domain.ErrInvalidToken
	}

	_ = u.tokenRepo.DeleteRefreshToken(ctx, refreshToken)

	return u.generateTokenPair(ctx, user.ID, user.Role)
}

func (u *AuthUsecase) ValidateToken(ctx context.Context, accessToken string) (string, string, error) {
	return u.jwtMgr.ValidateAccessToken(accessToken)
}

func (u *AuthUsecase) GetProfile(ctx context.Context, userID string) (*domain.User, error) {
	return u.userRepo.GetByID(ctx, userID)
}

func (u *AuthUsecase) generateTokenPair(ctx context.Context, userID, role string) (string, string, error) {
	accessToken, err := u.jwtMgr.GenerateAccessToken(userID, role, 15*time.Minute)
	if err != nil {
		return "", "", err
	}

	refreshToken, err := u.jwtMgr.GenerateRefreshToken()
	if err != nil {
		return "", "", err
	}

	if err := u.tokenRepo.StoreRefreshToken(ctx, userID, refreshToken, 7*24*time.Hour); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}
