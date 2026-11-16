package interceptors

import (
	"context"
	"strings"

	"go-boilerplate/internal/golibs/jwtutil"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const (
	UserIDKey   contextKey = "user_id"
	UserRoleKey contextKey = "user_role"
)

func ContextWithUser(ctx context.Context, userID, role string) context.Context {
	ctx = context.WithValue(ctx, UserIDKey, userID)
	return context.WithValue(ctx, UserRoleKey, role)
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok && userID != ""
}

func UserRoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(UserRoleKey).(string)
	return role, ok && role != ""
}

// TokenVerifier is satisfied by *jwtutil.HS256.
type TokenVerifier interface {
	Verify(token string) (*jwtutil.Claims, error)
}

// AuthInterceptor verifies the bearer token in the "authorization" metadata and
// puts the user id and role into the context. Health checks and the given
// publicMethods (full gRPC method names, e.g. "/user.v1.UserService/Login") skip auth.
func AuthInterceptor(verifier TokenVerifier, publicMethods ...string) grpc.UnaryServerInterceptor {
	public := make(map[string]struct{}, len(publicMethods))
	for _, m := range publicMethods {
		public[m] = struct{}{}
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if _, ok := public[info.FullMethod]; ok || strings.HasPrefix(info.FullMethod, "/grpc.health.v1.Health/") {
			return handler(ctx, req)
		}

		token, err := bearerFromMetadata(ctx)
		if err != nil {
			return nil, err
		}

		claims, err := verifier.Verify(token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}

		return handler(ContextWithUser(ctx, claims.UserID, claims.Role), req)
	}
}

func bearerFromMetadata(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}
	values := md.Get("authorization")
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization header")
	}
	scheme, token, found := strings.Cut(values[0], " ")
	if !found || !strings.EqualFold(scheme, "bearer") || token == "" {
		return "", status.Error(codes.Unauthenticated, "authorization header must be 'Bearer <token>'")
	}
	return token, nil
}
