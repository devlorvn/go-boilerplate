package bootstrap

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type HTTPServiceRegistrar interface {
	RegisterHTTP(router *gin.Engine)
}

type HTTPServerConfig struct {
	ServiceName     string        `json:"service_name"`
	Port            string        `json:"port"`
	IsDebug         bool          `json:"is_debug"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
}

type HTTPServerBootstrap struct {
	Config     HTTPServerConfig
	Logger     *zap.Logger
	Router     *gin.Engine
	HTTPServer *http.Server
}

func NewHTTPServerBootstrap(cfg HTTPServerConfig, logger *zap.Logger) *HTTPServerBootstrap {
	if !cfg.IsDebug {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	httpServer := &http.Server{
		Addr:    cfg.Port,
		Handler: router,
	}

	return &HTTPServerBootstrap{
		Config:     cfg,
		Logger:     logger,
		Router:     router,
		HTTPServer: httpServer,
	}
}

func (sb *HTTPServerBootstrap) Start(registrar HTTPServiceRegistrar) error {
	registrar.RegisterHTTP(sb.Router)
	sb.Logger.Info("Starting HTTP Server", zap.String("port", sb.Config.Port))

	go func() {
		if err := sb.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			sb.Logger.Error("Fail to serve HTTP", zap.Error(err))
		}
	}()
	return nil
}

func (sb *HTTPServerBootstrap) Stop(ctx context.Context) error {
	sb.Logger.Info("Shutting down HTTP server...")
	timeoutCtx, cancel := context.WithTimeout(ctx, sb.Config.ShutdownTimeout)
	defer cancel()

	if err := sb.HTTPServer.Shutdown(timeoutCtx); err != nil {
		sb.Logger.Error("HTTP Server forced shutdown", zap.Error(err))
		return err
	}
	sb.Logger.Info("Graceful shutdown HTTPServer done")
	return nil
}
