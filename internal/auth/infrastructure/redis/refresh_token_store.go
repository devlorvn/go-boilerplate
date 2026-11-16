// Package redis implements application.RefreshTokenStore.
package redis

import (
	"context"
	"errors"
	"time"

	"go-boilerplate/internal/auth/domain"
	"go-boilerplate/internal/golibs/redis"

	goredis "github.com/redis/go-redis/v9"
)

const keyPrefix = "refresh_token:"

type RefreshTokenStore struct {
	client *redis.RedisClient
}

func NewRefreshTokenStore(client *redis.RedisClient) *RefreshTokenStore {
	return &RefreshTokenStore{client: client}
}

func (s *RefreshTokenStore) Save(ctx context.Context, token, userID string, ttl time.Duration) error {
	return s.client.Client.Set(ctx, keyPrefix+token, userID, ttl).Err()
}

// Consume uses GETDEL so a token can be redeemed exactly once, even under
// concurrent refresh requests.
func (s *RefreshTokenStore) Consume(ctx context.Context, token string) (string, error) {
	userID, err := s.client.Client.GetDel(ctx, keyPrefix+token).Result()
	if errors.Is(err, goredis.Nil) {
		return "", domain.ErrInvalidToken
	}
	if err != nil {
		return "", err
	}
	return userID, nil
}
