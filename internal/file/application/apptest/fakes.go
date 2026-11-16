// Package apptest provides in-memory implementations of the file ports for
// application-layer unit tests.
package apptest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"go-boilerplate/internal/file/application"
	"go-boilerplate/internal/file/domain"
)

type Files struct {
	mu        sync.Mutex
	byKey     map[string]*domain.File
	CreateErr error
	LastList  application.ListFilter
}

func NewFiles(seed ...*domain.File) *Files {
	f := &Files{byKey: map[string]*domain.File{}}
	for _, s := range seed {
		cp := *s
		f.byKey[s.Key] = &cp
	}
	return f
}

func (f *Files) Has(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.byKey[key]
	return ok
}

func (f *Files) Get(key string) *domain.File {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.byKey[key]
}

func (f *Files) Create(_ context.Context, file *domain.File) error {
	if f.CreateErr != nil {
		return f.CreateErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	file.ID = fmt.Sprintf("id-%d", len(f.byKey)+1)
	cp := *file
	f.byKey[file.Key] = &cp
	return nil
}

func (f *Files) GetByKey(_ context.Context, key string) (*domain.File, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if file, ok := f.byKey[key]; ok {
		cp := *file
		return &cp, nil
	}
	return nil, domain.ErrFileNotFound
}

func (f *Files) List(_ context.Context, filter application.ListFilter) ([]*domain.File, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.LastList = filter
	var out []*domain.File
	for _, file := range f.byKey {
		if filter.OwnerID != "" && file.OwnerID != filter.OwnerID {
			continue
		}
		if filter.Status != "" && file.Status != filter.Status {
			continue
		}
		cp := *file
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	total := int64(len(out))
	if filter.Offset < len(out) {
		out = out[filter.Offset:]
	} else {
		out = nil
	}
	if len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, total, nil
}

func (f *Files) ListPendingOlderThan(_ context.Context, cutoff time.Time) ([]*domain.File, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*domain.File
	for _, file := range f.byKey {
		if file.Status == domain.StatusPending && file.CreatedAt.Before(cutoff) {
			cp := *file
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *Files) UpdateStatus(_ context.Context, key string, status domain.Status) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if file, ok := f.byKey[key]; ok {
		file.Status = status
	}
	return nil
}

func (f *Files) Delete(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.byKey, key)
	return nil
}

type Storage struct {
	mu      sync.Mutex
	Objects map[string][]byte
	Deleted []string
}

func NewStorage() *Storage { return &Storage{Objects: map[string][]byte{}} }

func (s *Storage) Put(_ context.Context, key string, body io.Reader, _ string) (string, error) {
	b, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Objects[key] = b
	return "https://bucket.example/" + key, nil
}

func (s *Storage) PresignGet(_ context.Context, key string, ttl time.Duration) (string, error) {
	return fmt.Sprintf("https://bucket.example/%s?get&ttl=%s", key, ttl), nil
}

func (s *Storage) PresignPut(_ context.Context, key string, ttl time.Duration) (string, error) {
	return fmt.Sprintf("https://bucket.example/%s?put&ttl=%s", key, ttl), nil
}

func (s *Storage) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if key == "" {
		return errors.New("empty key")
	}
	delete(s.Objects, key)
	s.Deleted = append(s.Deleted, key)
	return nil
}
