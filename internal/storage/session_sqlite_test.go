package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/pkg/model"
)

func TestSQLiteSessionStoreLifecycle(t *testing.T) {
	storage := newTestStorage(t)
	defer func() { _ = storage.Close() }()

	store := NewSQLiteSessionStore(storage.DB())
	ctx := context.Background()
	user := &model.User{
		ID:           "user-1",
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "hash",
		IsActive:     true,
	}
	if err := storage.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	session := &auth.Session{
		Token:     "token-1",
		UserID:    "user-1",
		Username:  "alice",
		IsAdmin:   true,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().Add(time.Hour).UTC(),
	}
	if err := store.Save(ctx, session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	got, err := store.Get(ctx, session.Token)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.UserID != session.UserID || !got.IsAdmin {
		t.Fatalf("unexpected session after save: %+v", got)
	}

	session.Username = "alice-updated"
	if err := store.Save(ctx, session); err != nil {
		t.Fatalf("Save update failed: %v", err)
	}
	got, err = store.Get(ctx, session.Token)
	if err != nil {
		t.Fatalf("Get after update failed: %v", err)
	}
	if got.Username != "alice-updated" {
		t.Fatalf("expected updated username, got %+v", got)
	}

	if err := store.Delete(ctx, session.Token); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, err := store.Get(ctx, session.Token); err != auth.ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestSQLiteSessionStoreExpiryCleanupAndDeleteByUser(t *testing.T) {
	storage := newTestStorage(t)
	defer func() { _ = storage.Close() }()

	store := NewSQLiteSessionStore(storage.DB())
	ctx := context.Background()
	user := &model.User{
		ID:           "user-1",
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "hash",
		IsActive:     true,
	}
	if err := storage.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	expired := &auth.Session{
		Token:     "expired",
		UserID:    "user-1",
		Username:  "alice",
		CreatedAt: time.Now().Add(-2 * time.Hour).UTC(),
		ExpiresAt: time.Now().Add(-time.Hour).UTC(),
	}
	active := &auth.Session{
		Token:     "active",
		UserID:    "user-1",
		Username:  "alice",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().Add(time.Hour).UTC(),
	}
	for _, sess := range []*auth.Session{expired, active} {
		if err := store.Save(ctx, sess); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	if _, err := store.Get(ctx, expired.Token); err != auth.ErrSessionExpired {
		t.Fatalf("expected ErrSessionExpired, got %v", err)
	}

	if err := store.Cleanup(ctx); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	if err := store.DeleteByUser(ctx, active.UserID); err != nil {
		t.Fatalf("DeleteByUser failed: %v", err)
	}
	if _, err := store.Get(ctx, active.Token); err != auth.ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound after DeleteByUser, got %v", err)
	}
}

// Extend only updates; a session deleted meanwhile stays deleted.
func TestSQLiteSessionStoreExtendNeverRecreates(t *testing.T) {
	storage := newTestStorage(t)
	defer func() { _ = storage.Close() }()
	store := NewSQLiteSessionStore(storage.DB())
	ctx := context.Background()
	if err := storage.CreateUser(ctx, &model.User{ID: "user-1", Username: "alice", Email: "a@example.com", PasswordHash: "h", IsActive: true}); err != nil {
		t.Fatal(err)
	}
	session := &auth.Session{Token: "token-1", UserID: "user-1", Username: "alice", CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().Add(time.Hour).UTC()}
	if err := store.Save(ctx, session); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(2 * time.Hour).UTC()
	if err := store.Extend(ctx, "token-1", later); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, "token-1")
	if err != nil || got.ExpiresAt.Sub(later).Abs() > time.Second {
		t.Fatalf("extended session = %+v, %v", got, err)
	}
	if err := store.DeleteByUser(ctx, "user-1"); err != nil {
		t.Fatal(err)
	}
	if err := store.Extend(ctx, "token-1", later); !errors.Is(err, auth.ErrSessionNotFound) {
		t.Errorf("Extend of a deleted session: %v, want ErrSessionNotFound", err)
	}
	if _, err := store.Get(ctx, "token-1"); err == nil {
		t.Error("Extend recreated the deleted session")
	}
}
