// Package postgres implements application.UserRepository with GORM.
// The schema is owned by migrations/auth; never AutoMigrate here.
package postgres

import (
	"context"
	"errors"
	"time"

	"go-boilerplate/internal/auth/domain"

	"gorm.io/gorm"
)

// userRow is the persistence shape; domain.User stays free of ORM tags.
type userRow struct {
	ID           string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Email        string
	PasswordHash string
	Name         string
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (userRow) TableName() string { return "users" }

func (r *userRow) toDomain() *domain.User {
	return &domain.User{
		ID:           r.ID,
		Email:        r.Email,
		PasswordHash: r.PasswordHash,
		Name:         r.Name,
		Role:         r.Role,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	row := &userRow{
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Name:         u.Name,
		Role:         u.Role,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domain.ErrEmailAlreadyExists
		}
		return err
	}
	u.ID = row.ID
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return r.first(ctx, "id = ?", id)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.first(ctx, "email = ?", email)
}

func (r *UserRepository) first(ctx context.Context, where string, arg any) (*domain.User, error) {
	var row userRow
	err := r.db.WithContext(ctx).Where(where, arg).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}
