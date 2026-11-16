package configs

import (
	"time"

	"go-boilerplate/internal/golibs/database"
	"go-boilerplate/internal/golibs/kafka"
	"go-boilerplate/internal/golibs/redis"
)

// Common holds settings every service has. Name is always the service name
// passed on the command line; it cannot be overridden from a file.
type Common struct {
	Name            string        `mapstructure:"name"`
	Env             string        `mapstructure:"env"`       // local | dev | staging | prod
	LogLevel        string        `mapstructure:"log_level"` // debug | info | warn | error
	GRPCPort        int           `mapstructure:"grpc_port"`
	HTTPPort        int           `mapstructure:"http_port"`
	MetricsPort     int           `mapstructure:"metrics_port"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	// DrainDelay is how long /readyz reports DRAINING before servers stop, so
	// load balancers deregister the instance first. 0 locally, ~5-15s on AWS.
	DrainDelay time.Duration `mapstructure:"drain_delay"`
}

func (c Common) IsLocal() bool { return c.Env == "local" || c.Env == "dev" }

type JWT struct {
	Secret string `mapstructure:"secret"`
}

// Base is the part of the config the bootstrap understands. Infra sections are
// only connected when a service asks for them (bootstrap.Resources is lazy),
// so it is fine for common.yaml to describe infra a given service never uses.
//
// A service config embeds it with `mapstructure:",squash"`:
//
//	type Config struct {
//		configs.Base `mapstructure:",squash"`
//		Storage      StorageConfig `mapstructure:"storage"`
//	}
type Base struct {
	Common   Common            `mapstructure:"common"`
	Postgres database.DBConfig `mapstructure:"postgres"`
	Redis    redis.RedisConfig `mapstructure:"redis"`
	Kafka    kafka.KafkaConfig `mapstructure:"kafka"`
	JWT      JWT               `mapstructure:"jwt"`
}

// GetBase has a value receiver so any struct embedding Base satisfies Provider.
func (b Base) GetBase() Base { return b }

type Provider interface {
	GetBase() Base
}
