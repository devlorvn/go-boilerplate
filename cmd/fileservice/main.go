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
	s3lib "go-boilerplate/internal/golibs/s3"

	"go.uber.org/zap"
)

func main() {
	bootstrap.Run(func(ctx context.Context, cfg *configs.Config, log *zap.Logger) error {
		db, err := database.NewGORM(cfg.DB, cfg.App.Env == "local")
		if err != nil {
			log.Warn("Database connection failed for file service", zap.Error(err))
		}

		fileRepo, err := infras.NewGORMFileRepository(db)
		if err != nil {
			log.Warn("Failed to initialize GORM File Repository", zap.Error(err))
		}

		s3Client := s3lib.NewS3Client(nil, "my-app-bucket")
		storageRepo := infras.NewS3StorageRepository(s3Client)

		fileUsecase := usecase.NewFileUsecase(fileRepo, storageRepo)
		app := fileapp.NewApp(fileUsecase)

		cleanupWorker := worker.NewCleanupWorker(fileUsecase, log, 24*time.Hour)
		cleanupWorker.Start(ctx)
		defer cleanupWorker.Stop()

		grpcServer := bootstrap.NewServerBootstrap(bootstrap.GRPCServerConfig{
			ServiceName:     "fileservice",
			Port:            ":50052",
			ShutdownTimeout: 5 * time.Second,
		}, log)

		if err := grpcServer.Start(app); err != nil {
			return fmt.Errorf("failed to start File Service gRPC server: %w", err)
		}

		<-ctx.Done()
		log.Info("Shutting down file service...")

		return grpcServer.Stop(context.Background())
	})
}
