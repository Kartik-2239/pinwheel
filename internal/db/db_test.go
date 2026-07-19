package db

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Kartik-2239/pinwheel/internal/utils"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	database, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return NewStore(database)
}

func TestCreateAndGetUser(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)

	hash := utils.HashString("sk-proxy-testsecret")
	user := &User{Name: "alice", APIKeyHash: hash, Last4Digits: "cret"}
	if err := s.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// default rate limit applied
	if user.RateLimit5hr == nil || *user.RateLimit5hr != DefaultRateLimit5hr {
		t.Fatalf("expected default rate limit %d, got %v", DefaultRateLimit5hr, user.RateLimit5hr)
	}

	got, err := s.GetUserByHash(ctx, hash)
	if err != nil {
		t.Fatalf("GetUserByHash: %v", err)
	}
	if got.Name != "alice" || got.Last4Digits != "cret" || got.TokensUsed != 0 {
		t.Fatalf("unexpected user: %+v", got)
	}

	if _, err := s.GetUserByHash(ctx, utils.HashString("nonexistent")); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateUserValidation(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)

	if err := s.CreateUser(ctx, &User{APIKeyHash: "h"}); err == nil {
		t.Fatal("expected error for empty name")
	}
	if err := s.CreateUser(ctx, &User{Name: "bob"}); err == nil {
		t.Fatal("expected error for empty api_key_hash")
	}
}

func TestUpdateUserUsage(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)

	user := &User{Name: "bob", APIKeyHash: utils.HashString("k1"), Last4Digits: "0001"}
	if err := s.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// usage accumulates across calls
	if err := s.UpdateUserUsage(ctx, user.ID, 100); err != nil {
		t.Fatalf("UpdateUserUsage: %v", err)
	}
	if err := s.UpdateUserUsage(ctx, user.ID, 250); err != nil {
		t.Fatalf("UpdateUserUsage: %v", err)
	}

	got, err := s.GetUserByHash(ctx, user.APIKeyHash)
	if err != nil {
		t.Fatalf("GetUserByHash: %v", err)
	}
	if got.TokensUsed != 350 {
		t.Fatalf("expected 350 tokens used, got %d", got.TokensUsed)
	}
	if got.LastUsedAt == nil {
		t.Fatal("expected LastUsedAt to be set")
	}

	if err := s.UpdateUserUsage(ctx, 9999, 1); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound for unknown id, got %v", err)
	}
}
