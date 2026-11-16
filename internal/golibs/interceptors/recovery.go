package interceptors

import (
	"context"
	"runtime/debug"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RecoveryInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				logger.Error("gRPC Handler Recovered from Panic",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
					zap.String("stacktrace", stack),
				)
				err = status.Error(codes.Internal, "internal error")
			}
		}()

		return handler(ctx, req)
	}
}
