package kafka

import (
	"context"
	"fmt"
	"sync"

	_kafka "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`
	GroupID string   `mapstructure:"group_id"`
}

type MsgHandler func(ctx context.Context, data []byte) (shouldRetry bool, err error)

type KafkaManagement interface {
	PublishContext(ctx context.Context, topic string, key []byte, value []byte) error
	Consume(ctx context.Context, topic string, groupID string, handler MsgHandler, opts ...KafkaConsumerOption) error
	UpsertTopic(topicConfig _kafka.TopicConfig) error
	Close() error
}

type kafkaManagementImpl struct {
	brokers           []string
	logger            *zap.Logger
	mutex             sync.RWMutex
	writerMap         map[string]Writer
	consumerWaitGroup sync.WaitGroup
	kafkaer           Kafkaer
}

func NewKafkaManagement(brokers []string, logger *zap.Logger) KafkaManagement {
	return &kafkaManagementImpl{
		brokers:   brokers,
		logger:    logger.Named("kafka"),
		writerMap: make(map[string]Writer),
		kafkaer:   newKafkaImpl(),
	}
}

func (k *kafkaManagementImpl) PublishContext(ctx context.Context, topic string, key []byte, value []byte) error {
	k.mutex.Lock()
	writer, exists := k.writerMap[topic]
	if !exists {
		writer = k.kafkaer.NewWriter(_kafka.WriterConfig{
			Brokers: k.brokers,
			Topic:   topic,
		})
		k.writerMap[topic] = writer
	}
	k.mutex.Unlock()
	msg := _kafka.Message{
		Key:   key,
		Value: value,
	}
	InjectTraceContext(ctx, &msg.Headers)
	err := writer.WriteMessages(ctx, msg)
	if err != nil {
		k.logger.Error("failed to publish kafka message", zap.String("topic", topic), zap.Error(err))
		return err
	}
	return nil
}

func (k *kafkaManagementImpl) Consume(ctx context.Context, topic string, groupID string, handler MsgHandler, opts ...KafkaConsumerOption) error {
	options := newDefaultKafkaConsumerOption()
	for _, opt := range opts {
		opt(options)
	}

	reader := k.kafkaer.NewReader(_kafka.ReaderConfig{
		Brokers: k.brokers,
		GroupID: groupID,
		Topic:   topic,
	})

	sc := &strictConsumer{reader: reader, logger: k.logger, opts: options}

	k.consumerWaitGroup.Add(1)
	go func() {
		defer k.consumerWaitGroup.Done()
		defer reader.Close()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := sc.readAndHandle(ctx, handler); err != nil {
					k.logger.Error("error processing message", zap.Error(err))
				}
			}
		}
	}()

	return nil
}

func (k *kafkaManagementImpl) UpsertTopic(topicConfig _kafka.TopicConfig) error {
	conn, err := k.kafkaer.DialConn("tcp", k.brokers[0])
	if err != nil {
		return fmt.Errorf("dial kafka failed: %w", err)
	}
	defer conn.Close()

	return conn.CreateTopics(topicConfig)
}

func (k *kafkaManagementImpl) Close() error {
	k.logger.Info("shutting down kafka management...")

	k.mutex.Lock()
	for _, writer := range k.writerMap {
		_ = writer.Close()
	}
	k.mutex.Unlock()

	k.consumerWaitGroup.Wait()
	k.logger.Info("kafka management shutdown complete")
	return nil
}
