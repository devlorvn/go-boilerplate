package queries_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-boilerplate/internal/auth/application/apptest"
	"go-boilerplate/internal/auth/application/queries"
	"go-boilerplate/internal/auth/domain"
	"go-boilerplate/internal/golibs/authz"
	"go-boilerplate/internal/golibs/jwtutil"
)

func TestGetProfile(t *testing.T) {
	users := apptest.NewUsers(
		&domain.User{ID: "u1", Email: "u1@x.io", Role: authz.RoleUser},
		&domain.User{ID: "u2", Email: "u2@x.io", Role: authz.RoleUser},
	)
	h := &queries.UserQueryHandler{Users: users}
	ctx := context.Background()
	u1 := authz.Actor{UserID: "u1", Role: authz.RoleUser}
	admin := authz.Actor{UserID: "admin", Role: authz.RoleAdmin}

	if got, err := h.GetProfile(ctx, u1, ""); err != nil || got.ID != "u1" {
		t.Fatalf("own profile (implicit): %v %v", got, err)
	}
	if got, err := h.GetProfile(ctx, admin, "u2"); err != nil || got.ID != "u2" {
		t.Fatalf("admin reads other: %v %v", got, err)
	}
	if _, err := h.GetProfile(ctx, admin, "missing"); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("admin reads missing: %v", err)
	}

	// A non-admin gets the same answer for existing and missing IDs, and the
	// repository is not even queried.
	before := users.Lookups
	for _, id := range []string{"u2", "missing"} {
		if _, err := h.GetProfile(ctx, u1, id); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("user reads %s: want ErrForbidden, got %v", id, err)
		}
	}
	if users.Lookups != before {
		t.Fatal("forbidden reads must not hit the repository")
	}
}

func TestValidateToken(t *testing.T) {
	signer, _ := jwtutil.NewHS256("s")
	h := &queries.UserQueryHandler{Signer: signer}
	tok, _ := signer.Sign("u1", authz.RoleAdmin, time.Minute)

	actor, err := h.ValidateToken(context.Background(), tok)
	if err != nil || actor != (authz.Actor{UserID: "u1", Role: authz.RoleAdmin}) {
		t.Fatalf("got %+v %v", actor, err)
	}
	if _, err := h.ValidateToken(context.Background(), "garbage"); !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("garbage token: %v", err)
	}
}
