package storage

import (
	"context"
	"database/sql"
	"errors"
)

// GetSSHHostKey retrieves a stored SSH host key for a specific host.
func (s *SQLiteStorage) GetSSHHostKey(ctx context.Context, host string) ([]byte, error) {
	var keyBytes []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT key_data
		FROM ssh_host_keys
		WHERE host = ?
	`, host).Scan(&keyBytes)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Not found, which is not an error for TOFU
		}
		return nil, err
	}

	return keyBytes, nil
}

// SaveSSHHostKey stores an SSH host key for TOFU persistence.
func (s *SQLiteStorage) SaveSSHHostKey(ctx context.Context, host string, key []byte) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO ssh_host_keys (host, key_data, created_at)
		VALUES (?, ?, ?)
		ON CONFLICT(host) DO UPDATE SET
			key_data = excluded.key_data,
			updated_at = ?
	`, host, key, nowUTC(), nowUTC())

	return err
}
