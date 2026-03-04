package auth

import (
	"testing"
	"time"
)

func TestNewSessionManager(t *testing.T) {
	sm := NewSessionManager(time.Hour, nil)
	if sm == nil {
		t.Fatal("NewSessionManager() returned nil")
	}
	defer sm.Stop()

	if sm.store == nil {
		t.Error("NewSessionManager() store not initialized")
	}
}

func TestCreateSession(t *testing.T) {
	sm := NewSessionManager(time.Hour, nil)
	defer sm.Stop()

	session, err := sm.CreateSession("user123", "testuser", true)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if session.Token == "" {
		t.Error("CreateSession() returned empty token")
	}

	if session.UserID != "user123" {
		t.Errorf("CreateSession() UserID = %v, want user123", session.UserID)
	}

	if session.Username != "testuser" {
		t.Errorf("CreateSession() Username = %v, want testuser", session.Username)
	}

	if !session.IsAdmin {
		t.Error("CreateSession() IsAdmin = false, want true")
	}

	if time.Since(session.CreatedAt) > time.Second {
		t.Error("CreateSession() CreatedAt is too old")
	}

	expectedExpiry := time.Now().Add(time.Hour)
	if session.ExpiresAt.Before(expectedExpiry.Add(-time.Second)) || session.ExpiresAt.After(expectedExpiry.Add(time.Second)) {
		t.Errorf("CreateSession() ExpiresAt = %v, want ~%v", session.ExpiresAt, expectedExpiry)
	}
}

func TestGetSession(t *testing.T) {
	sm := NewSessionManager(time.Hour, nil)
	defer sm.Stop()

	created, _ := sm.CreateSession("user123", "testuser", true)

	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{
			name:    "valid session",
			token:   created.Token,
			wantErr: nil,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: ErrSessionNotFound,
		},
		{
			name:    "invalid token",
			token:   "invalidtoken",
			wantErr: ErrSessionNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := sm.GetSession(tt.token)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("GetSession() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetSession() unexpected error: %v", err)
			}

			if session.UserID != created.UserID {
				t.Errorf("GetSession() UserID = %v, want %v", session.UserID, created.UserID)
			}
		})
	}
}

func TestGetSessionExpired(t *testing.T) {
	sm := NewSessionManager(10*time.Millisecond, nil)
	defer sm.Stop()

	session, _ := sm.CreateSession("user123", "testuser", true)

	time.Sleep(20 * time.Millisecond)

	_, err := sm.GetSession(session.Token)
	if err != ErrSessionExpired {
		t.Errorf("GetSession() error = %v, want %v", err, ErrSessionExpired)
	}
}

func TestRefreshSession(t *testing.T) {
	sm := NewSessionManager(time.Hour, nil)
	defer sm.Stop()

	session, _ := sm.CreateSession("user123", "testuser", true)
	time.Sleep(10 * time.Millisecond)

	originalExpiry := session.ExpiresAt

	refreshed, err := sm.RefreshSession(session.Token)
	if err != nil {
		t.Fatalf("RefreshSession() error = %v", err)
	}

	if refreshed.UserID != session.UserID {
		t.Errorf("RefreshSession() UserID = %v, want %v", refreshed.UserID, session.UserID)
	}

	if !refreshed.ExpiresAt.After(originalExpiry) {
		t.Error("RefreshSession() Expiry not extended")
	}
}

func TestInvalidateSession(t *testing.T) {
	sm := NewSessionManager(time.Hour, nil)
	defer sm.Stop()

	session, _ := sm.CreateSession("user123", "testuser", true)

	err := sm.InvalidateSession(session.Token)
	if err != nil {
		t.Fatalf("InvalidateSession() error = %v", err)
	}

	_, err = sm.GetSession(session.Token)
	if err != ErrSessionNotFound {
		t.Errorf("GetSession() after InvalidateSession() error = %v, want %v", err, ErrSessionNotFound)
	}
}

func TestInvalidateSessionNotFound(t *testing.T) {
	sm := NewSessionManager(time.Hour, nil)
	defer sm.Stop()

	err := sm.InvalidateSession("nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("InvalidateSession() error = %v, want %v", err, ErrSessionNotFound)
	}
}

func TestInvalidateUserSessions(t *testing.T) {
	sm := NewSessionManager(time.Hour, nil)
	defer sm.Stop()

	session1, _ := sm.CreateSession("user123", "testuser1", false)
	session2, _ := sm.CreateSession("user456", "testuser2", false)
	session3, _ := sm.CreateSession("user123", "testuser1", true)

	sm.InvalidateUserSessions("user123")

	_, err1 := sm.GetSession(session1.Token)
	if err1 != ErrSessionNotFound {
		t.Errorf("GetSession() after InvalidateUserSessions() error = %v, want %v", err1, ErrSessionNotFound)
	}

	_, err3 := sm.GetSession(session3.Token)
	if err3 != ErrSessionNotFound {
		t.Errorf("GetSession() after InvalidateUserSessions() error = %v, want %v", err3, ErrSessionNotFound)
	}

	_, err2 := sm.GetSession(session2.Token)
	if err2 != nil {
		t.Errorf("GetSession() for different user error = %v, want nil", err2)
	}
}

func TestCleanupExpiredSessions(t *testing.T) {
	sm := NewSessionManager(10*time.Millisecond, nil)
	defer sm.Stop()

	session1, _ := sm.CreateSession("user1", "testuser1", false)
	time.Sleep(15 * time.Millisecond)
	session2, _ := sm.CreateSession("user2", "testuser2", false)

	sm.cleanupExpiredSessions()

	_, err1 := sm.GetSession(session1.Token)
	if err1 == nil {
		t.Error("Expired session should be cleaned up")
	}

	_, err2 := sm.GetSession(session2.Token)
	if err2 != nil {
		t.Errorf("Valid session should not be cleaned up, error: %v", err2)
	}
}
