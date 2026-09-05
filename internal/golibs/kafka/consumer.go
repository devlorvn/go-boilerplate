package kafka

import (
	"context"
	"go-boilerplate/internal/golibs/retry"

	"go.uber.org/zap"
)

type consumer interface {
	readAndHandle(ctx context.Context, handler MsgHandler) error
}

type strictConsumer struct {
	reader Reader
	logger *zap.Logger
	opts   *kafkaConsumerOption
}

func (c *strictConsumer) readAndHandle(ctx context.Context, handler MsgHandler) error {
	msg, err := c.reader.FetchMessage(ctx)
	if err != nil {
		return err
	}

	msgCtx := ExtractTraceContext(ctx, &msg.Headers)

	err = retry.DoWithCtx(msgCtx, func(reqCtx context.Context, attempt int) (bool, error) {
		shouldRetry, hErr := handler(reqCtx, msg.Value)
		if hErr != nil && attempt < c.opts.retryLogicAttempts && shouldRetry {
			c.logger.Warn("kafka handler failed, retrying...", zap.Int("attempt", attempt), zap.Error(hErr))
			return true, hErr
		}
		return false, hErr
	})

	if err != nil {
		c.logger.Error("kafka handler processing failed permanently", zap.Error(err))
		return err
	}

	return c.reader.CommitMessages(ctx, msg)
}
