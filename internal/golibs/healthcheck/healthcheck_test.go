package healthcheck

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbe(t *testing.T) {
	dbErr := error(nil)
	p := NewProbe(func() map[string]Check {
		return map[string]Check{"postgres": func(context.Context) error { return dbErr }}
	})
	call := func(h http.HandlerFunc) int {
		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		return rec.Code
	}

	if got := call(p.Ready); got != http.StatusOK {
		t.Fatalf("ready with healthy deps = %d", got)
	}

	dbErr = errors.New("connection refused")
	if got := call(p.Ready); got != http.StatusServiceUnavailable {
		t.Fatalf("ready with failing dep = %d", got)
	}
	if got := call(p.Live); got != http.StatusOK {
		t.Fatalf("live must ignore dependencies, got %d", got)
	}

	dbErr = nil
	p.SetDraining()
	if got := call(p.Ready); got != http.StatusServiceUnavailable {
		t.Fatalf("ready while draining = %d", got)
	}
	if got := call(p.Live); got != http.StatusOK {
		t.Fatalf("live while draining = %d", got)
	}
}
