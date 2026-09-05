package configs

import (
	"fmt"
	"go-boilerplate/internal/golibs/database"
	"go-boilerplate/internal/golibs/kafka"
	"go-boilerplate/internal/golibs/redis"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config chứa toàn bộ cấu hình hệ thống, được nhóm theo từng công nghệ (Sub-structs)
type Config struct {
	App      AppConfig         `mapstructure:"APP"`
	Log      LogConfig         `mapstructure:"LOG"`
	GRPC     GRPCConfig        `mapstructure:"GRPC"`
	DB       database.DBConfig `mapstructure:"DB"`
	Redis    redis.RedisConfig `mapstructure:"REDIS"`
	Kafka    kafka.KafkaConfig `mapstructure:"KAFKA"`
	HTTP     HTTPConfig        `mapstructure:"HTTP"`
	Auth     AuthConfig        `mapstructure:"AUTH"`
	Provider ProviderConfig    `mapstructure:"PROVIDER"`
	Metrics  MetricsConfig     `mapstructure:"METRICS"`
}

type HTTPConfig struct {
	Port        int    `mapstructure:"PORT"`          // e.g. 8000
	AuthSvcAddr string `mapstructure:"AUTH_SVC_ADDR"` // e.g. "localhost:50051"
	FileSvcAddr string `mapstructure:"FILE_SVC_ADDR"` // e.g. "localhost:50052"
}

type AppConfig struct {
	Name string `mapstructure:"NAME"` // e.g. "bus-notification-service"
	Env  string `mapstructure:"ENV"`  // local | staging | prod
}

type LogConfig struct {
	Level string `mapstructure:"LEVEL"` // debug | info | warn | error
}

type GRPCConfig struct {
	Port    int           `mapstructure:"PORT"`    // 50051
	Timeout time.Duration `mapstructure:"TIMEOUT"` // 10s
}

type AuthConfig struct {
	JWTSecret string `mapstructure:"JWT_SECRET"`
}

type ProviderConfig struct {
	FCM FCMConfig `mapstructure:"FCM"`
	AWS AWSConfig `mapstructure:"AWS"`
}

type FCMConfig struct {
	ServerKey string `mapstructure:"SERVER_KEY"`
}

type AWSConfig struct {
	Region          string `mapstructure:"REGION"`
	AccessKeyID     string `mapstructure:"ACCESS_KEY_ID"`
	SecretAccessKey string `mapstructure:"SECRET_ACCESS_KEY"`
	SESFromEmail    string `mapstructure:"SES_FROM_EMAIL"`
}

type MetricsConfig struct {
	Enabled bool `mapstructure:"ENABLED"`
	Port    int  `mapstructure:"PORT"` // e.g. 2112
}

func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()
	v.AddConfigPath(configPath)
	v.SetConfigName(".env")
	v.SetConfigType("env")

	// Set Fallback Defaults
	v.SetDefault("APP.NAME", "authservice")
	v.SetDefault("APP.ENV", "local")
	v.SetDefault("LOG.LEVEL", "debug")
	v.SetDefault("HTTP.PORT", 8000)
	v.SetDefault("HTTP.AUTH_SVC_ADDR", "localhost:50051")
	v.SetDefault("HTTP.FILE_SVC_ADDR", "localhost:50052")
	v.SetDefault("GRPC.PORT", 50051)
	v.SetDefault("GRPC.TIMEOUT", "10s")
	v.SetDefault("DB.HOST", "localhost")
	v.SetDefault("DB.PORT", "5432")
	v.SetDefault("DB.USER", "postgres")
	v.SetDefault("DB.PASSWORD", "postgres")
	v.SetDefault("DB.DB_NAME", "auth_db")
	v.SetDefault("DB.SSL_MODE", "disable")
	v.SetDefault("REDIS.HOST", "localhost")
	v.SetDefault("REDIS.PORT", "6379")
	v.SetDefault("METRICS.PORT", 2112)

	// Hỗ trợ map môi trường dạng POSTGRES_HOST -> POSTGRES.HOST
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config file failed: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config failed: %w", err)
	}

	return &cfg, nil
}
