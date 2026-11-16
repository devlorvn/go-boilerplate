package jwtutil

import (
	"testing"
	"time"
)

func TestHS256_SignVerify(t *testing.T) {
	h, err := NewHS256("secret")
	if err != nil {
		t.Fatal(err)
	}
	tok, err := h.Sign("u1", "user", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	c, err := h.Verify(tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if c.UserID != "u1" || c.Role != "user" {
		t.Fatalf("unexpected claims: %+v", c)
	}
}

func TestHS256_Rejects(t *testing.T) {
	h, _ := NewHS256("secret")
	other, _ := NewHS256("other")

	expired, _ := h.Sign("u1", "user", -time.Minute)
	foreign, _ := other.Sign("u1", "user", time.Minute)

	for name, tok := range map[string]string{"expired": expired, "wrong secret": foreign, "garbage": "abc.def.ghi", "empty": ""} {
		if _, err := h.Verify(tok); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestNewHS256_EmptySecret(t *testing.T) {
	if _, err := NewHS256(""); err == nil {
		t.Fatal("expected error for empty secret")
	}
}
