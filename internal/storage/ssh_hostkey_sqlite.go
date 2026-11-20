package storage

import (
	"context"
	"database/sql"
	"errors"
)

// GetSSHHostKey retrieves a stored SSH host key for a specific host.
func (s *SQLiteStorage) GetSSHHostKey(ctx context.Context, host string) ([]byte, error) {
	var keyBytes []byte
	err := s.q.QueryRowContext(ctx, `
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
	now := nowUTC()
	_, err := execBuilt(ctx, s.q, s.q.builder().Insert("ssh_host_keys").
		Set("host", host).Set("key_data", key).Set("created_at", now).Set("updated_at", now).
		OnConflict("host").DoUpdate("key_data", "updated_at"))

	return err
}
