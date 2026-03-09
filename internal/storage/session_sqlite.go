package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/martinsuchenak/rackd/internal/auth"
)

// SQLiteSessionStore provides a SQLite-backed implementation of auth.SessionStore
type SQLiteSessionStore struct {
	db *sql.DB
}

// NewSQLiteSessionStore creates a new SQLiteSessionStore
func NewSQLiteSessionStore(db *sql.DB) *SQLiteSessionStore {
	return &SQLiteSessionStore{db: db}
}

// Save stores or updates a session
func (s *SQLiteSessionStore) Save(ctx context.Context, session *auth.Session) error {
	isAdminInt := 0
	if session.IsAdmin {
		isAdminInt = 1
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (token, user_id, username, is_admin, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(token) DO UPDATE SET
			expires_at = excluded.expires_at,
			user_id = excluded.user_id,
			username = excluded.username,
			is_admin = excluded.is_admin
	`, session.Token, session.UserID, session.Username, isAdminInt, session.CreatedAt.UTC(), session.ExpiresAt.UTC())
	return err
}

// Get retrieves a session by token
func (s *SQLiteSessionStore) Get(ctx context.Context, token string) (*auth.Session, error) {
	var sess auth.Session
	var createdAt, expiresAt time.Time
	var isAdminInt int

	err := s.db.QueryRowContext(ctx, `
		SELECT token, user_id, username, is_admin, created_at, expires_at
		FROM sessions
		WHERE token = ?
	`, token).Scan(&sess.Token, &sess.UserID, &sess.Username, &isAdminInt, &createdAt, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrSessionNotFound
		}
		return nil, err
	}

	sess.IsAdmin = isAdminInt == 1
	sess.CreatedAt = createdAt
	sess.ExpiresAt = expiresAt

	if nowUTC().After(sess.ExpiresAt) {
		_ = s.Delete(ctx, token)
		return nil, auth.ErrSessionExpired
	}

	return &sess, nil
}

// Delete removes a session by token
func (s *SQLiteSessionStore) Delete(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE token = ?", token)
	return err
}

// DeleteByUser removes all sessions for a specific user
func (s *SQLiteSessionStore) DeleteByUser(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE user_id = ?", userID)
	return err
}

// Cleanup removes all expired sessions
func (s *SQLiteSessionStore) Cleanup(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < ?", nowUTC())
	return err
}
