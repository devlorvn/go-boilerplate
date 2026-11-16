// Package s3 implements application.ObjectStorage on AWS S3 (or any
// S3-compatible store such as MinIO/LocalStack).
package s3

import (
	"context"
	"io"
	"time"

	s3lib "go-boilerplate/internal/golibs/s3"
)

type Storage struct {
	client *s3lib.S3Client
}

func NewStorage(client *s3lib.S3Client) *Storage {
	return &Storage{client: client}
}

func (s *Storage) Put(ctx context.Context, key string, body io.Reader, contentType string) (string, error) {
	return s.client.UploadFile(ctx, key, body, contentType)
}

func (s *Storage) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	return s.client.GetPresignedURL(ctx, key, ttl)
}

func (s *Storage) PresignPut(ctx context.Context, key string, ttl time.Duration) (string, error) {
	return s.client.GetUploadPresignedURL(ctx, key, ttl)
}

func (s *Storage) Delete(ctx context.Context, key string) error {
	return s.client.DeleteFile(ctx, key)
}
