package usecase

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"go-boilerplate/internal/file/domain"

	"github.com/google/uuid"
)

type FileUsecase struct {
	fileRepo    domain.FileRepository
	storageRepo domain.StorageRepository
}

func NewFileUsecase(fileRepo domain.FileRepository, storageRepo domain.StorageRepository) *FileUsecase {
	return &FileUsecase{
		fileRepo:    fileRepo,
		storageRepo: storageRepo,
	}
}

func (u *FileUsecase) UploadFile(ctx context.Context, fileName string, content []byte, contentType, folder string) (*domain.File, error) {
	if folder == "" {
		folder = "uploads"
	}
	fileKey := fmt.Sprintf("%s/%s_%s", folder, uuid.New().String(), fileName)

	url, err := u.storageRepo.Upload(ctx, fileKey, bytes.NewReader(content), contentType)
	if err != nil {
		return nil, fmt.Errorf("storage upload failed: %w", err)
	}

	file := &domain.File{
		FileKey:     fileKey,
		FileName:    fileName,
		FileSize:    int64(len(content)),
		ContentType: contentType,
		Status:      domain.StatusCompleted,
		URL:         url,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := u.fileRepo.Create(ctx, file); err != nil {
		return nil, fmt.Errorf("failed to save file metadata: %w", err)
	}

	return file, nil
}

func (u *FileUsecase) GetUploadPresignedUrl(ctx context.Context, fileName, contentType, folder string, expiresIn int64) (string, string, error) {
	if folder == "" {
		folder = "uploads"
	}
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	fileKey := fmt.Sprintf("%s/%s_%s", folder, uuid.New().String(), fileName)

	uploadURL, err := u.storageRepo.GetUploadPresignedURL(ctx, fileKey, time.Duration(expiresIn)*time.Second)
	if err != nil {
		return "", "", fmt.Errorf("failed to get presigned upload url: %w", err)
	}

	file := &domain.File{
		FileKey:     fileKey,
		FileName:    fileName,
		ContentType: contentType,
		Status:      domain.StatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := u.fileRepo.Create(ctx, file); err != nil {
		return "", "", fmt.Errorf("failed to save pending file metadata: %w", err)
	}

	return uploadURL, fileKey, nil
}

func (u *FileUsecase) GetPresignedUrl(ctx context.Context, fileKey string, expiresIn int64) (string, error) {
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	return u.storageRepo.GetPresignedURL(ctx, fileKey, time.Duration(expiresIn)*time.Second)
}

func (u *FileUsecase) DeleteFile(ctx context.Context, fileKey string) error {
	_ = u.storageRepo.Delete(ctx, fileKey)
	return u.fileRepo.Delete(ctx, fileKey)
}

func (u *FileUsecase) CleanupExpiredPendingFiles(ctx context.Context, olderThan time.Duration) (int, error) {
	expiredFiles, err := u.fileRepo.GetExpiredPendingFiles(ctx, olderThan)
	if err != nil {
		return 0, err
	}

	cleanedCount := 0
	for _, file := range expiredFiles {
		_ = u.storageRepo.Delete(ctx, file.FileKey)
		if err := u.fileRepo.UpdateStatus(ctx, file.FileKey, domain.StatusFailed, ""); err == nil {
			cleanedCount++
		}
	}
	return cleanedCount, nil
}
