package usecase

import (
	"context"
	"fmt"
	"go-boilerplate/internal/auth/domain"
)

type AuthUsecase struct {
	repo domain.UserRepository
}

func NewAuthUsecase(repo domain.UserRepository) *AuthUsecase {
	return &AuthUsecase{repo: repo}
}

func (u *AuthUsecase) GetProfile(ctx context.Context, userID string) (*domain.User, error) {
	if u.repo == nil {
		return &domain.User{ID: userID, Name: "Demo User", Email: "demo@example.com"}, nil
	}
	user, err := u.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth_usecase GetProfile: %w", err)
	}
	return user, nil
}
