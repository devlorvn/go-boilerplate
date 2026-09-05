package bootstrap

import (
	"fmt"
	"go-boilerplate/internal/golibs/healthcheck"
	"go-boilerplate/internal/golibs/metrics"
	"net/http"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func StartMonitorServer(port int, db *gorm.DB, rdb *redis.Client, logger *zap.Logger) *http.Server {
	mux := http.NewServeMux()

	healthChecker := healthcheck.NewHealthChecker(db, rdb)
	mux.HandleFunc("/healthz", healthChecker.Handler())
	mux.Handle("/metrics", metrics.Handler())

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		logger.Info("Starting Monitor HTTP Server", zap.Int("port", port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Monitor HTTP Server crashed", zap.Error(err))
		}
	}()

	return server
}
