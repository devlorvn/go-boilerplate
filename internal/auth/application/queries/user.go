package queries

import (
	"context"

	"go-boilerplate/internal/auth/application"
	"go-boilerplate/internal/auth/domain"
	"go-boilerplate/internal/golibs/authz"
)

type UserQueryHandler struct {
	Users  application.UserRepository
	Signer application.TokenSigner
}

// GetProfile returns userID's profile, or the actor's own when userID is "".
func (h *UserQueryHandler) GetProfile(ctx context.Context, actor authz.Actor, userID string) (*domain.User, error) {
	if userID == "" {
		userID = actor.UserID
	}
	if !domain.CanReadProfile(actor, userID) {
		return nil, domain.ErrForbidden
	}
	return h.Users.GetByID(ctx, userID)
}

// ValidateToken lets the gateway check a token without knowing the secret.
func (h *UserQueryHandler) ValidateToken(_ context.Context, token string) (authz.Actor, error) {
	claims, err := h.Signer.Verify(token)
	if err != nil {
		return authz.Actor{}, domain.ErrInvalidToken
	}
	return authz.Actor{UserID: claims.UserID, Role: claims.Role}, nil
}
