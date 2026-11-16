package main

import (
	"context"
	"fmt"
	"time"

	fileapp "go-boilerplate/internal/file/app"
	"go-boilerplate/internal/file/infras"
	"go-boilerplate/internal/file/usecase"
	"go-boilerplate/internal/file/worker"
	"go-boilerplate/internal/golibs/bootstrap"
	"go-boilerplate/internal/golibs/configs"
	"go-boilerplate/internal/golibs/database"
	"go-boilerplate/internal/golibs/interceptors"
	"go-boilerplate/internal/golibs/jwtutil"
	s3lib "go-boilerplate/internal/golibs/s3"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	bootstrap.Run(func(ctx context.Context, cfg *configs.Config, log *zap.Logger) error {
		verifier, err := jwtutil.NewHS256(cfg.Auth.JWTSecret)
		if err != nil {
			return fmt.Errorf("AUTH_JWT_SECRET: %w", err)
		}

		db, err := database.NewGORM(cfg.DB, cfg.App.Env == "local")
		if err != nil {
			return fmt.Errorf("connect database: %w", err)
		}

		fileRepo, err := infras.NewGORMFileRepository(db)
		if err != nil {
			return fmt.Errorf("init file repository: %w", err)
		}

		s3Opts := s3.Options{
			Region:       cfg.Provider.AWS.Region,
			Credentials:  credentials.NewStaticCredentialsProvider(cfg.Provider.AWS.AccessKeyID, cfg.Provider.AWS.SecretAccessKey, ""),
			UsePathStyle: true,
		}

		if cfg.Provider.AWS.Endpoint != "" {
			s3Opts.BaseEndpoint = aws.String(cfg.Provider.AWS.Endpoint)
		}

		rawS3Client := s3.New(s3Opts)

		bucket := cfg.Provider.AWS.Bucket
		if bucket == "" {
			bucket = "my-app-bucket"
		}

		s3Client := s3lib.NewS3Client(rawS3Client, bucket)
		storageRepo := infras.NewS3StorageRepository(s3Client)

		fileUsecase := usecase.NewFileUsecase(fileRepo, storageRepo)
		app := fileapp.NewApp(fileUsecase)

		cleanupWorker := worker.NewCleanupWorker(fileUsecase, log, 24*time.Hour)
		cleanupWorker.Start(ctx)
		defer cleanupWorker.Stop()

		grpcServer := bootstrap.NewServerBootstrap(bootstrap.GRPCServerConfig{
			ServiceName:     cfg.App.Name,
			Port:            fmt.Sprintf(":%d", cfg.GRPC.Port),
			ShutdownTimeout: 5 * time.Second,
		}, log, grpc.ChainUnaryInterceptor(
			interceptors.RecoveryInterceptor(log),
			interceptors.LoggingInterceptor(log),
			interceptors.AuthInterceptor(verifier),
		))

		if err := grpcServer.Start(app); err != nil {
			return fmt.Errorf("failed to start File Service gRPC server: %w", err)
		}

		bootstrap.StartMonitorServer(cfg.Metrics.Port, db, nil, log)

		<-ctx.Done()
		log.Info("Shutting down file service...")

		return grpcServer.Stop(context.Background())
	}, ".env.file")
}
