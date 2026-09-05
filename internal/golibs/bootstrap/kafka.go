package bootstrap

import (
	"context"
	"time"

	"go.uber.org/zap"

	"go-boilerplate/internal/golibs/kafka"
)

type KafkaConsumerRegistrar interface {
	RegisterConsumers(ctx context.Context, km kafka.KafkaManagement) error
}

type KafkaBootstrapConfig struct {
	ServiceName     string        `json:"service_name"`
	Brokers         []string      `json:"brokers"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
}

type KafkaConsumerBootstrap struct {
	Config          KafkaBootstrapConfig
	Logger          *zap.Logger
	KafkaManagement kafka.KafkaManagement
}

func NewKafkaConsumerBootstrap(cfg KafkaBootstrapConfig, logger *zap.Logger) *KafkaConsumerBootstrap {

	km := kafka.NewKafkaManagement(cfg.Brokers, logger)

	return &KafkaConsumerBootstrap{
		Config:          cfg,
		Logger:          logger,
		KafkaManagement: km,
	}
}

func (kb *KafkaConsumerBootstrap) Start(ctx context.Context, registrar KafkaConsumerRegistrar) error {
	if err := registrar.RegisterConsumers(ctx, kb.KafkaManagement); err != nil {
		kb.Logger.Error("Fail to register Kafka Consumers", zap.Error(err))
		return err
	}

	kb.Logger.Info("Kafka Consumers registered & listening...")

	return nil
}

func (kb *KafkaConsumerBootstrap) Stop(ctx context.Context) error {
	kb.Logger.Info("Shutting down Kafka Consumers...")
	_, cancel := context.WithTimeout(ctx, kb.Config.ShutdownTimeout)
	defer cancel()
	if err := kb.KafkaManagement.Close(); err != nil {
		kb.Logger.Error("Fail to close Kafka Management gracefully", zap.Error(err))
		return err
	}
	kb.Logger.Info("Graceful shutdown Kafka Consumers done")
	return nil
}
