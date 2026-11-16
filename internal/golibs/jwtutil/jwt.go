// Package jwtutil is the single source of truth for the access-token format,
// shared by the token issuer (auth service) and every verifier (gRPC interceptors).
package jwtutil

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrEmptySecret  = errors.New("jwt secret must not be empty")
	ErrInvalidToken = errors.New("invalid or expired token")
)

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type HS256 struct {
	secret []byte
}

func NewHS256(secret string) (*HS256, error) {
	if secret == "" {
		return nil, ErrEmptySecret
	}
	return &HS256{secret: []byte(secret)}, nil
}

func (h *HS256) Sign(userID, role string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(h.secret)
}

func (h *HS256) Verify(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return h.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil || !token.Valid || claims.UserID == "" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
