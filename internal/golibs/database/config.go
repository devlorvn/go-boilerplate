package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DBConfig struct {
	Host         string        `mapstructure:"HOST"`
	Port         string        `mapstructure:"PORT"`
	User         string        `mapstructure:"USER"`
	Password     string        `mapstructure:"PASSWORD"`
	DBName       string        `mapstructure:"DB_NAME"`
	SSLMode      string        `mapstructure:"SSL_MODE"`
	MaxOpenConns int           `mapstructure:"MAX_OPEN_CONNS"`
	MaxIdleConns int           `mapstructure:"MAX_IDLE_CONNS"`
	MaxLifetime  time.Duration `mapstructure:"MAX_LIFETIME"`
}

func (p DBConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		p.Host, p.Port, p.User, p.Password, p.DBName, p.SSLMode)
}

func (c *DBConfig) MigrationConnectionString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode,
	)
}

func NewGORM(cfg DBConfig, isDebug bool) (*gorm.DB, error) {
	logLevel := logger.Silent
	if isDebug {
		logLevel = logger.Info
	}
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql db: %v", err)
	}
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	if cfg.MaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.MaxLifetime)
	} else {
		sqlDB.SetConnMaxLifetime(1 * time.Hour)
	}

	return db, nil
}
