// Package worker drives periodic file use cases; it is a transport like gRPC,
// only triggered by a ticker instead of a request.
package worker

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// PendingExpirer is implemented by commands.FileCommandHandler.
type PendingExpirer interface {
	ExpirePending(ctx context.Context, olderThan time.Duration) (int, error)
}

type Cleanup struct {
	Expirer   PendingExpirer
	Logger    *zap.Logger
	Interval  time.Duration // how often to run
	OlderThan time.Duration // how long an upload may stay PENDING
}

// Run blocks until ctx is cancelled, running once immediately and then on
// every tick.
func (c *Cleanup) Run(ctx context.Context) {
	log := c.Logger.Named("cleanup")
	log.Info("started", zap.Duration("interval", c.Interval), zap.Duration("older_than", c.OlderThan))
	ticker := time.NewTicker(c.Interval)
	defer ticker.Stop()

	for {
		n, err := c.Expirer.ExpirePending(ctx, c.OlderThan)
		switch {
		case err != nil && ctx.Err() == nil:
			log.Error("expire pending files", zap.Int("expired", n), zap.Error(err))
		case n > 0:
			log.Info("expired pending files", zap.Int("count", n))
		}

		select {
		case <-ctx.Done():
			log.Info("stopped")
			return
		case <-ticker.C:
		}
	}
}
