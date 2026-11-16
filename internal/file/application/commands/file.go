package commands

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"go-boilerplate/internal/file/application"
	"go-boilerplate/internal/file/domain"
	"go-boilerplate/internal/golibs/authz"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type UploadCommand struct {
	FileName    string
	ContentType string
	Folder      string
	Content     []byte
}

type RequestUploadCommand struct {
	FileName    string
	ContentType string
	Folder      string
	TTLSeconds  int64
}

type UploadTicket struct {
	UploadURL string
	Key       string
}

type FileCommandHandler struct {
	Files   application.FileRepository
	Storage application.ObjectStorage
	Logger  *zap.Logger
	Now     func() time.Time
	NewID   func() string
}

func (h *FileCommandHandler) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

func (h *FileCommandHandler) newID() string {
	if h.NewID != nil {
		return h.NewID()
	}
	return uuid.NewString()
}

func (h *FileCommandHandler) newFile(actor authz.Actor, folder, name, contentType string, status domain.Status) (*domain.File, error) {
	key, err := domain.NewObjectKey(folder, actor.UserID, h.newID(), name)
	if err != nil {
		return nil, err
	}
	now := h.now()
	return &domain.File{
		OwnerID: actor.UserID, Key: key, Name: name, ContentType: contentType,
		Status: status, CreatedAt: now, UpdatedAt: now,
	}, nil
}

// Upload stores small files sent through gRPC directly.
func (h *FileCommandHandler) Upload(ctx context.Context, actor authz.Actor, cmd UploadCommand) (*domain.File, error) {
	f, err := h.newFile(actor, cmd.Folder, cmd.FileName, cmd.ContentType, domain.StatusCompleted)
	if err != nil {
		return nil, err
	}
	url, err := h.Storage.Put(ctx, f.Key, bytes.NewReader(cmd.Content), cmd.ContentType)
	if err != nil {
		return nil, fmt.Errorf("put object: %w", err)
	}
	f.URL, f.Size = url, int64(len(cmd.Content))
	if err := h.Files.Create(ctx, f); err != nil {
		// Do not leave an orphan object behind.
		if delErr := h.Storage.Delete(ctx, f.Key); delErr != nil {
			h.warn("delete orphan object", f.Key, delErr)
		}
		return nil, fmt.Errorf("save file metadata: %w", err)
	}
	return f, nil
}

// RequestUpload records a PENDING file and returns a presigned PUT URL so the
// client uploads straight to object storage, bypassing our servers.
func (h *FileCommandHandler) RequestUpload(ctx context.Context, actor authz.Actor, cmd RequestUploadCommand) (*UploadTicket, error) {
	f, err := h.newFile(actor, cmd.Folder, cmd.FileName, cmd.ContentType, domain.StatusPending)
	if err != nil {
		return nil, err
	}
	url, err := h.Storage.PresignPut(ctx, f.Key, domain.LinkTTL(cmd.TTLSeconds))
	if err != nil {
		return nil, fmt.Errorf("presign upload: %w", err)
	}
	if err := h.Files.Create(ctx, f); err != nil {
		return nil, fmt.Errorf("save pending file: %w", err)
	}
	return &UploadTicket{UploadURL: url, Key: f.Key}, nil
}

// Delete removes a file the actor owns (or any file, for admins). The
// metadata row goes first: a leftover object is invisible and cleanable,
// a leftover row would point at nothing.
func (h *FileCommandHandler) Delete(ctx context.Context, actor authz.Actor, key string) error {
	f, err := h.Files.GetByKey(ctx, key)
	if err != nil {
		return err
	}
	if !f.AccessibleBy(actor) {
		return domain.ErrFileNotFound
	}
	if err := h.Files.Delete(ctx, key); err != nil {
		return err
	}
	if err := h.Storage.Delete(ctx, key); err != nil {
		h.warn("delete object", key, err)
	}
	return nil
}

// ExpirePending marks uploads that never finished as FAILED and removes any
// partial object. Called periodically by transport/worker.
func (h *FileCommandHandler) ExpirePending(ctx context.Context, olderThan time.Duration) (int, error) {
	files, err := h.Files.ListPendingOlderThan(ctx, h.now().Add(-olderThan))
	if err != nil {
		return 0, err
	}
	var errs []error
	expired := 0
	for _, f := range files {
		if err := h.Storage.Delete(ctx, f.Key); err != nil {
			h.warn("delete expired object", f.Key, err)
		}
		if err := h.Files.UpdateStatus(ctx, f.Key, domain.StatusFailed); err != nil {
			errs = append(errs, err)
			continue
		}
		expired++
	}
	return expired, errors.Join(errs...)
}

func (h *FileCommandHandler) warn(msg, key string, err error) {
	if h.Logger != nil {
		h.Logger.Warn(msg, zap.String("key", key), zap.Error(err))
	}
}
