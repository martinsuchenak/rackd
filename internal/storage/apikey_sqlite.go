package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

// APIKeyStorage defines the interface for API key storage
type APIKeyStorage interface {
	CreateAPIKey(key *model.APIKey) error
	GetAPIKey(id string) (*model.APIKey, error)
	GetAPIKeyByKey(key string) (*model.APIKey, error)
	ListAPIKeys(filter *model.APIKeyFilter) ([]model.APIKey, error)
	UpdateAPIKeyLastUsed(id string, lastUsed time.Time) error
	DeleteAPIKey(id string) error
}

// CreateAPIKey creates a new API key
func (s *SQLiteStorage) CreateAPIKey(key *model.APIKey) error {
	if key.ID == "" {
		key.ID = newUUID()
	}
	if key.CreatedAt.IsZero() {
		key.CreatedAt = time.Now()
	}

	ctx := context.Background()
	query := `INSERT INTO api_keys (id, name, key, description, created_at, last_used_at, expires_at) 
	          VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query,
		key.ID, key.Name, key.Key, key.Description,
		key.CreatedAt, key.LastUsedAt, key.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	return nil
}

// GetAPIKey retrieves an API key by ID
func (s *SQLiteStorage) GetAPIKey(id string) (*model.APIKey, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	ctx := context.Background()
	query := `SELECT id, name, key, description, created_at, last_used_at, expires_at 
	          FROM api_keys WHERE id = ?`

	var key model.APIKey
	var lastUsedAt, expiresAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&key.ID, &key.Name, &key.Key, &key.Description,
		&key.CreatedAt, &lastUsedAt, &expiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("API key not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Time
	}
	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}

	return &key, nil
}

// GetAPIKeyByKey retrieves an API key by the key string
func (s *SQLiteStorage) GetAPIKeyByKey(keyStr string) (*model.APIKey, error) {
	if keyStr == "" {
		return nil, fmt.Errorf("key cannot be empty")
	}

	ctx := context.Background()
	query := `SELECT id, name, key, description, created_at, last_used_at, expires_at 
	          FROM api_keys WHERE key = ?`

	var key model.APIKey
	var lastUsedAt, expiresAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, keyStr).Scan(
		&key.ID, &key.Name, &key.Key, &key.Description,
		&key.CreatedAt, &lastUsedAt, &expiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("API key not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Time
	}
	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}

	return &key, nil
}

// ListAPIKeys retrieves all API keys matching the filter
func (s *SQLiteStorage) ListAPIKeys(filter *model.APIKeyFilter) ([]model.APIKey, error) {
	ctx := context.Background()
	query := `SELECT id, name, key, description, created_at, last_used_at, expires_at 
	          FROM api_keys`
	var args []interface{}

	if filter != nil && filter.Name != "" {
		query += " WHERE name LIKE ?"
		args = append(args, "%"+filter.Name+"%")
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	var keys []model.APIKey
	for rows.Next() {
		var key model.APIKey
		var lastUsedAt, expiresAt sql.NullTime

		if err := rows.Scan(
			&key.ID, &key.Name, &key.Key, &key.Description,
			&key.CreatedAt, &lastUsedAt, &expiresAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}

		if lastUsedAt.Valid {
			key.LastUsedAt = &lastUsedAt.Time
		}
		if expiresAt.Valid {
			key.ExpiresAt = &expiresAt.Time
		}

		keys = append(keys, key)
	}

	if keys == nil {
		keys = []model.APIKey{}
	}

	return keys, nil
}

// UpdateAPIKeyLastUsed updates the last used timestamp
func (s *SQLiteStorage) UpdateAPIKeyLastUsed(id string, lastUsed time.Time) error {
	if id == "" {
		return ErrInvalidID
	}

	ctx := context.Background()
	query := `UPDATE api_keys SET last_used_at = ? WHERE id = ?`

	_, err := s.db.ExecContext(ctx, query, lastUsed, id)
	if err != nil {
		return fmt.Errorf("failed to update API key last used: %w", err)
	}

	return nil
}

// DeleteAPIKey deletes an API key
func (s *SQLiteStorage) DeleteAPIKey(id string) error {
	if id == "" {
		return ErrInvalidID
	}

	ctx := context.Background()

	// Check if key exists
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM api_keys WHERE id = ?)`, id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check API key existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("API key not found")
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	return nil
}
