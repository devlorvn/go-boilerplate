package infras

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"go-boilerplate/internal/auth/domain"
	"go-boilerplate/internal/golibs/jwtutil"
)

type JWTManager struct {
	signer *jwtutil.HS256
}

func NewJWTManager(signer *jwtutil.HS256) *JWTManager {
	return &JWTManager{signer: signer}
}

func (m *JWTManager) GenerateAccessToken(userID, role string, ttl time.Duration) (string, error) {
	return m.signer.Sign(userID, role, ttl)
}

func (m *JWTManager) ValidateAccessToken(tokenString string) (string, string, error) {
	claims, err := m.signer.Verify(tokenString)
	if err != nil {
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
