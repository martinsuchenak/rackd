package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

// A request refreshing a session must not bring it back after it was
// deleted, for example by a password reset.
func TestRefreshDoesNotRecreateADeletedSession(t *testing.T) {
	sm := NewSessionManager(time.Hour, NewMemorySessionStore())
	defer sm.Stop()
	s, err := sm.CreateSession("u1", "alice", false)
	if err != nil {
		t.Fatal(err)
	}
	sm.InvalidateUserSessions("u1")
	if _, err := sm.RefreshSession(s.Token); err == nil {
		t.Fatal("refresh succeeded for a deleted session")
	}
	if _, err := sm.GetSession(s.Token); err == nil {
		t.Error("the deleted session exists again")
	}
}

func TestSessionsEndAtTheirMaximumLifetime(t *testing.T) {
	sm := NewSessionManager(time.Hour, NewMemorySessionStore())
	defer sm.Stop()
	sm.SetMaxLifetime(50 * time.Millisecond)
	s, err := sm.CreateSession("u1", "alice", false)
	if err != nil {
		t.Fatal(err)
	}
	refreshed, err := sm.RefreshSession(s.Token)
	if err != nil {
		t.Fatal(err)
	}
	if limit := s.CreatedAt.Add(50 * time.Millisecond); refreshed.ExpiresAt.After(limit) {
		t.Errorf("refresh extended the session to %v, past its limit %v", refreshed.ExpiresAt, limit)
	}
	time.Sleep(60 * time.Millisecond)
	if _, err := sm.RefreshSession(s.Token); !errors.Is(err, ErrSessionExpired) && !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("refresh after the lifetime: %v, want expired", err)
	}
}

// countingStore counts expiry writes.
type countingStore struct {
	SessionStore
	extends int
}

func (c *countingStore) Extend(ctx context.Context, token string, expiresAt time.Time) error {
	c.extends++
	return c.SessionStore.Extend(ctx, token, expiresAt)
}

// Refreshing on every request wrote to the database on every request; a page
// load's API calls then queued for the write lock (slow on network storage).
func TestRefreshWritesAtMostOncePerInterval(t *testing.T) {
	store := &countingStore{SessionStore: NewMemorySessionStore()}
	sm := NewSessionManager(time.Hour, store)
	defer sm.Stop()
	s, err := sm.CreateSession("u1", "alice", false)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		if _, err := sm.RefreshSession(s.Token); err != nil {
			t.Fatal(err)
		}
	}
	if store.extends != 0 {
		t.Fatalf("%d expiry writes for requests within a minute of creation, want 0", store.extends)
	}

	// Once the stored expiry is a minute or more behind, a refresh writes it.
	if err := store.SessionStore.Extend(context.Background(), HashToken(s.Token), time.Now().UTC().Add(50*time.Minute)); err != nil {
		t.Fatal(err)
	}
	refreshed, err := sm.RefreshSession(s.Token)
	if err != nil {
		t.Fatal(err)
	}
	if store.extends != 1 || time.Until(refreshed.ExpiresAt) < 59*time.Minute {
		t.Fatalf("after the expiry fell behind: %d writes, expires in %v; want 1 write and about an hour", store.extends, time.Until(refreshed.ExpiresAt))
	}
}
