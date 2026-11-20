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
	q portableDB
}

// Sessions returns a session store on this storage's database, whichever
// backend it is.
func (s *SQLiteStorage) Sessions() *SQLiteSessionStore {
	return &SQLiteSessionStore{q: s.q}
}

// Save stores or updates a session
func (s *SQLiteSessionStore) Save(ctx context.Context, session *auth.Session) error {
	_, err := execBuilt(ctx, s.q, s.q.builder().Insert("sessions").
		Set("token", session.Token).Set("user_id", session.UserID).Set("username", session.Username).
		Set("is_admin", session.IsAdmin).Set("created_at", session.CreatedAt.UTC()).Set("expires_at", session.ExpiresAt.UTC()).
		OnConflict("token").DoUpdate("expires_at", "user_id", "username", "is_admin"))
	return err
}

// Extend sets an existing session's expiry; it never creates a session.
func (s *SQLiteSessionStore) Extend(ctx context.Context, token string, expiresAt time.Time) error {
	res, err := s.q.ExecContext(ctx, `UPDATE sessions SET expires_at = ? WHERE token = ?`, expiresAt.UTC(), token)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return auth.ErrSessionNotFound
	}
	return nil
}

// Get retrieves a session by token
func (s *SQLiteSessionStore) Get(ctx context.Context, token string) (*auth.Session, error) {
	var sess auth.Session
	var createdAt, expiresAt time.Time

	err := s.q.QueryRowContext(ctx, `
		SELECT token, user_id, username, is_admin, created_at, expires_at
		FROM sessions
		WHERE token = ?
	`, token).Scan(&sess.Token, &sess.UserID, &sess.Username, &sess.IsAdmin, &createdAt, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrSessionNotFound
		}
		return nil, err
	}

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
	_, err := s.q.ExecContext(ctx, "DELETE FROM sessions WHERE token = ?", token)
	return err
}

// DeleteByUser removes all sessions for a specific user
func (s *SQLiteSessionStore) DeleteByUser(ctx context.Context, userID string) error {
	_, err := s.q.ExecContext(ctx, "DELETE FROM sessions WHERE user_id = ?", userID)
	return err
}

// Cleanup removes all expired sessions
func (s *SQLiteSessionStore) Cleanup(ctx context.Context) error {
	_, err := s.q.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < ?", nowUTC())
	return err
}
