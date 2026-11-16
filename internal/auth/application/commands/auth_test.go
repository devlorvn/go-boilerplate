package commands_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-boilerplate/internal/auth/application/apptest"
	"go-boilerplate/internal/auth/application/commands"
	"go-boilerplate/internal/auth/domain"
	"go-boilerplate/internal/golibs/authz"
	"go-boilerplate/internal/golibs/errorx"
	"go-boilerplate/internal/golibs/jwtutil"
)

type fixture struct {
	h       *commands.AuthCommandHandler
	users   *apptest.Users
	refresh *apptest.RefreshTokens
	events  *apptest.Events
	signer  *jwtutil.HS256
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	signer, _ := jwtutil.NewHS256("test-secret")
	f := &fixture{users: apptest.NewUsers(), refresh: apptest.NewRefreshTokens(), events: &apptest.Events{}, signer: signer}
	f.h = &commands.AuthCommandHandler{
		Users: f.users, Refresh: f.refresh, Signer: signer, Events: f.events,
		TTL: commands.TokenTTL{Access: time.Minute, Refresh: time.Hour},
	}
	return f
}

var validReg = commands.RegisterCommand{Email: "  Alice@Example.com ", Password: "Passw0rd!", Name: " Alice "}

func TestRegister(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	res, err := f.h.Register(ctx, validReg)
	if err != nil {
		t.Fatal(err)
	}
	if res.User.ID == "" || res.User.Email != "alice@example.com" || res.User.Name != "Alice" || res.User.Role != authz.RoleUser {
		t.Fatalf("unexpected user: %+v", res.User)
	}
	if res.User.PasswordHash == validReg.Password {
		t.Fatal("password stored in clear")
	}
	claims, err := f.signer.Verify(res.AccessToken)
	if err != nil || claims.UserID != res.User.ID || claims.Role != authz.RoleUser {
		t.Fatalf("bad access token: %+v %v", claims, err)
	}
	if len(f.events.Created) != 1 {
		t.Fatalf("expected 1 user.created event, got %d", len(f.events.Created))
	}

	if _, err := f.h.Register(ctx, commands.RegisterCommand{Email: "alice@example.com", Password: "another-pw", Name: "A2"}); !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Fatalf("duplicate (case-insensitive) email: got %v", err)
	}
}

func TestRegister_Validation(t *testing.T) {
	f := newFixture(t)
	for name, cmd := range map[string]commands.RegisterCommand{
		"bad email":      {Email: "nope", Password: "Passw0rd!", Name: "A"},
		"short password": {Email: "a@b.co", Password: "short", Name: "A"},
		"empty name":     {Email: "a@b.co", Password: "Passw0rd!", Name: "  "},
	} {
		_, err := f.h.Register(context.Background(), cmd)
		if errorx.KindOf(err) != errorx.InvalidArgument {
			t.Errorf("%s: want InvalidArgument, got %v", name, err)
		}
	}
}

func TestRegister_EventFailureDoesNotFailSignup(t *testing.T) {
	f := newFixture(t)
	f.events.Err = errors.New("kafka down")
	if _, err := f.h.Register(context.Background(), validReg); err != nil {
		t.Fatalf("register must succeed when publishing fails: %v", err)
	}
}

func TestLogin(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	if _, err := f.h.Register(ctx, validReg); err != nil {
		t.Fatal(err)
	}

	if _, err := f.h.Login(ctx, commands.LoginCommand{Email: "ALICE@example.com", Password: validReg.Password}); err != nil {
		t.Fatalf("valid login: %v", err)
	}
	for name, cmd := range map[string]commands.LoginCommand{
		"wrong password": {Email: "alice@example.com", Password: "wrong-password"},
		"unknown email":  {Email: "bob@example.com", Password: validReg.Password},
	} {
		if _, err := f.h.Login(ctx, cmd); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Errorf("%s: want ErrInvalidCredentials, got %v", name, err)
		}
	}
}

func TestRefreshTokens_RotatesAndIsSingleUse(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	res, err := f.h.Register(ctx, validReg)
	if err != nil {
		t.Fatal(err)
	}

	pair, err := f.h.RefreshTokens(ctx, res.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if pair.RefreshToken == res.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}
	if _, err := f.h.RefreshTokens(ctx, res.RefreshToken); !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("replayed refresh token must fail, got %v", err)
	}
	if _, err := f.h.RefreshTokens(ctx, ""); !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("empty token: got %v", err)
	}
}
