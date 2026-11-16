package auth

import (
	"context"

	"google.golang.org/grpc"

	"go-boilerplate/internal/auth/usecase"
	"go-boilerplate/internal/golibs/kafka"
	userv1 "go-boilerplate/pkg/genproto/user/v1"
)

// PublicMethods can be called without an access token.
var PublicMethods = []string{
	userv1.UserService_Register_FullMethodName,
	userv1.UserService_Login_FullMethodName,
	userv1.UserService_RefreshToken_FullMethodName,
	userv1.UserService_ValidateToken_FullMethodName,
}

type App struct {
	authUsecase *usecase.AuthUsecase
	grpcHandler *GRPCHandler
}

func NewApp(authUsecase *usecase.AuthUsecase) *App {
	return &App{
		authUsecase: authUsecase,
		grpcHandler: NewGRPCHandler(authUsecase),
	}
}

func (a *App) RegisterGRPC(server *grpc.Server) {
	userv1.RegisterUserServiceServer(server, a.grpcHandler)
}

func (a *App) RegisterConsumers(ctx context.Context, km kafka.KafkaManagement) error {
	return nil
}
