package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"go-boilerplate/internal/golibs/configs"
	"go-boilerplate/internal/golibs/healthcheck"
	"go-boilerplate/internal/golibs/interceptors"
	"go-boilerplate/internal/golibs/logger"
	"go-boilerplate/internal/golibs/metrics"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

const defaultShutdownTimeout = 10 * time.Second

// stopper gracefully stops one server within ctx's deadline.
type stopper func(ctx context.Context)

func run[T configs.Provider](ctx context.Context, name, configDir string, s Servicer[T]) error {
	cfg, err := configs.Load[T](name, configDir)
	if err != nil {
		return err
	}
	c := *cfg
	common := c.GetBase().Common

	log, err := logger.NewLogger(common.Env, common.LogLevel, name)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer func() { _ = log.Sync() }()

	rsc := NewResources(c.GetBase(), log)
	defer rsc.close()

	if err := s.InitDependencies(ctx, c, rsc); err != nil {
		return fmt.Errorf("init dependencies: %w", err)
	}

	g, gctx := errgroup.WithContext(ctx)
	var stops []stopper
	probe := healthcheck.NewProbe(rsc.healthChecks)
	var drains []func() // mark the service not ready (grpc health, /readyz)

	// Listeners are opened synchronously so "port already in use" fails the
	// start instead of surfacing later from a goroutine.
	if gs, ok := any(s).(GRPCServicer[T]); ok {
		stop, drain, err := startGRPC(ctx, g, c, common, gs, rsc)
		if err != nil {
			return err
		}
		stops = append(stops, stop)
		drains = append(drains, drain)
	}

	if hs, ok := any(s).(HTTPServicer[T]); ok {
		stop, err := startHTTP(ctx, g, common, func(r *gin.Engine) { hs.RegisterHTTP(c, r) }, log)
		if err != nil {
			return err
		}
		stops = append(stops, stop)
	}

	// The monitor server stops last so probes keep answering while draining.
	var stopMonitor stopper = func(context.Context) {}
	if common.MetricsPort != 0 {
		stop, err := startMonitor(ctx, g, common.MetricsPort, probe, log)
		if err != nil {
			return err
		}
		stopMonitor = stop
	}
	drains = append(drains, probe.SetDraining)

	if bs, ok := any(s).(BackgroundServicer[T]); ok {
		g.Go(func() error {
			err := bs.RunBackground(gctx, c, rsc)
			if gctx.Err() != nil {
				return nil //nolint:nilerr // workers return ctx errors on normal shutdown
			}
			if err == nil {
				err = errors.New("background worker exited unexpectedly")
			}
			return fmt.Errorf("background: %w", err)
		})
	}

	timeout := common.ShutdownTimeout
	if timeout <= 0 {
		timeout = defaultShutdownTimeout
	}
	g.Go(func() error {
		<-gctx.Done()
		log.Info("shutting down", zap.Duration("drain_delay", common.DrainDelay), zap.Duration("timeout", timeout))
		for _, drain := range drains {
			drain()
		}
		time.Sleep(common.DrainDelay)

		sctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		for i := len(stops) - 1; i >= 0; i-- {
			stops[i](sctx)
		}
		if sd, ok := any(s).(Shutdowner); ok {
			sd.Shutdown(sctx)
		}
		stopMonitor(sctx)
		return nil
	})

	log.Info("service started", zap.String("env", common.Env),
		zap.Int("grpc_port", common.GRPCPort), zap.Int("http_port", common.HTTPPort), zap.Int("metrics_port", common.MetricsPort))

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("service stopped with error", zap.Error(err))
		return err
	}
	log.Info("service stopped")
	return nil
}

func listen(ctx context.Context, kind string, port int) (net.Listener, error) {
	if port == 0 {
		return nil, fmt.Errorf("%s port is not configured", kind)
	}
	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("%s listen: %w", kind, err)
	}
	return lis, nil
}

func startGRPC[T configs.Provider](ctx context.Context, g *errgroup.Group, c T, common configs.Common, gs GRPCServicer[T], rsc *Resources) (stopper, func(), error) {
	lis, err := listen(ctx, "grpc", common.GRPCPort)
	if err != nil {
		return nil, nil, err
	}
	log := rsc.Logger()

	unary := append([]grpc.UnaryServerInterceptor{
		interceptors.RecoveryInterceptor(log),
		interceptors.LoggingInterceptor(log),
		interceptors.ErrorInterceptor(log),
	}, gs.UnaryInterceptors(c, rsc)...)

	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(unary...))
	hs := health.NewServer()
	healthpb.RegisterHealthServer(srv, hs)
	if common.IsLocal() {
		reflection.Register(srv) // lets grpcurl/postman discover services locally
	}
	gs.RegisterGRPC(c, srv)

	g.Go(func() error {
		if err := srv.Serve(lis); err != nil {
			return fmt.Errorf("grpc serve: %w", err)
		}
		return nil
	})

	stop := func(ctx context.Context) {
		done := make(chan struct{})
		go func() { srv.GracefulStop(); close(done) }()
		select {
		case <-done:
		case <-ctx.Done():
			log.Warn("grpc graceful stop timed out, forcing")
			srv.Stop()
		}
	}
	// NOT_SERVING tells gRPC-aware load balancers to drain this instance.
	return stop, hs.Shutdown, nil
}

func startHTTP(ctx context.Context, g *errgroup.Group, common configs.Common, register func(*gin.Engine), log *zap.Logger) (stopper, error) {
	lis, err := listen(ctx, "http", common.HTTPPort)
	if err != nil {
		return nil, err
	}
	if !common.IsLocal() {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	register(r)

	return serveHTTP(g, "http", lis, r, log), nil
}

func startMonitor(ctx context.Context, g *errgroup.Group, port int, probe *healthcheck.Probe, log *zap.Logger) (stopper, error) {
	lis, err := listen(ctx, "metrics", port)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/livez", probe.Live)
	mux.HandleFunc("/readyz", probe.Ready)
	mux.Handle("/metrics", metrics.Handler())
	return serveHTTP(g, "metrics", lis, mux, log), nil
}

func serveHTTP(g *errgroup.Group, kind string, lis net.Listener, h http.Handler, log *zap.Logger) stopper {
	srv := &http.Server{Handler: h, ReadHeaderTimeout: 10 * time.Second}
	g.Go(func() error {
		if err := srv.Serve(lis); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("%s serve: %w", kind, err)
		}
		return nil
	})
	return func(ctx context.Context) {
		if err := srv.Shutdown(ctx); err != nil {
			log.Warn("http shutdown", zap.String("server", kind), zap.Error(err))
		}
	}
}
