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
	"go-boilerplate/internal/golibs/kafka"
	"go-boilerplate/internal/golibs/redis"

	_redis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	bootstrap.Run(func(ctx context.Context, cfg *configs.Config, log *zap.Logger) error {
		db, err := database.NewGORM(cfg.DB, cfg.App.Env == "local")
		if err != nil {
			log.Warn("Database connection failed", zap.Error(err))
		}

		rdb, err := redis.NewRedisClient(ctx, cfg.Redis, log)
		if err != nil {
			log.Warn("Redis connection failed", zap.Error(err))
		} else {
			defer rdb.Close()
		}

		var km kafka.KafkaManagement
		if len(cfg.Kafka.Brokers) > 0 {
			km = kafka.NewKafkaManagement(cfg.Kafka.Brokers, log)
		}

		userRepo, err := infras.NewGORMUserRepository(db)
		if err != nil {
			log.Warn("Failed to initialize GORM User Repository", zap.Error(err))
		}

		tokenRepo := infras.NewRedisTokenStore(rdb)
		jwtMgr := infras.NewJWTManager(cfg.Auth.JWTSecret)
		eventPub := infras.NewKafkaEventPublisher(km)

		authUsecase := usecase.NewAuthUsecase(userRepo, tokenRepo, jwtMgr, eventPub)
		app := auth.NewApp(authUsecase)

		grpcServer := bootstrap.NewServerBootstrap(bootstrap.GRPCServerConfig{
			ServiceName:     cfg.App.Name,
			Port:            fmt.Sprintf(":%d", cfg.GRPC.Port),
			ShutdownTimeout: 5 * time.Second,
		}, log)

		if err := grpcServer.Start(app); err != nil {
			return fmt.Errorf("failed to start gRPC server: %w", err)
		}

		var redisClient *_redis.Client
		if rdb != nil {
			redisClient = rdb.Client
		}
		bootstrap.StartMonitorServer(cfg.Metrics.Port, db, redisClient, log)

		<-ctx.Done()
		log.Info("Shutting down auth service...")

		return grpcServer.Stop(context.Background())
	}, ".env.auth")
}
