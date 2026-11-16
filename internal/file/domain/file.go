// Package domain is the file service's core model.
package domain

import (
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	"go-boilerplate/internal/golibs/authz"
	"go-boilerplate/internal/golibs/errorx"
)

var (
	// ErrFileNotFound is also returned when the file exists but belongs to
	// someone else, so callers cannot probe other users' keys.
	ErrFileNotFound = errorx.New(errorx.NotFound, "file not found")
)

type Status string

const (
	StatusPending   Status = "PENDING"
	StatusCompleted Status = "COMPLETED"
	StatusFailed    Status = "FAILED"
)

func ParseStatus(s string) (Status, error) {
	switch st := Status(strings.ToUpper(s)); st {
	case "", StatusPending, StatusCompleted, StatusFailed:
		return st, nil
	}
	return "", errorx.Invalid("unknown status %q", s)
}

type File struct {
	ID          string
	OwnerID     string // empty for files created before ownership existed
	Key         string
	Name        string
	Size        int64
	ContentType string
	Status      Status
	URL         string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// AccessibleBy: owners and admins only.
func (f *File) AccessibleBy(actor authz.Actor) bool {
	return actor.IsAdmin() || (f.OwnerID != "" && f.OwnerID == actor.UserID)
}

var folderPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}$`)

const DefaultFolder = "uploads"

// NewObjectKey builds "<folder>/<owner>/<id>_<name>". The owner segment keeps
// users' objects apart (handy for S3 lifecycle rules and IAM prefixes), and
// only the base name of fileName is kept so "../" tricks are harmless.
func NewObjectKey(folder, ownerID, id, fileName string) (string, error) {
	if folder == "" {
		folder = DefaultFolder
	}
	if !folderPattern.MatchString(folder) {
		return "", errorx.Invalid("folder must match %s", folderPattern)
	}
	name := path.Base(strings.ReplaceAll(fileName, `\`, "/"))
	if name == "." || name == "/" || name == "" {
		return "", errorx.Invalid("file name is required")
	}
	return fmt.Sprintf("%s/%s/%s_%s", folder, ownerID, id, name), nil
}

// Link lifetimes accepted from clients.
const (
	DefaultLinkTTL = time.Hour
	MaxLinkTTL     = 24 * time.Hour
)

// LinkTTL clamps a client-requested lifetime in seconds.
func LinkTTL(seconds int64) time.Duration {
	if seconds <= 0 {
		return DefaultLinkTTL
	}
	ttl := time.Duration(seconds) * time.Second
	if ttl > MaxLinkTTL {
		return MaxLinkTTL
	}
	return ttl
}
