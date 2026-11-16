package main

import (
	"context"
	"fmt"
	"time"

	auth "go-boilerplate/internal/auth/app"
	infras "go-boilerplate/internal/auth/infras"
	"go-boilerplate/internal/auth/usecase"
	"go-boilerplate/internal/golibs/bootstrap"
	"go-boilerplate/internal/golibs/configs"
	"go-boilerplate/internal/golibs/database"
	"go-boilerplate/internal/golibs/interceptors"
	"go-boilerplate/internal/golibs/jwtutil"
	"go-boilerplate/internal/golibs/kafka"
	"go-boilerplate/internal/golibs/redis"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	bootstrap.Run(func(ctx context.Context, cfg *configs.Config, log *zap.Logger) error {
		signer, err := jwtutil.NewHS256(cfg.Auth.JWTSecret)
		if err != nil {
			return fmt.Errorf("AUTH_JWT_SECRET: %w", err)
		}

		db, err := database.NewGORM(cfg.DB, cfg.App.Env == "local")
		if err != nil {
			return fmt.Errorf("connect database: %w", err)
		}

		rdb, err := redis.NewRedisClient(ctx, cfg.Redis, log)
		if err != nil {
			return fmt.Errorf("connect redis: %w", err)
		}
		defer rdb.Close()

		var km kafka.KafkaManagement
		if len(cfg.Kafka.Brokers) > 0 {
			km = kafka.NewKafkaManagement(cfg.Kafka.Brokers, log)
		}

		userRepo, err := infras.NewGORMUserRepository(db)
		if err != nil {
			return fmt.Errorf("init user repository: %w", err)
		}

		tokenRepo := infras.NewRedisTokenStore(rdb)
		jwtMgr := infras.NewJWTManager(signer)
		eventPub := infras.NewKafkaEventPublisher(km)

		authUsecase := usecase.NewAuthUsecase(userRepo, tokenRepo, jwtMgr, eventPub)
		app := auth.NewApp(authUsecase)

		grpcServer := bootstrap.NewServerBootstrap(bootstrap.GRPCServerConfig{
			ServiceName:     cfg.App.Name,
			Port:            fmt.Sprintf(":%d", cfg.GRPC.Port),
			ShutdownTimeout: 5 * time.Second,
		}, log, grpc.ChainUnaryInterceptor(
			interceptors.RecoveryInterceptor(log),
			interceptors.LoggingInterceptor(log),
			interceptors.AuthInterceptor(signer, auth.PublicMethods...),
		))

		if err := grpcServer.Start(app); err != nil {
			return fmt.Errorf("failed to start gRPC server: %w", err)
		}

		bootstrap.StartMonitorServer(cfg.Metrics.Port, db, rdb.Client, log)

		<-ctx.Done()
		log.Info("Shutting down auth service...")

		return grpcServer.Stop(context.Background())
	}, ".env.auth")
}
