package auth

import (
	"context"
	"sync"
	"time"
)

// MemorySessionStore provides an in-memory implementation of SessionStore
type MemorySessionStore struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewMemorySessionStore creates a new MemorySessionStore
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		sessions: make(map[string]*Session),
	}
}

func (m *MemorySessionStore) Save(ctx context.Context, session *Session) error {
	m.mu.Lock()
	m.sessions[session.Token] = session
	m.mu.Unlock()
	return nil
}

func (m *MemorySessionStore) Get(ctx context.Context, token string) (*Session, error) {
	m.mu.RLock()
	session, exists := m.sessions[token]
	m.mu.RUnlock()

	if !exists {
		return nil, ErrSessionNotFound
	}

	if time.Now().UTC().After(session.ExpiresAt) {
		_ = m.Delete(ctx, token)
		return nil, ErrSessionExpired
	}

	return session, nil
}

func (m *MemorySessionStore) Delete(ctx context.Context, token string) error {
	m.mu.Lock()
	if _, exists := m.sessions[token]; !exists {
		m.mu.Unlock()
		return ErrSessionNotFound
	}
	delete(m.sessions, token)
	m.mu.Unlock()
	return nil
}

func (m *MemorySessionStore) DeleteByUser(ctx context.Context, userID string) error {
	m.mu.Lock()
	for token, session := range m.sessions {
		if session.UserID == userID {
			delete(m.sessions, token)
		}
	}
	m.mu.Unlock()
	return nil
}

func (m *MemorySessionStore) Cleanup(ctx context.Context) error {
	m.mu.Lock()
	now := time.Now().UTC()
	for token, session := range m.sessions {
		if now.After(session.ExpiresAt) {
			delete(m.sessions, token)
		}
	}
	m.mu.Unlock()
	return nil
}
