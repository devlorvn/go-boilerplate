package kafka

import (
	"context"
	"strings"

	_kafka "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
)

// KafkaHeaderCarrier implements OpenTelemetry's propagation.TextMapCarrier interface
type KafkaHeaderCarrier struct {
	Headers *[]_kafka.Header
}

func (c *KafkaHeaderCarrier) Get(key string) string {
	for _, h := range *c.Headers {
		if strings.EqualFold(h.Key, key) {
			return string(h.Value)
		}
	}
	return ""
}

func (c *KafkaHeaderCarrier) Set(key string, value string) {
	newHeaders := make([]_kafka.Header, 0, len(*c.Headers))
	for _, h := range *c.Headers {
		if !strings.EqualFold(h.Key, key) {
			newHeaders = append(newHeaders, h)
		}
	}
	newHeaders = append(newHeaders, _kafka.Header{
		Key:   key,
		Value: []byte(value),
	})
	*c.Headers = newHeaders
}

func (c *KafkaHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(*c.Headers))
	for _, h := range *c.Headers {
		keys = append(keys, h.Key)
	}
	return keys
}

// InjectTraceContext injects the active Trace ID into Kafka Message Headers before Publish
func InjectTraceContext(ctx context.Context, headers *[]_kafka.Header) {
	otel.GetTextMapPropagator().Inject(ctx, &KafkaHeaderCarrier{Headers: headers})
}

// ExtractTraceContext extracts Trace ID from Kafka Message Headers into context when Consuming
func ExtractTraceContext(ctx context.Context, headers *[]_kafka.Header) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, &KafkaHeaderCarrier{Headers: headers})
}
