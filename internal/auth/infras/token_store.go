package infras

import (
	"context"
	"fmt"
	"time"

	"go-boilerplate/internal/auth/domain"
	"go-boilerplate/internal/golibs/redis"
)

type RedisTokenStore struct {
	rClient *redis.RedisClient
}

func NewRedisTokenStore(rClient *redis.RedisClient) *RedisTokenStore {
	return &RedisTokenStore{rClient: rClient}
}

func (s *RedisTokenStore) StoreRefreshToken(ctx context.Context, userID, refreshToken string, ttl time.Duration) error {
	if s.rClient == nil || s.rClient.Client == nil {
		return nil
	}
	key := fmt.Sprintf("refresh_token:%s", refreshToken)
	return s.rClient.Client.Set(ctx, key, userID, ttl).Err()
}

func (s *RedisTokenStore) GetRefreshToken(ctx context.Context, refreshToken string) (string, error) {
	if s.rClient == nil || s.rClient.Client == nil {
		return "", domain.ErrInvalidToken
	}
	key := fmt.Sprintf("refresh_token:%s", refreshToken)
	userID, err := s.rClient.Client.Get(ctx, key).Result()
	if err != nil {
		return "", domain.ErrInvalidToken
	}
	return userID, nil
}

func (s *RedisTokenStore) DeleteRefreshToken(ctx context.Context, refreshToken string) error {
	if s.rClient == nil || s.rClient.Client == nil {
		return nil
	}
	key := fmt.Sprintf("refresh_token:%s", refreshToken)
	return s.rClient.Client.Del(ctx, key).Err()
}
