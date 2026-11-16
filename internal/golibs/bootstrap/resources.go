package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go-boilerplate/internal/golibs/configs"
	"go-boilerplate/internal/golibs/database"
	"go-boilerplate/internal/golibs/healthcheck"
	"go-boilerplate/internal/golibs/kafka"
	"go-boilerplate/internal/golibs/redis"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var ErrKafkaDisabled = errors.New("kafka disabled: kafka.brokers is empty")

// Resources hands out shared infra clients. Each client is created on first
// use from the Base config and closed by the bootstrap on shutdown, so a
// service only connects to what it actually asks for.
type Resources struct {
	base   configs.Base
	logger *zap.Logger

	mu    sync.Mutex
	db    *gorm.DB
	redis *redis.RedisClient
	kafka kafka.KafkaManagement
}

func NewResources(base configs.Base, logger *zap.Logger) *Resources {
	return &Resources{base: base, logger: logger}
}

func (r *Resources) Logger() *zap.Logger { return r.logger }

func (r *Resources) DB() (*gorm.DB, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.db != nil {
		return r.db, nil
	}
	sqlLog := r.base.Common.IsLocal() && r.base.Common.LogLevel == "debug"
	db, err := database.NewGORM(r.base.Postgres, sqlLog)
	if err != nil {
		return nil, fmt.Errorf("postgres %s/%s: %w", r.base.Postgres.Host, r.base.Postgres.DBName, err)
	}
	r.db = db
	return db, nil
}

func (r *Resources) Redis(ctx context.Context) (*redis.RedisClient, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.redis != nil {
		return r.redis, nil
	}
	rc, err := redis.NewRedisClient(ctx, r.base.Redis, r.logger)
	if err != nil {
		return nil, fmt.Errorf("redis %s: %w", r.base.Redis.Addr(), err)
	}
	r.redis = rc
	return rc, nil
}

// Kafka returns ErrKafkaDisabled when no brokers are configured, so callers
// decide whether kafka is optional for them.
func (r *Resources) Kafka() (kafka.KafkaManagement, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.kafka != nil {
		return r.kafka, nil
	}
	if len(r.base.Kafka.Brokers) == 0 {
		return nil, ErrKafkaDisabled
	}
	r.kafka = kafka.NewKafkaManagement(r.base.Kafka.Brokers, r.logger)
	return r.kafka, nil
}

// healthChecks pings only the clients that were actually opened.
func (r *Resources) healthChecks() map[string]healthcheck.Check {
	r.mu.Lock()
	defer r.mu.Unlock()
	checks := map[string]healthcheck.Check{}
	if db := r.db; db != nil {
		checks["postgres"] = func(ctx context.Context) error {
			sqlDB, err := db.DB()
			if err != nil {
				return err
			}
			return sqlDB.PingContext(ctx)
		}
	}
	if rc := r.redis; rc != nil {
		checks["redis"] = func(ctx context.Context) error { return rc.Client.Ping(ctx).Err() }
	}
	return checks
}

func (r *Resources) close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.kafka != nil {
		if err := r.kafka.Close(); err != nil {
			r.logger.Warn("close kafka", zap.Error(err))
		}
	}
	if r.redis != nil {
		r.redis.Close()
	}
	if r.db != nil {
		if sqlDB, err := r.db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
}
