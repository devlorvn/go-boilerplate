package s3

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	client *s3.Client
	bucket string
	mu     sync.RWMutex
}

func NewS3Client(client *s3.Client, bucket string) *S3Client {
	return &S3Client{
		client: client,
		bucket: bucket,
	}
}

func (s *S3Client) SetBucket(bucket string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bucket = bucket
}

func (s *S3Client) GetBucket() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.bucket
}

func (s *S3Client) UploadFile(ctx context.Context, key string, file io.Reader, contentType string) (string, error) {
	bucket := s.GetBucket()
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("upload to s3 failed: %w", err)
	}

	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", bucket, key), nil
}

func (s *S3Client) GetFile(ctx context.Context, key string) (io.ReadCloser, error) {
	bucket := s.GetBucket()
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("get object %s from s3 failed: %w", key, err)
	}

	return output.Body, nil
}

func (s *S3Client) DeleteFile(ctx context.Context, key string) error {
	bucket := s.GetBucket()
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete object %s from s3 failed: %w", key, err)
	}

	return nil
}

func (s *S3Client) GetPresignedURL(ctx context.Context, key string, lifetime time.Duration) (string, error) {
	bucket := s.GetBucket()
	presignClient := s3.NewPresignClient(s.client)

	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(lifetime))

	if err != nil {
		return "", fmt.Errorf("presign get object %s failed: %w", key, err)
	}

	return req.URL, nil
}

func (s *S3Client) GetUploadPresignedURL(ctx context.Context, key string, lifetime time.Duration) (string, error) {
	bucket := s.GetBucket()
	presignClient := s3.NewPresignClient(s.client)

	req, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(lifetime))

	if err != nil {
		return "", fmt.Errorf("presign put object %s failed: %w", key, err)
	}

	return req.URL, nil
}
