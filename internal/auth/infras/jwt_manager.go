package infras

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"go-boilerplate/internal/auth/domain"

	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secretKey string
}

func NewJWTManager(secretKey string) *JWTManager {
	if secretKey == "" {
		secretKey = "super-secret-default-key"
	}
	return &JWTManager{secretKey: secretKey}
}

func (m *JWTManager) GenerateAccessToken(userID, role string, ttl time.Duration) (string, error) {
	claims := JWTClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.secretKey))
}

func (m *JWTManager) ValidateAccessToken(tokenString string) (string, string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(m.secretKey), nil
	})
	if err != nil || !token.Valid {
		return "", "", domain.ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return "", "", domain.ErrInvalidToken
	}
	return claims.UserID, claims.Role, nil
}

func (m *JWTManager) GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", errors.New("failed to generate random refresh token")
	}
	return hex.EncodeToString(b), nil
}
