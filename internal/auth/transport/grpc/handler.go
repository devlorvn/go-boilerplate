// Package grpc adapts the auth use cases to user.v1.UserService. Handlers only
// translate protobuf <-> application types; errors are returned as-is and
// mapped to gRPC codes by interceptors.ErrorInterceptor.
package grpc

import (
	"context"
	"time"

	"go-boilerplate/internal/auth/application/commands"
	"go-boilerplate/internal/auth/application/queries"
	"go-boilerplate/internal/auth/domain"
	"go-boilerplate/internal/golibs/authz"
	"go-boilerplate/internal/golibs/interceptors"
	userv1 "go-boilerplate/pkg/genproto/user/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Policy is the RBAC table of this service. Every RPC must appear here;
// Policy.Validate enforces it at startup.
var Policy = authz.Policy{
	userv1.UserService_Register_FullMethodName:      authz.Public,
	userv1.UserService_Login_FullMethodName:         authz.Public,
	userv1.UserService_RefreshToken_FullMethodName:  authz.Public,
	userv1.UserService_ValidateToken_FullMethodName: authz.Public,
	userv1.UserService_GetProfile_FullMethodName:    authz.Authenticated, // ownership checked in queries
}

type Handler struct {
	userv1.UnimplementedUserServiceServer
	Commands *commands.AuthCommandHandler
	Queries  *queries.UserQueryHandler
}

func (h *Handler) Register(ctx context.Context, req *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	res, err := h.Commands.Register(ctx, commands.RegisterCommand{Email: req.Email, Password: req.Password, Name: req.Name})
	if err != nil {
		return nil, err
	}
	return &userv1.RegisterResponse{User: toUserInfo(res.User), AccessToken: res.AccessToken, RefreshToken: res.RefreshToken}, nil
}

func (h *Handler) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error) {
	res, err := h.Commands.Login(ctx, commands.LoginCommand{Email: req.Email, Password: req.Password})
	if err != nil {
		return nil, err
	}
	return &userv1.LoginResponse{User: toUserInfo(res.User), AccessToken: res.AccessToken, RefreshToken: res.RefreshToken}, nil
}

func (h *Handler) RefreshToken(ctx context.Context, req *userv1.RefreshTokenRequest) (*userv1.RefreshTokenResponse, error) {
	pair, err := h.Commands.RefreshTokens(ctx, req.RefreshToken)
	if err != nil {
		return nil, err
	}
	return &userv1.RefreshTokenResponse{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken}, nil
}

// ValidateToken answers valid=false instead of an error: an invalid token is
// a normal answer for this RPC.
func (h *Handler) ValidateToken(ctx context.Context, req *userv1.ValidateTokenRequest) (*userv1.ValidateTokenResponse, error) {
	actor, err := h.Queries.ValidateToken(ctx, req.AccessToken)
	if err != nil {
		return &userv1.ValidateTokenResponse{Valid: false}, nil
	}
	return &userv1.ValidateTokenResponse{Valid: true, UserId: actor.UserID, Role: actor.Role}, nil
}

func (h *Handler) GetProfile(ctx context.Context, req *userv1.GetProfileRequest) (*userv1.GetProfileResponse, error) {
	actor, ok := interceptors.ActorFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing caller")
	}
	user, err := h.Queries.GetProfile(ctx, actor, req.UserId)
	if err != nil {
		return nil, err
	}
	return &userv1.GetProfileResponse{User: toUserInfo(user)}, nil
}

func toUserInfo(u *domain.User) *userv1.UserInfo {
	return &userv1.UserInfo{
		Id:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Role:      u.Role,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
	}
}
