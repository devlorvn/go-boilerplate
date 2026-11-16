package interceptors

import (
	"context"

	"go-boilerplate/internal/golibs/errorx"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// ErrorInterceptor lets handlers return plain domain errors. It converts them
// with errorx.ToGRPC and logs the original cause of Internal errors, which
// clients only see as "internal error".
func ErrorInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err == nil {
			return resp, nil
		}
		if _, isStatus := status.FromError(err); !isStatus && errorx.KindOf(err) == errorx.Internal {
			logger.Error("internal error", zap.String("method", info.FullMethod), zap.Error(err))
		}
		return resp, errorx.ToGRPC(err)
	}
}
