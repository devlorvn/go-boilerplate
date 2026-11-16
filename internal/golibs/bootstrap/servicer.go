package bootstrap

import (
	"context"

	"go-boilerplate/internal/golibs/configs"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

// Servicer is the only interface a service must implement. The bootstrap then
// checks for the optional interfaces below and starts whatever matches: a gRPC
// server, an HTTP server, background workers. The monitor server (/healthz,
// /metrics) always runs.
type Servicer[T configs.Provider] interface {
	// InitDependencies builds repositories, usecases and handlers. Returning an
	// error stops the process before any port is opened.
	InitDependencies(ctx context.Context, c T, rsc *Resources) error
}

type GRPCServicer[T configs.Provider] interface {
	RegisterGRPC(c T, s *grpc.Server)
	// UnaryInterceptors are chained after the default recovery + logging.
	UnaryInterceptors(c T, rsc *Resources) []grpc.UnaryServerInterceptor
}

type HTTPServicer[T configs.Provider] interface {
	RegisterHTTP(c T, r *gin.Engine)
}

// BackgroundServicer runs workers or consumers. It must block until ctx is
// cancelled; returning an error before that stops the whole service.
type BackgroundServicer[T configs.Provider] interface {
	RunBackground(ctx context.Context, c T, rsc *Resources) error
}

// Shutdowner is called once all servers have stopped, before infra is closed.
type Shutdowner interface {
	Shutdown(ctx context.Context)
}
