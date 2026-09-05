package auth

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"go-boilerplate/internal/auth/usecase"
	"go-boilerplate/internal/golibs/kafka"
	userv1 "go-boilerplate/pkg/genproto/user/v1"
)

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

func (a *App) RegisterHTTP(router *gin.Engine) {
	router.GET("/v1/auth/me", func(c *gin.Context) {
		profile, err := a.authUsecase.GetProfile(c.Request.Context(), "user_123")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, profile)
	})
}

func (a *App) RegisterConsumers(ctx context.Context, km kafka.KafkaManagement) error {
	return nil
}
