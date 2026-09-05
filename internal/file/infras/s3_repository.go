package infras

import (
	"context"
	"io"
	"time"

	"go-boilerplate/internal/golibs/s3"
)

type S3StorageRepository struct {
	s3Client *s3.S3Client
}

func NewS3StorageRepository(s3Client *s3.S3Client) *S3StorageRepository {
	return &S3StorageRepository{s3Client: s3Client}
}

func (s *S3StorageRepository) Upload(ctx context.Context, key string, body io.Reader, contentType string) (string, error) {
	if s.s3Client == nil {
		return "https://mock-bucket.s3.amazonaws.com/" + key, nil
	}
	return s.s3Client.UploadFile(ctx, key, body, contentType)
}

func (s *S3StorageRepository) GetPresignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if s.s3Client == nil {
		return "https://mock-bucket.s3.amazonaws.com/" + key + "?presigned=true", nil
	}
	return s.s3Client.GetPresignedURL(ctx, key, ttl)
}

func (s *S3StorageRepository) GetUploadPresignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if s.s3Client == nil {
		return "https://mock-bucket.s3.amazonaws.com/" + key + "?upload_presigned=true", nil
	}
	return s.s3Client.GetUploadPresignedURL(ctx, key, ttl)
}

func (s *S3StorageRepository) Delete(ctx context.Context, key string) error {
	if s.s3Client == nil {
		return nil
	}
	return s.s3Client.DeleteFile(ctx, key)
}
