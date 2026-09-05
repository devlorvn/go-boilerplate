package infras

import (
	"context"
	"encoding/json"

	"go-boilerplate/internal/auth/domain"
	"go-boilerplate/internal/golibs/kafka"
)

type KafkaEventPublisher struct {
	km kafka.KafkaManagement
}

func NewKafkaEventPublisher(km kafka.KafkaManagement) *KafkaEventPublisher {
	return &KafkaEventPublisher{km: km}
}

func (p *KafkaEventPublisher) PublishUserCreated(ctx context.Context, user *domain.User) error {
	if p.km == nil {
		return nil
	}
	payload, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return p.km.PublishContext(ctx, "user-events", []byte(user.ID), payload)
}
