// Package apptest provides in-memory implementations of the auth ports for
// application-layer unit tests (no DB, no Redis, no Docker).
package apptest

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go-boilerplate/internal/auth/domain"
)

type Users struct {
	mu      sync.Mutex
	byID    map[string]*domain.User
	nextID  int
	Lookups int // number of GetByID/GetByEmail calls
}

func NewUsers(seed ...*domain.User) *Users {
	u := &Users{byID: map[string]*domain.User{}}
	for _, s := range seed {
		cp := *s
		u.byID[s.ID] = &cp
	}
	return u
}

func (u *Users) Create(_ context.Context, user *domain.User) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	for _, existing := range u.byID {
		if existing.Email == user.Email {
			return domain.ErrEmailAlreadyExists
		}
	}
	u.nextID++
	user.ID = fmt.Sprintf("user-%d", u.nextID)
	cp := *user
	u.byID[user.ID] = &cp
	return nil
}

func (u *Users) GetByID(_ context.Context, id string) (*domain.User, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.Lookups++
	if user, ok := u.byID[id]; ok {
		cp := *user
		return &cp, nil
	}
	return nil, domain.ErrUserNotFound
}

func (u *Users) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.Lookups++
	for _, user := range u.byID {
		if user.Email == email {
			cp := *user
			return &cp, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

type RefreshTokens struct {
	mu     sync.Mutex
	tokens map[string]string
}

func NewRefreshTokens() *RefreshTokens { return &RefreshTokens{tokens: map[string]string{}} }

func (r *RefreshTokens) Save(_ context.Context, token, userID string, _ time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens[token] = userID
	return nil
}

func (r *RefreshTokens) Consume(_ context.Context, token string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	userID, ok := r.tokens[token]
	if !ok {
		return "", domain.ErrInvalidToken
	}
	delete(r.tokens, token)
	return userID, nil
}

type Events struct {
	Err     error
	Created []*domain.User
}

func (e *Events) PublishUserCreated(_ context.Context, u *domain.User) error {
	e.Created = append(e.Created, u)
	return e.Err
}
