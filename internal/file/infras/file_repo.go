package infras

import (
	"context"
	"errors"
	"time"

	"go-boilerplate/internal/file/domain"

	"gorm.io/gorm"
)

type filePO struct {
	ID          string            `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	FileKey     string            `gorm:"uniqueIndex;not null"`
	FileName    string            `gorm:"not null"`
	FileSize    int64             `gorm:"default:0"`
	ContentType string            `gorm:"not null"`
	Status      domain.FileStatus `gorm:"not null;default:'PENDING'"`
	URL         string            `gorm:"default:''"`
	CreatedAt   time.Time         `gorm:"autoCreateTime"`
	UpdatedAt   time.Time         `gorm:"autoUpdateTime"`
}

func (p *filePO) TableName() string {
	return "files"
}

func (p *filePO) toDomain() *domain.File {
	if p == nil {
		return nil
	}
	return &domain.File{
		ID:          p.ID,
		FileKey:     p.FileKey,
		FileName:    p.FileName,
		FileSize:    p.FileSize,
		ContentType: p.ContentType,
		Status:      p.Status,
		URL:         p.URL,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func fromDomain(f *domain.File) *filePO {
	if f == nil {
		return nil
	}
	return &filePO{
		ID:          f.ID,
		FileKey:     f.FileKey,
		FileName:    f.FileName,
		FileSize:    f.FileSize,
		ContentType: f.ContentType,
		Status:      f.Status,
		URL:         f.URL,
		CreatedAt:   f.CreatedAt,
		UpdatedAt:   f.UpdatedAt,
	}
}

type GORMFileRepository struct {
	db *gorm.DB
}

func NewGORMFileRepository(db *gorm.DB) (*GORMFileRepository, error) {
	// Schema is owned by migrations/ (golang-migrate); never AutoMigrate here.
	if db == nil {
		return nil, errors.New("nil *gorm.DB")
	}
	return &GORMFileRepository{db: db}, nil
}

func (r *GORMFileRepository) Create(ctx context.Context, file *domain.File) error {
	if r.db == nil {
		return nil
	}
	po := fromDomain(file)
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return err
	}
	file.ID = po.ID
	file.CreatedAt = po.CreatedAt
	file.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *GORMFileRepository) UpdateStatus(ctx context.Context, fileKey string, status domain.FileStatus, url string) error {
	if r.db == nil {
		return nil
	}
	return r.db.WithContext(ctx).Model(&filePO{}).Where("file_key = ?", fileKey).Updates(map[string]interface{}{
		"status": status,
		"url":    url,
	}).Error
}

func (r *GORMFileRepository) GetByFileKey(ctx context.Context, fileKey string) (*domain.File, error) {
	if r.db == nil {
		return nil, domain.ErrFileNotFound
	}
	var po filePO
	if err := r.db.WithContext(ctx).First(&po, "file_key = ?", fileKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrFileNotFound
		}
		return nil, err
	}
	return po.toDomain(), nil
}

func (r *GORMFileRepository) GetExpiredPendingFiles(ctx context.Context, olderThan time.Duration) ([]*domain.File, error) {
	if r.db == nil {
		return nil, nil
	}
	cutoff := time.Now().Add(-olderThan)
	var pos []filePO
	if err := r.db.WithContext(ctx).Where("status = ? AND created_at < ?", domain.StatusPending, cutoff).Find(&pos).Error; err != nil {
		return nil, err
	}

	files := make([]*domain.File, len(pos))
	for i, po := range pos {
		files[i] = po.toDomain()
	}
	return files, nil
}

func (r *GORMFileRepository) GetAll(ctx context.Context, limit, offset int, status string) ([]*domain.File, int64, error) {
	if r.db == nil {
		return nil, 0, nil
	}
	query := r.db.WithContext(ctx).Model(&filePO{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	var pos []filePO
	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	files := make([]*domain.File, len(pos))
	for i, po := range pos {
		files[i] = po.toDomain()
	}
	return files, total, nil
}

func (r *GORMFileRepository) Delete(ctx context.Context, fileKey string) error {
	if r.db == nil {
		return nil
	}
	return r.db.WithContext(ctx).Where("file_key = ?", fileKey).Delete(&filePO{}).Error
}
