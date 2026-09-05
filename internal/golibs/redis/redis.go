package redis

import (
	"context"
	"fmt"

	_redis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisConfig struct {
	Host     string `mapstructure:"HOST"`
	Port     string `mapstructure:"PORT"`
	Password string `mapstructure:"PASSWORD"`
	DB       int    `mapstructure:"DB"`
}

func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}

type RedisClient struct {
	Client *_redis.Client
	Logger *zap.Logger
}

func NewRedisClient(ctx context.Context, cfg RedisConfig, logger *zap.Logger) (*RedisClient, error) {
	logger = logger.Named("redis")

	rdb := _redis.NewClient(&_redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Error("Failed to connect Redis", zap.String("addr", cfg.Addr()), zap.Error(err))
		return nil, err
	}

	logger.Info("Connected to Redis", zap.String("addr", cfg.Addr()))
	return &RedisClient{
		Client: rdb,
		Logger: logger,
	}, nil
}

func (rc *RedisClient) Close() {
	if err := rc.Client.Close(); err != nil {
		rc.Logger.Error("Failed to close Redis", zap.Error(err))
	}
	rc.Logger.Info("Closed Redis")
}
