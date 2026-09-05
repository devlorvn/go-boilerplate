package auth

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"go-boilerplate/internal/auth/usecase"
	"go-boilerplate/internal/golibs/kafka"
)

type App struct {
	authUsecase *usecase.AuthUsecase
}

func NewApp(authUsecase *usecase.AuthUsecase) *App {
	return &App{authUsecase: authUsecase}
}

// 1. Implement ServiceRegistrar cho gRPC
func (a *App) RegisterGRPC(server *grpc.Server) {
	// pb.RegisterAuthServiceServer(server, a.grpcHandler)
}

// 2. Implement HTTPServiceRegistrar cho Gin HTTP
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

// 3. Implement KafkaConsumerRegistrar cho Kafka
func (a *App) RegisterConsumers(ctx context.Context, km kafka.KafkaManagement) error {
	// km.Consume(ctx, "user-created", "auth-group", a.kafkaHandler.HandleUserCreated)
	return nil
}
