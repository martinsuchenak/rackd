package auth

import (
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
