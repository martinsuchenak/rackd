package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
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

type SessionManager struct {
	sessions    map[string]*Session
	mu          sync.RWMutex
	sessionTTL  time.Duration
	cleanupTick *time.Ticker
	stopCleanup chan struct{}
}

func NewSessionManager(sessionTTL time.Duration) *SessionManager {
	sm := &SessionManager{
		sessions:    make(map[string]*Session),
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
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for token, session := range sm.sessions {
		if now.After(session.ExpiresAt) {
			delete(sm.sessions, token)
		}
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

	now := time.Now()
	session := &Session{
		Token:     token,
		UserID:    userID,
		Username:  username,
		IsAdmin:   isAdmin,
		CreatedAt: now,
		ExpiresAt: now.Add(sm.sessionTTL),
	}

	sm.mu.Lock()
	sm.sessions[token] = session
	sm.mu.Unlock()

	return session, nil
}

func (sm *SessionManager) GetSession(token string) (*Session, error) {
	if token == "" {
		return nil, ErrSessionNotFound
	}

	sm.mu.RLock()
	session, exists := sm.sessions[token]
	sm.mu.RUnlock()

	if !exists {
		return nil, ErrSessionNotFound
	}

	if time.Now().After(session.ExpiresAt) {
		sm.mu.Lock()
		delete(sm.sessions, token)
		sm.mu.Unlock()
		return nil, ErrSessionExpired
	}

	return session, nil
}

func (sm *SessionManager) RefreshSession(token string) (*Session, error) {
	session, err := sm.GetSession(token)
	if err != nil {
		return nil, err
	}

	sm.mu.Lock()
	session.ExpiresAt = time.Now().Add(sm.sessionTTL)
	sm.mu.Unlock()

	return session, nil
}

func (sm *SessionManager) InvalidateSession(token string) error {
	if token == "" {
		return ErrSessionNotFound
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.sessions[token]; !exists {
		return ErrSessionNotFound
	}

	delete(sm.sessions, token)
	return nil
}

func (sm *SessionManager) InvalidateUserSessions(userID string) {
	if userID == "" {
		return
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	for token, session := range sm.sessions {
		if session.UserID == userID {
			delete(sm.sessions, token)
		}
	}
}
