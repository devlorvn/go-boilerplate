package kafka

import (
	"time"

	_kafka "github.com/segmentio/kafka-go"
)

const (
	defaultMaxAttempts             = 10
	defaultMaxBytes                = 10e6 // 10MB
	defaultReconnectAttempts       = 10
	defaultWaitingTimeToReconnect  = 5 * time.Second
	defaultRetryLogicAttempts      = 5
	defaultWaitingTimeToRetryLogic = 2 * time.Second
	defaultStrictCommit            = true
)

type kafkaConsumerOption struct {
	kafkaConsumerConfig     *_kafka.ReaderConfig
	reconnectAttempts       int
	retryLogicAttempts      int
	waitingTimeToReconnect  time.Duration
	waitingTimeToRetryLogic time.Duration
	strictCommit            bool
}

type KafkaConsumerOption func(opts *kafkaConsumerOption)

func newDefaultKafkaConsumerOption() *kafkaConsumerOption {
	return &kafkaConsumerOption{
		kafkaConsumerConfig:     &_kafka.ReaderConfig{},
		reconnectAttempts:       defaultReconnectAttempts,
		retryLogicAttempts:      defaultRetryLogicAttempts,
		waitingTimeToReconnect:  defaultWaitingTimeToReconnect,
		waitingTimeToRetryLogic: defaultWaitingTimeToRetryLogic,
		strictCommit:            defaultStrictCommit,
	}
}

func WithAutoCommit() KafkaConsumerOption {
	return func(opts *kafkaConsumerOption) {
		opts.strictCommit = false
	}
}

func WithRetryAttempts(attempts int, waitTime time.Duration) KafkaConsumerOption {
	return func(opts *kafkaConsumerOption) {
		opts.retryLogicAttempts = attempts
		opts.waitingTimeToRetryLogic = waitTime
	}
}
