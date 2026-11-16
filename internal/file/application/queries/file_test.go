package queries_test

import (
	"context"
	"errors"
	"testing"

	"go-boilerplate/internal/file/application/apptest"
	"go-boilerplate/internal/file/application/queries"
	"go-boilerplate/internal/file/domain"
	"go-boilerplate/internal/golibs/authz"
	"go-boilerplate/internal/golibs/errorx"
)

var (
	alice = authz.Actor{UserID: "alice", Role: authz.RoleUser}
	bob   = authz.Actor{UserID: "bob", Role: authz.RoleUser}
	admin = authz.Actor{UserID: "root", Role: authz.RoleAdmin}
)

func seeded() *apptest.Files {
	return apptest.NewFiles(
		&domain.File{Key: "a1", OwnerID: "alice", Status: domain.StatusCompleted},
		&domain.File{Key: "a2", OwnerID: "alice", Status: domain.StatusPending},
		&domain.File{Key: "b1", OwnerID: "bob", Status: domain.StatusCompleted},
		&domain.File{Key: "legacy", OwnerID: "", Status: domain.StatusCompleted},
	)
}

func TestList_ScopedToOwner(t *testing.T) {
	files := seeded()
	h := &queries.FileQueryHandler{Files: files}
	ctx := context.Background()

	got, total, err := h.List(ctx, alice, queries.ListQuery{})
	if err != nil || total != 2 || len(got) != 2 {
		t.Fatalf("alice sees %d/%d files, err %v", len(got), total, err)
	}
	if _, total, _ := h.List(ctx, admin, queries.ListQuery{}); total != 4 {
		t.Fatalf("admin must see all 4 files, got %d", total)
	}
	if _, total, _ := h.List(ctx, alice, queries.ListQuery{Status: "pending"}); total != 1 {
		t.Fatalf("status filter (case-insensitive): got %d", total)
	}
	if _, _, err := h.List(ctx, alice, queries.ListQuery{Status: "weird"}); errorx.KindOf(err) != errorx.InvalidArgument {
		t.Fatalf("bad status: %v", err)
	}

	_, _, _ = h.List(ctx, alice, queries.ListQuery{Limit: 10_000, Offset: -5})
	if files.LastList.Limit != queries.MaxPageSize || files.LastList.Offset != 0 {
		t.Fatalf("paging not clamped: %+v", files.LastList)
	}
}

func TestDownloadURL_Ownership(t *testing.T) {
	h := &queries.FileQueryHandler{Files: seeded(), Storage: apptest.NewStorage()}
	ctx := context.Background()

	if _, err := h.DownloadURL(ctx, alice, "a1", 0); err != nil {
		t.Fatalf("owner: %v", err)
	}
	if _, err := h.DownloadURL(ctx, admin, "b1", 0); err != nil {
		t.Fatalf("admin: %v", err)
	}
	for _, key := range []string{"a1", "legacy", "missing"} {
		if _, err := h.DownloadURL(ctx, bob, key, 0); !errors.Is(err, domain.ErrFileNotFound) {
			t.Fatalf("bob -> %s: want not found, got %v", key, err)
		}
	}
}
