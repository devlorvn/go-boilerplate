package interceptors

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/golibs/jwtutil"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthInterceptor(t *testing.T) {
	v, _ := jwtutil.NewHS256("secret")
	good, _ := v.Sign("u1", "user", time.Minute)
	icpt := AuthInterceptor(v, "/svc/Public")

	var gotUser string
	handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
		gotUser, _ = UserIDFromContext(ctx)
		return "ok", nil
	}
	withAuth := func(h string) context.Context {
		return metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", h))
	}

	cases := []struct {
		name     string
		ctx      context.Context
		method   string
		wantCode codes.Code
		wantUser string
	}{
		{"valid token", withAuth("Bearer " + good), "/svc/Private", codes.OK, "u1"},
		{"no metadata", context.Background(), "/svc/Private", codes.Unauthenticated, ""},
		{"bad scheme", withAuth("Basic " + good), "/svc/Private", codes.Unauthenticated, ""},
		{"bad token", withAuth("Bearer nope"), "/svc/Private", codes.Unauthenticated, ""},
		{"public method", context.Background(), "/svc/Public", codes.OK, ""},
		{"health check", context.Background(), "/grpc.health.v1.Health/Check", codes.OK, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotUser = ""
			_, err := icpt(tc.ctx, nil, &grpc.UnaryServerInfo{FullMethod: tc.method}, handler)
			if status.Code(err) != tc.wantCode {
				t.Fatalf("code = %v, want %v (err=%v)", status.Code(err), tc.wantCode, err)
			}
			if gotUser != tc.wantUser {
				t.Fatalf("user = %q, want %q", gotUser, tc.wantUser)
			}
		})
	}
}
