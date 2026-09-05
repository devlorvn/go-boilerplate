package main

import (
	"context"
	"fmt"
	"time"

	auth "go-boilerplate/internal/auth/app"
	"go-boilerplate/internal/auth/usecase"
	"go-boilerplate/internal/golibs/bootstrap"
	"go-boilerplate/internal/golibs/configs"
	"go-boilerplate/internal/golibs/database"
	"go-boilerplate/internal/golibs/redis"

	_redis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	bootstrap.Run(func(ctx context.Context, cfg *configs.Config, log *zap.Logger) error {
		// 1. Khởi tạo DB & Redis
		db, err := database.NewGORM(cfg.DB, cfg.App.Env == "local")
		if err != nil {
			log.Warn("Database disabled or connection failed", zap.Error(err))
		}

		rdb, err := redis.NewRedisClient(ctx, cfg.Redis, log)
		if err != nil {
			log.Warn("Redis disabled or connection failed", zap.Error(err))
		} else {
			defer rdb.Close()
		}

		// 2. Khởi tạo Business App & Usecase
		authUsecase := usecase.NewAuthUsecase(nil)
		app := auth.NewApp(authUsecase)

		// 3. Khởi tạo gRPC, Gin HTTP, Kafka Bootstrappers
		grpcServer := bootstrap.NewServerBootstrap(bootstrap.GRPCServerConfig{
			ServiceName:     cfg.App.Name,
			Port:            fmt.Sprintf(":%d", cfg.GRPC.Port),
			ShutdownTimeout: 5 * time.Second,
		}, log)

		httpServer := bootstrap.NewHTTPServerBootstrap(bootstrap.HTTPServerConfig{
			ServiceName:     cfg.App.Name,
			Port:            ":8080",
			IsDebug:         cfg.App.Env == "local",
			ShutdownTimeout: 5 * time.Second,
		}, log)

		kafkaConsumer := bootstrap.NewKafkaConsumerBootstrap(bootstrap.KafkaBootstrapConfig{
			ServiceName:     cfg.App.Name,
			Brokers:         cfg.Kafka.Brokers,
			ShutdownTimeout: 5 * time.Second,
		}, log)

		// 4. Start các Protocol Servers
		_ = grpcServer.Start(app)
		_ = httpServer.Start(app)
		if len(cfg.Kafka.Brokers) > 0 {
			_ = kafkaConsumer.Start(ctx, app)
		}

		// Start HTTP Monitor Server (Port :2112 cho /metrics & /healthz)
		var redisClient *_redis.Client
		if rdb != nil {
			redisClient = rdb.Client
		}
		bootstrap.StartMonitorServer(cfg.Metrics.Port, db, redisClient, log)

		// 5. Chờ tín hiệu ngắt OS Signal (SIGINT/SIGTERM) từ bootstrap.Run
		<-ctx.Done()
		log.Info("Root context cancelled, shutting down all servers...")

		// Graceful Shutdown toàn bộ servers
		_ = httpServer.Stop(context.Background())
		_ = grpcServer.Stop(context.Background())
		if len(cfg.Kafka.Brokers) > 0 {
			_ = kafkaConsumer.Stop(context.Background())
		}

		return nil
	})
}
