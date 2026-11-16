// Package domain is the auth service's core model: entities, invariants and
// errors. It depends on nothing but the standard library and errorx.
package domain

import (
	"net/mail"
	"strings"
	"time"

	"go-boilerplate/internal/golibs/authz"
	"go-boilerplate/internal/golibs/errorx"
)

const MinPasswordLength = 8

var (
	ErrUserNotFound       = errorx.New(errorx.NotFound, "user not found")
	ErrEmailAlreadyExists = errorx.New(errorx.AlreadyExists, "email already exists")
	ErrInvalidCredentials = errorx.New(errorx.Unauthenticated, "invalid email or password")
	ErrInvalidToken       = errorx.New(errorx.Unauthenticated, "invalid or expired token")
	ErrForbidden          = errorx.New(errorx.PermissionDenied, "not allowed to access this user")
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NormalizeEmail is applied everywhere an email enters the system so lookups
// and the unique index agree.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidateRegistration checks the raw input of a new account.
func ValidateRegistration(email, password, name string) error {
	if _, err := mail.ParseAddress(email); err != nil || !strings.Contains(email, "@") {
		return errorx.Invalid("invalid email address")
	}
	if len(password) < MinPasswordLength {
		return errorx.Invalid("password must be at least %d characters", MinPasswordLength)
	}
	if strings.TrimSpace(name) == "" {
		return errorx.Invalid("name is required")
	}
	return nil
}

// NewUser builds a regular user; the ID is assigned by the repository.
func NewUser(email, passwordHash, name string, now time.Time) *User {
	return &User{
		Email:        NormalizeEmail(email),
		PasswordHash: passwordHash,
		Name:         strings.TrimSpace(name),
		Role:         authz.RoleUser,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// CanReadProfile: a user reads their own profile; admins read anyone's.
// It only needs the target ID, so it is checked before any lookup and a
// non-admin cannot learn which user IDs exist.
func CanReadProfile(actor authz.Actor, userID string) bool {
	return actor.UserID == userID || actor.IsAdmin()
}
