package bootstrap

import (
	"context"
	"net"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type ServiceRegistrar interface {
	RegisterGRPC(server *grpc.Server)
}

type GRPCServerConfig struct {
	ServiceName     string        `json:"service_name"`
	Port            string        `json:"port"`
	MetricsPort     string        `json:"metrics_port"`
	IsDebug         bool          `json:"is_debug"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
}

type ServerBootstrap struct {
	Config     GRPCServerConfig
	Logger     *zap.Logger
	GRPCServer *grpc.Server
}

func NewServerBootstrap(cfg GRPCServerConfig, logger *zap.Logger, opts ...grpc.ServerOption) *ServerBootstrap {
	grpcServer := grpc.NewServer(opts...)

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus(cfg.ServiceName, healthpb.HealthCheckResponse_SERVING)

	return &ServerBootstrap{
		Config:     cfg,
		Logger:     logger,
		GRPCServer: grpcServer,
	}
}

func (sb *ServerBootstrap) Start(registrar ServiceRegistrar) error {
	// 1. Register GRPC services
	registrar.RegisterGRPC(sb.GRPCServer)

	// 2. Start GRPC server
	listener, err := net.Listen("tcp", sb.Config.Port)

	if err != nil {
		sb.Logger.Error("Fail to listen on port", zap.String("port", sb.Config.Port), zap.Error(err))
		return err
	}

	sb.Logger.Info("Starting gRPC Server", zap.String("port", sb.Config.Port))

	go func() {
		sb.Logger.Info("gRpc Server listing on port:", zap.String("port", sb.Config.Port))
		if err = sb.GRPCServer.Serve(listener); err != nil {
			sb.Logger.Error("Fail to serve gRPC", zap.Error(err))
		}
	}()

	return nil
}

func (sb *ServerBootstrap) Stop(ctx context.Context) error {
	sb.Logger.Info("Shutting down gRPC server...")

	timeoutCtx, cancel := context.WithTimeout(ctx, sb.Config.ShutdownTimeout)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		sb.GRPCServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		sb.Logger.Info("Graceful shutdown GrpcServer done")
	case <-timeoutCtx.Done():
		sb.Logger.Warn("Graceful shutdown timeout, forcing stop")
		sb.GRPCServer.Stop()
	}
	return nil
}
