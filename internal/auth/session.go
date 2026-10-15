package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
)

type Session struct {
	Token     string
	UserID    string
	Username  string
	IsAdmin   bool
	CreatedAt time.Time
	ExpiresAt time.Time
}

type SessionStore interface {
	Save(ctx context.Context, session *Session) error
	// Extend sets an existing session's expiry. Unlike Save it never creates
	// one, so refreshing can't bring back a session deleted meanwhile; it
	// returns ErrSessionNotFound when the session is gone.
	Extend(ctx context.Context, token string, expiresAt time.Time) error
	Get(ctx context.Context, token string) (*Session, error)
	Delete(ctx context.Context, token string) error
	DeleteByUser(ctx context.Context, userID string) error
	Cleanup(ctx context.Context) error
}

type SessionManager struct {
	store       SessionStore
	sessionTTL  time.Duration
	maxLifetime time.Duration
	cleanupTick *time.Ticker
	stopCleanup chan struct{}
}

func NewSessionManager(sessionTTL time.Duration, store SessionStore) *SessionManager {
	if store == nil {
		store = NewMemorySessionStore()
	}

	sm := &SessionManager{
		store:       store,
		sessionTTL:  sessionTTL,
		stopCleanup: make(chan struct{}),
	}

	sm.startCleanup()

	return sm
}

func (sm *SessionManager) startCleanup() {
	sm.cleanupTick = time.NewTicker(time.Minute)
	go func() {
		for {
			select {
			case <-sm.cleanupTick.C:
				sm.cleanupExpiredSessions()
			case <-sm.stopCleanup:
				sm.cleanupTick.Stop()
				return
			}
		}
	}()
}

func (sm *SessionManager) cleanupExpiredSessions() {
	if sm.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = sm.store.Cleanup(ctx)
	}
}

func (sm *SessionManager) Stop() {
	close(sm.stopCleanup)
}

func (sm *SessionManager) generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (sm *SessionManager) CreateSession(userID, username string, isAdmin bool) (*Session, error) {
	token, err := sm.generateToken()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	session := &Session{
		Token:     token,
		UserID:    userID,
		Username:  username,
		IsAdmin:   isAdmin,
		CreatedAt: now,
		ExpiresAt: now.Add(sm.sessionTTL),
	}

	if sm.store != nil {
		if err := sm.store.Save(context.Background(), hashSession(session)); err != nil {
			return nil, err
		}
	}

	return session, nil
}

func (sm *SessionManager) GetSession(token string) (*Session, error) {
	if token == "" {
		return nil, ErrSessionNotFound
	}

	if sm.store != nil {
		// Sessions are persisted with only the token hash, so a database or
		// Valkey dump cannot resurrect live sessions.
		session, err := sm.store.Get(context.Background(), HashToken(token))
		if err != nil {
			return nil, err
		}
		session.Token = token
		return session, nil
	}

	return nil, ErrSessionNotFound
}

// SetMaxLifetime limits how long a session lasts from sign-in, however often
// it is used; zero means no limit.
func (sm *SessionManager) SetMaxLifetime(d time.Duration) {
	sm.maxLifetime = d
}

// RefreshSession slides the session's expiry forward by the TTL, up to its
// maximum lifetime. It fails if the session has expired or was deleted.
func (sm *SessionManager) RefreshSession(token string) (*Session, error) {
	session, err := sm.GetSession(token)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	expires := now.Add(sm.sessionTTL)
	if sm.maxLifetime > 0 {
		limit := session.CreatedAt.Add(sm.maxLifetime)
		if !now.Before(limit) {
			_ = sm.InvalidateSession(token)
			return nil, ErrSessionExpired
		}
		if expires.After(limit) {
			expires = limit
		}
	}

	if sm.store != nil {
		if err := sm.store.Extend(context.Background(), HashToken(token), expires); err != nil {
			return nil, err
		}
	}
	session.ExpiresAt = expires
	return session, nil
}

func (sm *SessionManager) InvalidateSession(token string) error {
	if token == "" {
		return ErrSessionNotFound
	}

	if sm.store != nil {
		return sm.store.Delete(context.Background(), HashToken(token))
	}

	return nil
}

func (sm *SessionManager) InvalidateUserSessions(userID string) {
	if userID == "" {
		return
	}

	if sm.store != nil {
		_ = sm.store.DeleteByUser(context.Background(), userID)
	}
}

// hashSession returns a copy of the session with the token replaced by its
// SHA-256 hash for at-rest persistence.
func hashSession(session *Session) *Session {
	stored := *session
	stored.Token = HashToken(session.Token)
	return &stored
}
