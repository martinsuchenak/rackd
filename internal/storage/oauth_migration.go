package storage

import (
	"context"
	"database/sql"
	"fmt"
)

func migrateAddOAuthTablesUp(ctx context.Context, tx *sql.Tx) error {
	// OAuth clients table
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS oauth_clients (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			secret_hash TEXT,
			redirect_uris TEXT NOT NULL DEFAULT '[]',
			grant_types TEXT NOT NULL DEFAULT '[]',
			response_types TEXT NOT NULL DEFAULT '[]',
			token_endpoint_auth TEXT NOT NULL DEFAULT 'none',
			scope TEXT,
			client_uri TEXT,
			logo_uri TEXT,
			is_confidential INTEGER NOT NULL DEFAULT 0,
			created_by_user_id TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			FOREIGN KEY (created_by_user_id) REFERENCES users(id)
		)
	`); err != nil {
		return fmt.Errorf("failed to create oauth_clients table: %w", err)
	}

	// OAuth authorization codes table
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS oauth_authorization_codes (
			code_hash TEXT PRIMARY KEY,
			client_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			redirect_uri TEXT NOT NULL,
			scope TEXT,
			code_challenge TEXT NOT NULL,
			code_challenge_method TEXT NOT NULL DEFAULT 'S256',
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL,
			used INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (client_id) REFERENCES oauth_clients(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`); err != nil {
		return fmt.Errorf("failed to create oauth_authorization_codes table: %w", err)
	}

	// OAuth tokens table (access and refresh tokens)
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS oauth_tokens (
			id TEXT PRIMARY KEY,
			token_type TEXT NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			client_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			scope TEXT,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL,
			revoked_at DATETIME,
			parent_token_id TEXT,
			FOREIGN KEY (client_id) REFERENCES oauth_clients(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`); err != nil {
		return fmt.Errorf("failed to create oauth_tokens table: %w", err)
	}

	// Indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_oauth_clients_created_by ON oauth_clients(created_by_user_id)",
		"CREATE INDEX IF NOT EXISTS idx_oauth_codes_client ON oauth_authorization_codes(client_id)",
		"CREATE INDEX IF NOT EXISTS idx_oauth_codes_user ON oauth_authorization_codes(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_oauth_codes_expires ON oauth_authorization_codes(expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_oauth_tokens_hash ON oauth_tokens(token_hash)",
		"CREATE INDEX IF NOT EXISTS idx_oauth_tokens_client ON oauth_tokens(client_id)",
		"CREATE INDEX IF NOT EXISTS idx_oauth_tokens_user ON oauth_tokens(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_oauth_tokens_expires ON oauth_tokens(expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_oauth_tokens_parent ON oauth_tokens(parent_token_id)",
	}

	for _, idx := range indexes {
		if _, err := tx.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create oauth index: %w", err)
		}
	}

	return nil
}

func migrateAddOAuthTablesDown(ctx context.Context, tx *sql.Tx) error {
	tables := []string{"oauth_tokens", "oauth_authorization_codes", "oauth_clients"}
	for _, table := range tables {
		if _, err := tx.ExecContext(ctx, "DROP TABLE IF EXISTS "+table); err != nil {
			return fmt.Errorf("failed to drop %s table: %w", table, err)
		}
	}
	return nil
}
