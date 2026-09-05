package domain

import (
	"context"
	"errors"
	"io"
	"time"
)

var (
	ErrFileNotFound = errors.New("file not found")
)

type FileStatus string

const (
	StatusPending   FileStatus = "PENDING"
	StatusCompleted FileStatus = "COMPLETED"
	StatusFailed    FileStatus = "FAILED"
)

type File struct {
	ID          string
	FileKey     string
	FileName    string
	FileSize    int64
	ContentType string
	Status      FileStatus
	URL         string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type FileRepository interface {
	Create(ctx context.Context, file *File) error
	UpdateStatus(ctx context.Context, fileKey string, status FileStatus, url string) error
	GetByFileKey(ctx context.Context, fileKey string) (*File, error)
	GetExpiredPendingFiles(ctx context.Context, olderThan time.Duration) ([]*File, error)
	Delete(ctx context.Context, fileKey string) error
}

type StorageRepository interface {
	Upload(ctx context.Context, key string, body io.Reader, contentType string) (url string, err error)
	GetPresignedURL(ctx context.Context, key string, ttl time.Duration) (string, error)
	GetUploadPresignedURL(ctx context.Context, key string, ttl time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
}
