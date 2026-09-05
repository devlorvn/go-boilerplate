package kafka

import (
	"context"

	_kafka "github.com/segmentio/kafka-go"
)

type Writer interface {
	Close() error
	WriteMessages(ctx context.Context, msgs ..._kafka.Message) error
}

type Reader interface {
	Close() error
	CommitMessages(ctx context.Context, msgs ..._kafka.Message) error
	FetchMessage(ctx context.Context) (_kafka.Message, error)
	ReadMessage(ctx context.Context) (_kafka.Message, error)
}

type Conn interface {
	ApiVersions() ([]_kafka.ApiVersion, error)
	Close() error
	Controller() (broker _kafka.Broker, err error)
	CreateTopics(topics ..._kafka.TopicConfig) error
	ReadPartitions(topics ...string) (partitions []_kafka.Partition, err error)
}
type Kafkaer interface {
	DialConn(network string, address string) (Conn, error)
	NewWriter(config _kafka.WriterConfig) Writer
	NewReader(config _kafka.ReaderConfig) Reader
}
