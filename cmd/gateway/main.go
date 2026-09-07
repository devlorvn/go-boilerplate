package main

import (
	"context"
	"fmt"
	"time"

	"go-boilerplate/internal/gateway/app"
	"go-boilerplate/internal/golibs/bootstrap"
	"go-boilerplate/internal/golibs/configs"
	"go-boilerplate/internal/golibs/redis"

	_redis "github.com/redis/go-redis/v9"
	filev1 "go-boilerplate/pkg/genproto/file/v1"
	userv1 "go-boilerplate/pkg/genproto/user/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	bootstrap.Run(func(ctx context.Context, cfg *configs.Config, log *zap.Logger) error {
		rClient, err := redis.NewRedisClient(ctx, cfg.Redis, log)
		if err != nil {
			log.Warn("Failed to initialize Redis client for Gateway rate limiter", zap.Error(err))
		}

		userConn, err := grpc.NewClient(cfg.HTTP.AuthSvcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("failed to connect to user auth service: %w", err)
		}
		defer userConn.Close()
		userClient := userv1.NewUserServiceClient(userConn)

		fileConn, err := grpc.NewClient(cfg.HTTP.FileSvcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("failed to connect to file service: %w", err)
		}
		defer fileConn.Close()
		fileClient := filev1.NewFileServiceClient(fileConn)

		gatewayApp := app.NewGatewayApp(cfg, log, rClient, userClient, fileClient)

		httpCfg := bootstrap.HTTPServerConfig{
			ServiceName:     cfg.App.Name,
			Port:            fmt.Sprintf(":%d", cfg.HTTP.Port),
			IsDebug:         cfg.App.Env == "local",
			ShutdownTimeout: 5 * time.Second,
		}

		httpServer := bootstrap.NewHTTPServerBootstrap(httpCfg, log)
		if err := httpServer.Start(gatewayApp); err != nil {
			return fmt.Errorf("failed to start HTTP gateway: %w", err)
		}

		var redisClient *_redis.Client
		if rClient != nil {
			redisClient = rClient.Client
		}
		bootstrap.StartMonitorServer(cfg.Metrics.Port, nil, redisClient, log)

		<-ctx.Done()
		return httpServer.Stop(context.Background())
	}, ".env.gateway")
}
