package infras

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-boilerplate/internal/auth/domain"

	"gorm.io/gorm"
)

type userPO struct {
	ID           string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Email        string    `gorm:"uniqueIndex;not null"`
	PasswordHash string    `gorm:"not null"`
	Name         string    `gorm:"not null"`
	Role         string    `gorm:"not null;default:'user'"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

func (p *userPO) TableName() string {
	return "users"
}

func (p *userPO) toDomain() *domain.User {
	if p == nil {
		return nil
	}
	return &domain.User{
		ID:           p.ID,
		Email:        p.Email,
		PasswordHash: p.PasswordHash,
		Name:         p.Name,
		Role:         p.Role,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

func fromDomain(u *domain.User) *userPO {
	if u == nil {
		return nil
	}
	return &userPO{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Name:         u.Name,
		Role:         u.Role,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

type GORMUserRepository struct {
	db *gorm.DB
}

func NewGORMUserRepository(db *gorm.DB) (*GORMUserRepository, error) {
	if db != nil {
		if err := db.AutoMigrate(&userPO{}); err != nil {
			return nil, fmt.Errorf("failed to automigrate User table: %w", err)
		}
	}
	return &GORMUserRepository{db: db}, nil
}

func (r *GORMUserRepository) Create(ctx context.Context, user *domain.User) error {
	if r.db == nil {
		return nil
	}
	po := fromDomain(user)
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domain.ErrEmailAlreadyExists
		}
		return err
	}
	user.ID = po.ID
	user.CreatedAt = po.CreatedAt
	user.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *GORMUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if r.db == nil {
		return &domain.User{ID: id, Email: "mock@example.com", Name: "Mock User", Role: "user"}, nil
	}
	var po userPO
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return po.toDomain(), nil
}

func (r *GORMUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if r.db == nil {
		return nil, domain.ErrUserNotFound
	}
	var po userPO
	if err := r.db.WithContext(ctx).First(&po, "email = ?", email).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return po.toDomain(), nil
}
