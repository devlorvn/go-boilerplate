package auth

import (
	"context"
	"errors"
	"strings"

	"go-boilerplate/internal/auth/domain"
	"go-boilerplate/internal/auth/usecase"
	"go-boilerplate/internal/golibs/constants"
	"go-boilerplate/internal/golibs/interceptors"
	userv1 "go-boilerplate/pkg/genproto/user/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCHandler struct {
	userv1.UnimplementedUserServiceServer
	usecase *usecase.AuthUsecase
}

func NewGRPCHandler(u *usecase.AuthUsecase) *GRPCHandler {
	return &GRPCHandler{usecase: u}
}

func (h *GRPCHandler) Register(ctx context.Context, req *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	user, accessToken, refreshToken, err := h.usecase.Register(ctx, req.Email, req.Password, req.Name)
	if err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &userv1.RegisterResponse{
		User:         toUserInfo(user),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (h *GRPCHandler) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error) {
	user, accessToken, refreshToken, err := h.usecase.Login(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &userv1.LoginResponse{
		User:         toUserInfo(user),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (h *GRPCHandler) RefreshToken(ctx context.Context, req *userv1.RefreshTokenRequest) (*userv1.RefreshTokenResponse, error) {
	accessToken, refreshToken, err := h.usecase.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	return &userv1.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (h *GRPCHandler) ValidateToken(ctx context.Context, req *userv1.ValidateTokenRequest) (*userv1.ValidateTokenResponse, error) {
	userID, role, err := h.usecase.ValidateToken(ctx, req.AccessToken)
	if err != nil {
		return &userv1.ValidateTokenResponse{Valid: false}, nil
	}

	return &userv1.ValidateTokenResponse{
		Valid:  true,
		UserId: userID,
		Role:   role,
	}, nil
}

// GetProfile returns the caller's own profile. Reading another user's profile
// (req.UserId set to someone else) is only allowed for admins.
func (h *GRPCHandler) GetProfile(ctx context.Context, req *userv1.GetProfileRequest) (*userv1.GetProfileResponse, error) {
	callerID, ok := interceptors.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user in context")
	}
	targetID := callerID
	if req.UserId != "" && req.UserId != callerID {
		role, _ := interceptors.UserRoleFromContext(ctx)
		if !strings.EqualFold(role, constants.RoleAdmin) {
			return nil, status.Error(codes.PermissionDenied, "cannot read another user's profile")
		}
		targetID = req.UserId
	}

	user, err := h.usecase.GetProfile(ctx, targetID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &userv1.GetProfileResponse{
		User: toUserInfo(user),
	}, nil
}

func toUserInfo(u *domain.User) *userv1.UserInfo {
	if u == nil {
		return nil
	}
	return &userv1.UserInfo{
		Id:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Role:      u.Role,
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
