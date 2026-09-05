package kafka

import _kafka "github.com/segmentio/kafka-go"

type kafkaerImpl struct{}

func newKafkaImpl() *kafkaerImpl {
	return &kafkaerImpl{}
}

func (n *kafkaerImpl) NewWriter(config _kafka.WriterConfig) Writer {
	return _kafka.NewWriter(config)
}

func (n *kafkaerImpl) NewReader(config _kafka.ReaderConfig) Reader {
	return _kafka.NewReader(config)
}

func (n *kafkaerImpl) DialConn(network string, address string) (Conn, error) {
	return _kafka.Dial(network, address)
}
