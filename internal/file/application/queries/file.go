package queries

import (
	"context"

	"go-boilerplate/internal/file/application"
	"go-boilerplate/internal/file/domain"
	"go-boilerplate/internal/golibs/authz"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type ListQuery struct {
	Status string
	Limit  int
	Offset int
}

type FileQueryHandler struct {
	Files   application.FileRepository
	Storage application.ObjectStorage
}

// List returns the actor's files; admins see everyone's.
func (h *FileQueryHandler) List(ctx context.Context, actor authz.Actor, q ListQuery) ([]*domain.File, int64, error) {
	st, err := domain.ParseStatus(q.Status)
	if err != nil {
		return nil, 0, err
	}
	f := application.ListFilter{Status: st, Limit: q.Limit, Offset: max(q.Offset, 0)}
	if f.Limit <= 0 {
		f.Limit = DefaultPageSize
	}
	f.Limit = min(f.Limit, MaxPageSize)
	if !actor.IsAdmin() {
		f.OwnerID = actor.UserID
	}
	return h.Files.List(ctx, f)
}

// DownloadURL returns a short-lived presigned GET URL for a file the actor
// may access.
func (h *FileQueryHandler) DownloadURL(ctx context.Context, actor authz.Actor, key string, ttlSeconds int64) (string, error) {
	f, err := h.Files.GetByKey(ctx, key)
	if err != nil {
		return "", err
	}
	if !f.AccessibleBy(actor) {
		return "", domain.ErrFileNotFound
	}
	return h.Storage.PresignGet(ctx, key, domain.LinkTTL(ttlSeconds))
}
