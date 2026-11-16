package errorx

import (
	"errors"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToGRPC(t *testing.T) {
	notFound := New(NotFound, "user not found")
	cases := []struct {
		name     string
		err      error
		wantCode codes.Code
		wantMsg  string
	}{
		{"kind", notFound, codes.NotFound, "user not found"},
		{"wrapped kind", fmt.Errorf("get profile: %w", notFound), codes.NotFound, "user not found"},
		{"plain error hides details", errors.New("pq: connection refused at 10.0.0.1"), codes.Internal, "internal error"},
		{"status passes through", status.Error(codes.Unauthenticated, "x"), codes.Unauthenticated, "x"},
		{"invalid", Invalid("email %q is invalid", "a"), codes.InvalidArgument, `email "a" is invalid`},
	}
	for _, c := range cases {
		st := status.Convert(ToGRPC(c.err))
		if st.Code() != c.wantCode || st.Message() != c.wantMsg {
			t.Errorf("%s: got %v %q", c.name, st.Code(), st.Message())
		}
	}
	if ToGRPC(nil) != nil {
		t.Error("nil must stay nil")
	}
}
