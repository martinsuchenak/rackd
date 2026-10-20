package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// APIKeyStorage defines the interface for API key storage
type APIKeyStorage interface {
	CreateAPIKey(ctx context.Context, key *model.APIKey) error
	GetAPIKey(ctx context.Context, id string) (*model.APIKey, error)
	GetAPIKeyByKey(ctx context.Context, key string) (*model.APIKey, error)
	ListAPIKeys(ctx context.Context, filter *model.APIKeyFilter) ([]model.APIKey, error)
	UpdateAPIKeyLastUsed(ctx context.Context, id string, lastUsed time.Time) error
	DeleteAPIKey(ctx context.Context, id string) error
}

// CreateAPIKey creates a new API key
func (s *SQLiteStorage) CreateAPIKey(ctx context.Context, key *model.APIKey) error {
	if key.ID == "" {
		key.ID = newUUID()
	}
	if key.CreatedAt.IsZero() {
		key.CreatedAt = nowUTC()
	}

	query := `INSERT INTO api_keys (id, name, key, user_id, description, created_at, last_used_at, expires_at, scopes)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var userID sql.NullString
	if key.UserID != "" {
		userID = sql.NullString{String: key.UserID, Valid: true}
	}

	_, err := s.db.ExecContext(ctx, query,
		key.ID, key.Name, key.Key, userID, key.Description,
		key.CreatedAt, key.LastUsedAt, key.ExpiresAt, nullString(strings.Join(key.Scopes, " ")),
	)
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	return nil
}

// apiKeyColumns are the columns scanAPIKey reads, in order.
const apiKeyColumns = `id, name, key, COALESCE(user_id, ''), description, created_at, last_used_at, expires_at, scopes`

func scanAPIKey(row interface{ Scan(dest ...any) error }) (*model.APIKey, error) {
	var key model.APIKey
	var lastUsedAt, expiresAt sql.NullTime
	var scopes sql.NullString
	if err := row.Scan(&key.ID, &key.Name, &key.Key, &key.UserID, &key.Description,
		&key.CreatedAt, &lastUsedAt, &expiresAt, &scopes); err != nil {
		return nil, err
	}
	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Time
	}
	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}
	key.Scopes = strings.Fields(scopes.String)
	return &key, nil
}

// GetAPIKey retrieves an API key by ID
func (s *SQLiteStorage) GetAPIKey(ctx context.Context, id string) (*model.APIKey, error) {
	if id == "" {
		return nil, ErrInvalidID
	}
	key, err := scanAPIKey(s.db.QueryRowContext(ctx, `SELECT `+apiKeyColumns+` FROM api_keys WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return nil, ErrAPIKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	return key, nil
}

// GetAPIKeyByKey retrieves an API key by the key string
func (s *SQLiteStorage) GetAPIKeyByKey(ctx context.Context, keyStr string) (*model.APIKey, error) {
	if keyStr == "" {
		return nil, fmt.Errorf("key cannot be empty")
	}
	key, err := scanAPIKey(s.db.QueryRowContext(ctx, `SELECT `+apiKeyColumns+` FROM api_keys WHERE key = ?`, keyStr))
	if err == sql.ErrNoRows {
		return nil, ErrAPIKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	return key, nil
}

// ListAPIKeys retrieves all API keys matching the filter
func (s *SQLiteStorage) ListAPIKeys(ctx context.Context, filter *model.APIKeyFilter) ([]model.APIKey, error) {
	query := `SELECT ` + apiKeyColumns + ` FROM api_keys`
	var conditions []string
	var args []any

	if filter != nil {
		if filter.Name != "" {
			conditions = append(conditions, "name LIKE ?")
			args = append(args, "%"+filter.Name+"%")
		}
		if filter.UserID != "" {
			conditions = append(conditions, "user_id = ?")
			args = append(args, filter.UserID)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			query += " AND " + c
		}
	}

	query += " ORDER BY created_at DESC"

	var pg *model.Pagination
	if filter != nil {
		pg = &filter.Pagination
	}
	query, args = appendPagination(query, args, pg)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var keys []model.APIKey
	for rows.Next() {
		key, err := scanAPIKey(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		keys = append(keys, *key)
	}

	if keys == nil {
		keys = []model.APIKey{}
	}

	return keys, nil
}

// UpdateAPIKeyLastUsed updates the last used timestamp
func (s *SQLiteStorage) UpdateAPIKeyLastUsed(ctx context.Context, id string, lastUsed time.Time) error {
	if id == "" {
		return ErrInvalidID
	}

	query := `UPDATE api_keys SET last_used_at = ? WHERE id = ?`

	_, err := s.db.ExecContext(ctx, query, lastUsed, id)
	if err != nil {
		return fmt.Errorf("failed to update API key last used: %w", err)
	}

	return nil
}

// DeleteAPIKey deletes an API key
func (s *SQLiteStorage) DeleteAPIKey(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidID
	}

	// Check if key exists
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM api_keys WHERE id = ?)`, id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check API key existence: %w", err)
	}
	if !exists {
		return ErrAPIKeyNotFound
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	return nil
}
