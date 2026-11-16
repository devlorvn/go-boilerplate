package commands_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go-boilerplate/internal/file/application/apptest"
	"go-boilerplate/internal/file/application/commands"
	"go-boilerplate/internal/file/domain"
	"go-boilerplate/internal/golibs/authz"
	"go-boilerplate/internal/golibs/errorx"
)

var (
	alice = authz.Actor{UserID: "alice", Role: authz.RoleUser}
	bob   = authz.Actor{UserID: "bob", Role: authz.RoleUser}
	admin = authz.Actor{UserID: "root", Role: authz.RoleAdmin}
	t0    = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
)

func newHandler(files *apptest.Files, storage *apptest.Storage) *commands.FileCommandHandler {
	return &commands.FileCommandHandler{
		Files: files, Storage: storage,
		Now:   func() time.Time { return t0 },
		NewID: func() string { return "fixed-id" },
	}
}

func TestUpload(t *testing.T) {
	files, storage := apptest.NewFiles(), apptest.NewStorage()
	h := newHandler(files, storage)

	f, err := h.Upload(context.Background(), alice, commands.UploadCommand{FileName: "../../etc/passwd", Content: []byte("hi")})
	if err != nil {
		t.Fatal(err)
	}
	if f.Key != "uploads/alice/fixed-id_passwd" {
		t.Fatalf("key must be owner-scoped and path-safe, got %q", f.Key)
	}
	if f.OwnerID != "alice" || f.Status != domain.StatusCompleted || f.Size != 2 {
		t.Fatalf("unexpected file %+v", f)
	}
	if string(storage.Objects[f.Key]) != "hi" {
		t.Fatal("object not stored")
	}
}

func TestUpload_RollsBackObjectWhenMetadataFails(t *testing.T) {
	files, storage := apptest.NewFiles(), apptest.NewStorage()
	files.CreateErr = errors.New("db down")
	h := newHandler(files, storage)

	if _, err := h.Upload(context.Background(), alice, commands.UploadCommand{FileName: "a.txt", Content: []byte("x")}); err == nil {
		t.Fatal("expected error")
	}
	if len(storage.Objects) != 0 {
		t.Fatal("orphan object left in storage")
	}
}

func TestUpload_Validation(t *testing.T) {
	h := newHandler(apptest.NewFiles(), apptest.NewStorage())
	for name, cmd := range map[string]commands.UploadCommand{
		"bad folder": {FileName: "a.txt", Folder: "../other"},
		"no name":    {FileName: ""},
	} {
		if _, err := h.Upload(context.Background(), alice, cmd); errorx.KindOf(err) != errorx.InvalidArgument {
			t.Errorf("%s: want InvalidArgument, got %v", name, err)
		}
	}
}

func TestRequestUpload(t *testing.T) {
	files := apptest.NewFiles()
	h := newHandler(files, apptest.NewStorage())

	ticket, err := h.RequestUpload(context.Background(), alice, commands.RequestUploadCommand{FileName: "a.png", Folder: "avatars", TTLSeconds: 999999})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ticket.UploadURL, "ttl=24h0m0s") {
		t.Fatalf("ttl must be clamped to 24h: %s", ticket.UploadURL)
	}
	if f := files.Get(ticket.Key); f == nil || f.Status != domain.StatusPending || f.OwnerID != "alice" {
		t.Fatalf("pending row not recorded: %+v", f)
	}
}

func TestDelete_Ownership(t *testing.T) {
	seed := func() (*apptest.Files, *apptest.Storage) {
		files := apptest.NewFiles(&domain.File{Key: "k", OwnerID: "alice", Status: domain.StatusCompleted})
		storage := apptest.NewStorage()
		storage.Objects["k"] = []byte("x")
		return files, storage
	}

	files, storage := seed()
	err := newHandler(files, storage).Delete(context.Background(), bob, "k")
	if !errors.Is(err, domain.ErrFileNotFound) {
		t.Fatalf("other user's file must look not found, got %v", err)
	}
	if !files.Has("k") || len(storage.Deleted) != 0 {
		t.Fatal("non-owner must not delete anything")
	}

	for _, who := range []authz.Actor{alice, admin} {
		files, storage := seed()
		if err := newHandler(files, storage).Delete(context.Background(), who, "k"); err != nil {
			t.Fatalf("%s delete: %v", who.UserID, err)
		}
		if files.Has("k") || len(storage.Deleted) != 1 {
			t.Fatalf("%s: file not fully deleted", who.UserID)
		}
	}
}

func TestExpirePending(t *testing.T) {
	files := apptest.NewFiles(
		&domain.File{Key: "old", Status: domain.StatusPending, CreatedAt: t0.Add(-48 * time.Hour)},
		&domain.File{Key: "fresh", Status: domain.StatusPending, CreatedAt: t0.Add(-time.Hour)},
		&domain.File{Key: "done", Status: domain.StatusCompleted, CreatedAt: t0.Add(-48 * time.Hour)},
	)
	storage := apptest.NewStorage()

	n, err := newHandler(files, storage).ExpirePending(context.Background(), 24*time.Hour)
	if err != nil || n != 1 {
		t.Fatalf("expired %d, err %v", n, err)
	}
	if files.Get("old").Status != domain.StatusFailed || files.Get("fresh").Status != domain.StatusPending || files.Get("done").Status != domain.StatusCompleted {
		t.Fatal("wrong files expired")
	}
}
