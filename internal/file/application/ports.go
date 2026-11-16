// Package application holds the file use cases: commands change state,
// queries read it. Infrastructure implements the ports below.
package application

import (
	"context"
	"io"
	"time"

	"go-boilerplate/internal/file/domain"
)

type ListFilter struct {
	OwnerID string // empty = all owners (admin only, enforced by queries)
	Status  domain.Status
	Limit   int
	Offset  int
}

type FileRepository interface {
	Create(ctx context.Context, f *domain.File) error
	// GetByKey returns domain.ErrFileNotFound when absent.
	GetByKey(ctx context.Context, key string) (*domain.File, error)
	List(ctx context.Context, filter ListFilter) ([]*domain.File, int64, error)
	ListPendingOlderThan(ctx context.Context, cutoff time.Time) ([]*domain.File, error)
	UpdateStatus(ctx context.Context, key string, status domain.Status) error
	Delete(ctx context.Context, key string) error
}

type ObjectStorage interface {
	Put(ctx context.Context, key string, body io.Reader, contentType string) (url string, err error)
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
	PresignPut(ctx context.Context, key string, ttl time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
}
