package interceptors

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/golibs/authz"
	"go-boilerplate/internal/golibs/jwtutil"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthInterceptor(t *testing.T) {
	v, _ := jwtutil.NewHS256("secret")
	userTok, _ := v.Sign("u1", authz.RoleUser, time.Minute)
	adminTok, _ := v.Sign("a1", authz.RoleAdmin, time.Minute)

	icpt := AuthInterceptor(v, authz.Policy{
		"/svc/Public":  authz.Public,
		"/svc/Private": authz.Authenticated,
		"/svc/Admin":   authz.Roles(authz.RoleAdmin),
	})

	var got authz.Actor
	handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
		got, _ = ActorFromContext(ctx)
		return "ok", nil
	}
	bearer := func(tok string) context.Context {
		return metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tok))
	}

	cases := []struct {
		name     string
		ctx      context.Context
		method   string
		wantCode codes.Code
		wantUser string
	}{
		{"valid token", bearer(userTok), "/svc/Private", codes.OK, "u1"},
		{"no metadata", context.Background(), "/svc/Private", codes.Unauthenticated, ""},
		{"bad scheme", metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Basic "+userTok)), "/svc/Private", codes.Unauthenticated, ""},
		{"bad token", bearer("nope"), "/svc/Private", codes.Unauthenticated, ""},
		{"public method", context.Background(), "/svc/Public", codes.OK, ""},
		{"health check", context.Background(), "/grpc.health.v1.Health/Check", codes.OK, ""},
		{"role allowed", bearer(adminTok), "/svc/Admin", codes.OK, "a1"},
		{"role denied", bearer(userTok), "/svc/Admin", codes.PermissionDenied, ""},
		{"method without rule is denied", bearer(adminTok), "/svc/Unknown", codes.PermissionDenied, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got = authz.Actor{}
			_, err := icpt(tc.ctx, nil, &grpc.UnaryServerInfo{FullMethod: tc.method}, handler)
			if status.Code(err) != tc.wantCode {
				t.Fatalf("code = %v, want %v (err=%v)", status.Code(err), tc.wantCode, err)
			}
			if got.UserID != tc.wantUser {
				t.Fatalf("user = %q, want %q", got.UserID, tc.wantUser)
			}
		})
	}
}
