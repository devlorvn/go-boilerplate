package bootstrap

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-boilerplate/internal/golibs/configs"
	"go-boilerplate/internal/golibs/logger"

	"go.uber.org/zap"
)

func Run(runner func(ctx context.Context, cfg *configs.Config, log *zap.Logger) error, configName ...string) {
	cName := ".env"
	if len(configName) > 0 && configName[0] != "" {
		cName = configName[0]
	}

	cfg, err := configs.LoadConfig(".", cName)
	if err != nil {
		panic(err)
	}

	log, err := logger.NewLogger(cfg.App.Env, cfg.Log.Level, cfg.App.Name)
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("Starting Application...", zap.String("app_name", cfg.App.Name), zap.String("env", cfg.App.Env))

	if err := runner(ctx, cfg, log); err != nil && err != context.Canceled {
		log.Error("Application crashed", zap.Error(err))
	}

	log.Info("Application gracefully stopped")
	time.Sleep(500 * time.Millisecond)
}
