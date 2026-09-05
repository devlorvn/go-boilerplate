package worker

import (
	"context"
	"time"

	"go-boilerplate/internal/file/usecase"

	"go.uber.org/zap"
)

type CleanupWorker struct {
	usecase  *usecase.FileUsecase
	logger   *zap.Logger
	interval time.Duration
	stopChan chan struct{}
}

func NewCleanupWorker(u *usecase.FileUsecase, logger *zap.Logger, interval time.Duration) *CleanupWorker {
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	return &CleanupWorker{
		usecase:  u,
		logger:   logger.Named("cleanup_worker"),
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

func (w *CleanupWorker) Start(ctx context.Context) {
	w.logger.Info("Starting expired pending file cleanup worker", zap.Duration("interval", w.interval))
	ticker := time.NewTicker(w.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				count, err := w.usecase.CleanupExpiredPendingFiles(ctx, 24*time.Hour)
				if err != nil {
					w.logger.Error("Failed to cleanup expired pending files", zap.Error(err))
				} else if count > 0 {
					w.logger.Info("Cleaned up expired pending files", zap.Int("count", count))
				}
			case <-w.stopChan:
				ticker.Stop()
				w.logger.Info("Cleanup worker stopped")
				return
			case <-ctx.Done():
				ticker.Stop()
				w.logger.Info("Cleanup worker stopped via context cancel")
				return
			}
		}
	}()
}

func (w *CleanupWorker) Stop() {
	close(w.stopChan)
}
