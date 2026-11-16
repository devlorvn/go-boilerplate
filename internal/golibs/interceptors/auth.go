package interceptors

import (
	"context"
	"strings"

	"go-boilerplate/internal/golibs/authz"
	"go-boilerplate/internal/golibs/jwtutil"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type actorKey struct{}

func ContextWithActor(ctx context.Context, a authz.Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, a)
}

// ActorFromContext returns the caller set by AuthInterceptor. ok is false on
// public methods.
func ActorFromContext(ctx context.Context) (authz.Actor, bool) {
	a, ok := ctx.Value(actorKey{}).(authz.Actor)
	return a, ok && a.UserID != ""
}

// TokenVerifier is satisfied by *jwtutil.HS256.
type TokenVerifier interface {
	Verify(token string) (*jwtutil.Claims, error)
}

// AuthInterceptor authenticates the bearer token and authorizes the caller
// against policy. Methods missing from policy are denied; health checks are
// always allowed.
func AuthInterceptor(verifier TokenVerifier, policy authz.Policy) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if strings.HasPrefix(info.FullMethod, "/grpc.health.v1.Health/") {
			return handler(ctx, req)
		}
		rule, ok := policy[info.FullMethod]
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "method not allowed")
		}
		if rule.IsPublic() {
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
		if !rule.Allows(claims.Role) {
			return nil, status.Error(codes.PermissionDenied, "insufficient role")
		}

		return handler(ContextWithActor(ctx, authz.Actor{UserID: claims.UserID, Role: claims.Role}), req)
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
