// Package kafka implements application.EventPublisher.
package kafka

import (
	"context"
	"encoding/json"
	"time"

	"go-boilerplate/internal/auth/domain"
	"go-boilerplate/internal/golibs/kafka"
)

const TopicUserEvents = "user-events"

// UserCreated is the public event contract. It is deliberately not
// domain.User, which carries the password hash.
type UserCreated struct {
	Type      string    `json:"type"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type UserEventPublisher struct {
	km kafka.KafkaManagement
}

func NewUserEventPublisher(km kafka.KafkaManagement) *UserEventPublisher {
	return &UserEventPublisher{km: km}
}

func (p *UserEventPublisher) PublishUserCreated(ctx context.Context, u *domain.User) error {
	payload, err := json.Marshal(UserCreated{
		Type:      "user.created",
		UserID:    u.ID,
		Email:     u.Email,
		Name:      u.Name,
		CreatedAt: u.CreatedAt,
	})
	if err != nil {
		return err
	}
	return p.km.PublishContext(ctx, TopicUserEvents, []byte(u.ID), payload)
}
