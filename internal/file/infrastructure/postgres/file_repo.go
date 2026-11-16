// Package postgres implements application.FileRepository with GORM.
// The schema is owned by migrations/file; never AutoMigrate here.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"go-boilerplate/internal/file/application"
	"go-boilerplate/internal/file/domain"

	"gorm.io/gorm"
)

type fileRow struct {
	ID          string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	OwnerID     sql.NullString
	FileKey     string
	FileName    string
	FileSize    int64
	ContentType string
	Status      string
	URL         string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (fileRow) TableName() string { return "files" }

func (r *fileRow) toDomain() *domain.File {
	return &domain.File{
		ID:          r.ID,
		OwnerID:     r.OwnerID.String,
		Key:         r.FileKey,
		Name:        r.FileName,
		Size:        r.FileSize,
		ContentType: r.ContentType,
		Status:      domain.Status(r.Status),
		URL:         r.URL,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

type FileRepository struct {
	db *gorm.DB
}

func NewFileRepository(db *gorm.DB) *FileRepository {
	return &FileRepository{db: db}
}

func (r *FileRepository) Create(ctx context.Context, f *domain.File) error {
	row := &fileRow{
		OwnerID:     sql.NullString{String: f.OwnerID, Valid: f.OwnerID != ""},
		FileKey:     f.Key,
		FileName:    f.Name,
		FileSize:    f.Size,
		ContentType: f.ContentType,
		Status:      string(f.Status),
		URL:         f.URL,
		CreatedAt:   f.CreatedAt,
		UpdatedAt:   f.UpdatedAt,
	}
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return err
	}
	f.ID = row.ID
	return nil
}

func (r *FileRepository) GetByKey(ctx context.Context, key string) (*domain.File, error) {
	var row fileRow
	err := r.db.WithContext(ctx).Where("file_key = ?", key).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrFileNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *FileRepository) List(ctx context.Context, f application.ListFilter) ([]*domain.File, int64, error) {
	q := r.db.WithContext(ctx).Model(&fileRow{})
	if f.OwnerID != "" {
		q = q.Where("owner_id = ?", f.OwnerID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", string(f.Status))
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []fileRow
	if err := q.Order("created_at DESC").Limit(f.Limit).Offset(f.Offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return toDomainList(rows), total, nil
}

func (r *FileRepository) ListPendingOlderThan(ctx context.Context, cutoff time.Time) ([]*domain.File, error) {
	var rows []fileRow
	err := r.db.WithContext(ctx).
		Where("status = ? AND created_at < ?", string(domain.StatusPending), cutoff).
		Limit(500). // bounded batch; the worker runs again on the next tick
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return toDomainList(rows), nil
}

func (r *FileRepository) UpdateStatus(ctx context.Context, key string, status domain.Status) error {
	return r.db.WithContext(ctx).Model(&fileRow{}).Where("file_key = ?", key).
		Updates(map[string]any{"status": string(status), "updated_at": time.Now()}).Error
}

func (r *FileRepository) Delete(ctx context.Context, key string) error {
	return r.db.WithContext(ctx).Where("file_key = ?", key).Delete(&fileRow{}).Error
}

func toDomainList(rows []fileRow) []*domain.File {
	out := make([]*domain.File, len(rows))
	for i := range rows {
		out[i] = rows[i].toDomain()
	}
	return out
}
