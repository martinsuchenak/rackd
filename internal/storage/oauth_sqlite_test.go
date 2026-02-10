package storage

import (
	"context"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestOAuthClientCRUD(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	client := &model.OAuthClient{
		Name:              "Test Client",
		RedirectURIs:      []string{"http://localhost:8080/callback"},
		GrantTypes:        []string{"authorization_code", "refresh_token"},
		ResponseTypes:     []string{"code"},
		TokenEndpointAuth: "none",
		Scope:             "devices:read devices:write",
		ClientURI:         "http://example.com",
	}

	// Create
	if err := s.CreateOAuthClient(ctx, client); err != nil {
		t.Fatalf("CreateOAuthClient failed: %v", err)
	}
	if client.ID == "" {
		t.Fatal("client ID not set")
	}

	// Get
	got, err := s.GetOAuthClient(client.ID)
	if err != nil {
		t.Fatalf("GetOAuthClient failed: %v", err)
	}
	if got.Name != "Test Client" {
		t.Fatalf("name mismatch: got %q", got.Name)
	}
	if len(got.RedirectURIs) != 1 || got.RedirectURIs[0] != "http://localhost:8080/callback" {
		t.Fatalf("redirect URIs mismatch: got %v", got.RedirectURIs)
	}
	if got.Scope != "devices:read devices:write" {
		t.Fatalf("scope mismatch: got %q", got.Scope)
	}

	// List
	clients, err := s.ListOAuthClients("")
	if err != nil {
		t.Fatalf("ListOAuthClients failed: %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(clients))
	}

	// Get not found
	_, err = s.GetOAuthClient("nonexistent")
	if err != ErrOAuthClientNotFound {
		t.Fatalf("expected ErrOAuthClientNotFound, got %v", err)
	}

	// Delete
	if err := s.DeleteOAuthClient(ctx, client.ID); err != nil {
		t.Fatalf("DeleteOAuthClient failed: %v", err)
	}
	_, err = s.GetOAuthClient(client.ID)
	if err != ErrOAuthClientNotFound {
		t.Fatalf("expected ErrOAuthClientNotFound after delete, got %v", err)
	}
}

func createTestUserAndClient(t *testing.T, s *SQLiteStorage) {
	t.Helper()
	ctx := context.Background()
	s.CreateUser(ctx, &model.User{ID: "user1", Username: "test", PasswordHash: "hash", IsActive: true})
	s.CreateOAuthClient(ctx, &model.OAuthClient{
		ID:                "client1",
		Name:              "Test",
		RedirectURIs:      []string{"http://localhost/cb"},
		GrantTypes:        []string{"authorization_code"},
		ResponseTypes:     []string{"code"},
		TokenEndpointAuth: "none",
	})
}

func TestOAuthAuthorizationCode(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()
	createTestUserAndClient(t, s)

	code := &model.OAuthAuthorizationCode{
		CodeHash:            "test-code-hash",
		ClientID:            "client1",
		UserID:              "user1",
		RedirectURI:         "http://localhost/cb",
		Scope:               "devices:read",
		CodeChallenge:       "test-challenge",
		CodeChallengeMethod: "S256",
		ExpiresAt:           time.Now().Add(10 * time.Minute),
		CreatedAt:           time.Now().UTC(),
	}

	// Create
	if err := s.CreateAuthorizationCode(ctx, code); err != nil {
		t.Fatalf("CreateAuthorizationCode failed: %v", err)
	}

	// Get
	got, err := s.GetAuthorizationCode("test-code-hash")
	if err != nil {
		t.Fatalf("GetAuthorizationCode failed: %v", err)
	}
	if got.ClientID != "client1" {
		t.Fatalf("client_id mismatch: got %q", got.ClientID)
	}

	// Mark used
	if err := s.MarkAuthorizationCodeUsed("test-code-hash"); err != nil {
		t.Fatalf("MarkAuthorizationCodeUsed failed: %v", err)
	}

	// Get used code should fail
	_, err = s.GetAuthorizationCode("test-code-hash")
	if err != ErrOAuthCodeUsed {
		t.Fatalf("expected ErrOAuthCodeUsed, got %v", err)
	}

	// Get not found
	_, err = s.GetAuthorizationCode("nonexistent")
	if err != ErrOAuthCodeNotFound {
		t.Fatalf("expected ErrOAuthCodeNotFound, got %v", err)
	}
}

func TestOAuthToken(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()
	createTestUserAndClient(t, s)

	token := &model.OAuthToken{
		TokenType: "access",
		TokenHash: "test-token-hash",
		ClientID:  "client1",
		UserID:    "user1",
		Scope:     "devices:read",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	// Create
	if err := s.CreateOAuthToken(ctx, token); err != nil {
		t.Fatalf("CreateOAuthToken failed: %v", err)
	}
	if token.ID == "" {
		t.Fatal("token ID not set")
	}

	// Get by hash
	got, err := s.GetOAuthTokenByHash("test-token-hash")
	if err != nil {
		t.Fatalf("GetOAuthTokenByHash failed: %v", err)
	}
	if got.TokenType != "access" {
		t.Fatalf("token_type mismatch: got %q", got.TokenType)
	}
	if got.UserID != "user1" {
		t.Fatalf("user_id mismatch: got %q", got.UserID)
	}

	// Revoke
	if err := s.RevokeOAuthToken(token.ID); err != nil {
		t.Fatalf("RevokeOAuthToken failed: %v", err)
	}

	// Get revoked token should fail
	_, err = s.GetOAuthTokenByHash("test-token-hash")
	if err != ErrOAuthTokenRevoked {
		t.Fatalf("expected ErrOAuthTokenRevoked, got %v", err)
	}

	// Get not found
	_, err = s.GetOAuthTokenByHash("nonexistent")
	if err != ErrOAuthTokenNotFound {
		t.Fatalf("expected ErrOAuthTokenNotFound, got %v", err)
	}
}

func TestOAuthTokenExpiry(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()
	createTestUserAndClient(t, s)

	// Create expired token
	token := &model.OAuthToken{
		TokenType: "access",
		TokenHash: "expired-token-hash",
		ClientID:  "client1",
		UserID:    "user1",
		Scope:     "devices:read",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // already expired
	}
	if err := s.CreateOAuthToken(ctx, token); err != nil {
		t.Fatalf("CreateOAuthToken failed: %v", err)
	}

	_, err := s.GetOAuthTokenByHash("expired-token-hash")
	if err != ErrOAuthTokenExpired {
		t.Fatalf("expected ErrOAuthTokenExpired, got %v", err)
	}
}

func TestRevokeOAuthTokensByClient(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()
	createTestUserAndClient(t, s)

	// Create two tokens
	for _, hash := range []string{"token-1", "token-2"} {
		s.CreateOAuthToken(ctx, &model.OAuthToken{
			TokenType: "access",
			TokenHash: hash,
			ClientID:  "client1",
			UserID:    "user1",
			Scope:     "*",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		})
	}

	// Revoke all by client
	if err := s.RevokeOAuthTokensByClient("client1"); err != nil {
		t.Fatalf("RevokeOAuthTokensByClient failed: %v", err)
	}

	// Both should be revoked
	for _, hash := range []string{"token-1", "token-2"} {
		_, err := s.GetOAuthTokenByHash(hash)
		if err != ErrOAuthTokenRevoked {
			t.Fatalf("expected ErrOAuthTokenRevoked for %s, got %v", hash, err)
		}
	}
}
