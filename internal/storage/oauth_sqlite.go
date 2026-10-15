package storage

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/pkg/model"
)

// --- OAuth Clients ---

func (s *SQLiteStorage) CreateOAuthClient(ctx context.Context, client *model.OAuthClient) error {
	if client.ID == "" {
		client.ID = uuid.New().String()
	}
	now := nowUTC()
	client.CreatedAt = now
	client.UpdatedAt = now

	redirectURIs, _ := json.Marshal(client.RedirectURIs)
	grantTypes, _ := json.Marshal(client.GrantTypes)
	responseTypes, _ := json.Marshal(client.ResponseTypes)

	// Pass NULL for empty created_by_user_id to satisfy FK constraint
	var createdByUserID any
	if client.CreatedByUserID != "" {
		createdByUserID = client.CreatedByUserID
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO oauth_clients (id, name, secret_hash, redirect_uris, grant_types, response_types,
			token_endpoint_auth, scope, client_uri, logo_uri, is_confidential, created_by_user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, client.ID, client.Name, client.SecretHash, string(redirectURIs), string(grantTypes), string(responseTypes),
		client.TokenEndpointAuth, client.Scope, client.ClientURI, client.LogoURI,
		client.IsConfidential, createdByUserID, client.CreatedAt, client.UpdatedAt)
	return err
}

func (s *SQLiteStorage) GetOAuthClient(ctx context.Context, clientID string) (*model.OAuthClient, error) {
	var client model.OAuthClient
	var redirectURIs, grantTypes, responseTypes string
	var createdByUserID sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, secret_hash, redirect_uris, grant_types, response_types,
			token_endpoint_auth, scope, client_uri, logo_uri, is_confidential,
			created_by_user_id, created_at, updated_at
		FROM oauth_clients WHERE id = ?
	`, clientID).Scan(
		&client.ID, &client.Name, &client.SecretHash, &redirectURIs, &grantTypes, &responseTypes,
		&client.TokenEndpointAuth, &client.Scope, &client.ClientURI, &client.LogoURI,
		&client.IsConfidential, &createdByUserID, &client.CreatedAt, &client.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrOAuthClientNotFound
	}
	if err != nil {
		return nil, err
	}

	client.CreatedByUserID = createdByUserID.String
	_ = json.Unmarshal([]byte(redirectURIs), &client.RedirectURIs)
	_ = json.Unmarshal([]byte(grantTypes), &client.GrantTypes)
	_ = json.Unmarshal([]byte(responseTypes), &client.ResponseTypes)

	return &client, nil
}

func (s *SQLiteStorage) ListOAuthClients(ctx context.Context, createdByUserID string) ([]model.OAuthClient, error) {
	var rows *sql.Rows
	var err error

	if createdByUserID == "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, name, secret_hash, redirect_uris, grant_types, response_types,
				token_endpoint_auth, scope, client_uri, logo_uri, is_confidential,
				created_by_user_id, created_at, updated_at
			FROM oauth_clients ORDER BY created_at DESC
		`)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, name, secret_hash, redirect_uris, grant_types, response_types,
				token_endpoint_auth, scope, client_uri, logo_uri, is_confidential,
				created_by_user_id, created_at, updated_at
			FROM oauth_clients WHERE created_by_user_id = ? ORDER BY created_at DESC
		`, createdByUserID)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var clients []model.OAuthClient
	for rows.Next() {
		var client model.OAuthClient
		var redirectURIs, grantTypes, responseTypes string
		var createdByUserID sql.NullString
		if err := rows.Scan(
			&client.ID, &client.Name, &client.SecretHash, &redirectURIs, &grantTypes, &responseTypes,
			&client.TokenEndpointAuth, &client.Scope, &client.ClientURI, &client.LogoURI,
			&client.IsConfidential, &createdByUserID, &client.CreatedAt, &client.UpdatedAt,
		); err != nil {
			return nil, err
		}
		client.CreatedByUserID = createdByUserID.String
		_ = json.Unmarshal([]byte(redirectURIs), &client.RedirectURIs)
		_ = json.Unmarshal([]byte(grantTypes), &client.GrantTypes)
		_ = json.Unmarshal([]byte(responseTypes), &client.ResponseTypes)
		clients = append(clients, client)
	}
	return clients, rows.Err()
}

func (s *SQLiteStorage) DeleteOAuthClient(ctx context.Context, clientID string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM oauth_clients WHERE id = ?`, clientID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrOAuthClientNotFound
	}
	return nil
}

// --- Authorization Codes ---

func (s *SQLiteStorage) CreateAuthorizationCode(ctx context.Context, code *model.OAuthAuthorizationCode) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO oauth_authorization_codes (code_hash, client_id, user_id, redirect_uri, scope,
			code_challenge, code_challenge_method, expires_at, created_at, used)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0)
	`, code.CodeHash, code.ClientID, code.UserID, code.RedirectURI, code.Scope,
		code.CodeChallenge, code.CodeChallengeMethod, code.ExpiresAt, code.CreatedAt)
	return err
}

func (s *SQLiteStorage) GetAuthorizationCode(ctx context.Context, codeHash string) (*model.OAuthAuthorizationCode, error) {
	var code model.OAuthAuthorizationCode
	err := s.db.QueryRowContext(ctx, `
		SELECT code_hash, client_id, user_id, redirect_uri, scope,
			code_challenge, code_challenge_method, expires_at, created_at, used
		FROM oauth_authorization_codes WHERE code_hash = ?
	`, codeHash).Scan(
		&code.CodeHash, &code.ClientID, &code.UserID, &code.RedirectURI, &code.Scope,
		&code.CodeChallenge, &code.CodeChallengeMethod, &code.ExpiresAt, &code.CreatedAt, &code.Used,
	)
	if err == sql.ErrNoRows {
		return nil, ErrOAuthCodeNotFound
	}
	if err != nil {
		return nil, err
	}
	if code.Used {
		return nil, ErrOAuthCodeUsed
	}
	if nowUTC().After(code.ExpiresAt) {
		return nil, ErrOAuthCodeExpired
	}
	return &code, nil
}

func (s *SQLiteStorage) MarkAuthorizationCodeUsed(ctx context.Context, codeHash string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE oauth_authorization_codes SET used = 1 WHERE code_hash = ? AND used = 0`, codeHash)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrOAuthCodeUsed
	}

	return nil
}

func (s *SQLiteStorage) CleanupExpiredCodes(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM oauth_authorization_codes WHERE expires_at < ? OR used = 1`, nowUTC())
	return err
}

// --- OAuth Tokens ---

func (s *SQLiteStorage) CreateOAuthToken(ctx context.Context, token *model.OAuthToken) error {
	if token.ID == "" {
		token.ID = uuid.New().String()
	}
	token.CreatedAt = nowUTC()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO oauth_tokens (id, token_type, token_hash, client_id, user_id, scope,
			expires_at, created_at, revoked_at, parent_token_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, token.ID, token.TokenType, token.TokenHash, token.ClientID, token.UserID,
		token.Scope, token.ExpiresAt, token.CreatedAt, token.RevokedAt, token.ParentTokenID)
	return err
}

func (s *SQLiteStorage) GetOAuthTokenByHash(ctx context.Context, tokenHash string) (*model.OAuthToken, error) {
	var token model.OAuthToken
	err := s.db.QueryRowContext(ctx, `
		SELECT id, token_type, token_hash, client_id, user_id, scope,
			expires_at, created_at, revoked_at, parent_token_id
		FROM oauth_tokens WHERE token_hash = ?
	`, tokenHash).Scan(
		&token.ID, &token.TokenType, &token.TokenHash, &token.ClientID, &token.UserID,
		&token.Scope, &token.ExpiresAt, &token.CreatedAt, &token.RevokedAt, &token.ParentTokenID,
	)
	if err == sql.ErrNoRows {
		return nil, ErrOAuthTokenNotFound
	}
	if err != nil {
		return nil, err
	}
	if token.RevokedAt != nil {
		return nil, ErrOAuthTokenRevoked
	}
	if nowUTC().After(token.ExpiresAt) {
		return nil, ErrOAuthTokenExpired
	}
	return &token, nil
}

// GetOAuthTokenByHashIncludingRevoked gets a token by hash even if it's revoked.
// This is used for refresh token replay detection.
func (s *SQLiteStorage) GetOAuthTokenByHashIncludingRevoked(ctx context.Context, tokenHash string) (*model.OAuthToken, error) {
	var token model.OAuthToken
	err := s.db.QueryRowContext(ctx, `
		SELECT id, token_type, token_hash, client_id, user_id, scope,
			expires_at, created_at, revoked_at, parent_token_id
		FROM oauth_tokens WHERE token_hash = ?
	`, tokenHash).Scan(
		&token.ID, &token.TokenType, &token.TokenHash, &token.ClientID, &token.UserID,
		&token.Scope, &token.ExpiresAt, &token.CreatedAt, &token.RevokedAt, &token.ParentTokenID,
	)
	if err == sql.ErrNoRows {
		return nil, ErrOAuthTokenNotFound
	}
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (s *SQLiteStorage) RevokeOAuthToken(ctx context.Context, tokenID string) error {
	now := nowUTC()
	_, err := s.db.ExecContext(ctx, `UPDATE oauth_tokens SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`, now, tokenID)
	return err
}

// ClaimOAuthToken revokes tokenID if it isn't revoked yet and reports whether
// this call did. Refresh uses it so a token can be exchanged only once, even
// by two requests at the same moment.
func (s *SQLiteStorage) ClaimOAuthToken(ctx context.Context, tokenID string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE oauth_tokens SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`, nowUTC(), tokenID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// RevokeOAuthTokenChain revokes a refresh token and every token descended
// from it. It is called when refresh token replay is detected: each rotation
// issues an access token whose child is the next refresh token, so a thief who
// refreshed first holds a grandchild, which revoking direct children missed.
func (s *SQLiteStorage) RevokeOAuthTokenChain(ctx context.Context, refreshTokenID string) error {
	_, err := s.db.ExecContext(ctx, `
		WITH RECURSIVE chain(id) AS (
			SELECT ?
			UNION
			SELECT t.id FROM oauth_tokens t JOIN chain c ON t.parent_token_id = c.id
		)
		UPDATE oauth_tokens SET revoked_at = ?
		WHERE id IN (SELECT id FROM chain) AND revoked_at IS NULL
	`, refreshTokenID, nowUTC())
	return err
}

func (s *SQLiteStorage) RevokeOAuthTokensByClient(ctx context.Context, clientID string) error {
	now := nowUTC()
	_, err := s.db.ExecContext(ctx, `UPDATE oauth_tokens SET revoked_at = ? WHERE client_id = ? AND revoked_at IS NULL`, now, clientID)
	return err
}

func (s *SQLiteStorage) RevokeOAuthTokensByUser(ctx context.Context, userID string) error {
	now := nowUTC()
	_, err := s.db.ExecContext(ctx, `UPDATE oauth_tokens SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`, now, userID)
	return err
}

func (s *SQLiteStorage) CleanupExpiredTokens(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM oauth_tokens WHERE expires_at < ?`, nowUTC())
	return err
}
