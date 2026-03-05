package service

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

func init() {
	// Initialize logger for tests (discard output)
	log.Init("console", "error", io.Discard)
}

// mockOAuthStorage implements the subset of storage.ExtendedStorage needed by OAuth tests.
// It embeds storage.ExtendedStorage to satisfy the interface; unused methods will panic.
type mockOAuthStorage struct {
	storage.ExtendedStorage
	tokens   map[string]*model.OAuthToken   // by ID
	byHash   map[string]*model.OAuthToken   // by token hash
	clients  map[string]*model.OAuthClient  // by ID
	codes    map[string]*model.OAuthAuthorizationCode // by code hash
	users    map[string]*model.User         // by ID
}

func newMockOAuthStorage() *mockOAuthStorage {
	return &mockOAuthStorage{
		tokens:  make(map[string]*model.OAuthToken),
		byHash:  make(map[string]*model.OAuthToken),
		clients: make(map[string]*model.OAuthClient),
		codes:   make(map[string]*model.OAuthAuthorizationCode),
		users:   make(map[string]*model.User),
	}
}

func (m *mockOAuthStorage) CreateOAuthToken(_ context.Context, token *model.OAuthToken) error {
	if token.ID == "" {
		token.ID = "token-" + time.Now().Format("20060102150405.000000000")
	}
	token.CreatedAt = time.Now().UTC()
	m.tokens[token.ID] = token
	m.byHash[token.TokenHash] = token
	return nil
}

func (m *mockOAuthStorage) GetOAuthTokenByHash(_ context.Context, tokenHash string) (*model.OAuthToken, error) {
	token, ok := m.byHash[tokenHash]
	if !ok {
		return nil, storage.ErrOAuthTokenNotFound
	}
	if token.RevokedAt != nil {
		return nil, storage.ErrOAuthTokenRevoked
	}
	if time.Now().After(token.ExpiresAt) {
		return nil, storage.ErrOAuthTokenExpired
	}
	return token, nil
}

func (m *mockOAuthStorage) GetOAuthTokenByHashIncludingRevoked(_ context.Context, tokenHash string) (*model.OAuthToken, error) {
	token, ok := m.byHash[tokenHash]
	if !ok {
		return nil, storage.ErrOAuthTokenNotFound
	}
	return token, nil
}

func (m *mockOAuthStorage) RevokeOAuthToken(_ context.Context, tokenID string) error {
	token, ok := m.tokens[tokenID]
	if !ok {
		return nil
	}
	now := time.Now().UTC()
	token.RevokedAt = &now
	return nil
}

func (m *mockOAuthStorage) RevokeOAuthTokenChain(_ context.Context, refreshTokenID string) error {
	now := time.Now().UTC()
	// Revoke all tokens that have this refresh token as parent
	for _, token := range m.tokens {
		if token.ParentTokenID == refreshTokenID {
			token.RevokedAt = &now
		}
	}
	// Revoke the refresh token itself
	if token, ok := m.tokens[refreshTokenID]; ok {
		token.RevokedAt = &now
	}
	return nil
}

func (m *mockOAuthStorage) GetOAuthClient(_ context.Context, clientID string) (*model.OAuthClient, error) {
	client, ok := m.clients[clientID]
	if !ok {
		return nil, storage.ErrOAuthClientNotFound
	}
	return client, nil
}

func (m *mockOAuthStorage) GetUser(_ context.Context, userID string) (*model.User, error) {
	user, ok := m.users[userID]
	if !ok {
		return nil, storage.ErrUserNotFound
	}
	return user, nil
}

func (m *mockOAuthStorage) CreateOAuthClient(_ context.Context, client *model.OAuthClient) error {
	m.clients[client.ID] = client
	return nil
}

func (m *mockOAuthStorage) CreateUser(_ context.Context, user *model.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockOAuthStorage) GetAuthorizationCode(_ context.Context, codeHash string) (*model.OAuthAuthorizationCode, error) {
	code, ok := m.codes[codeHash]
	if !ok {
		return nil, storage.ErrOAuthCodeNotFound
	}
	if code.Used {
		return nil, storage.ErrOAuthCodeUsed
	}
	if time.Now().After(code.ExpiresAt) {
		return nil, storage.ErrOAuthCodeExpired
	}
	return code, nil
}

func (m *mockOAuthStorage) CreateAuthorizationCode(_ context.Context, code *model.OAuthAuthorizationCode) error {
	m.codes[code.CodeHash] = code
	return nil
}

func (m *mockOAuthStorage) MarkAuthorizationCodeUsed(_ context.Context, codeHash string) error {
	code, ok := m.codes[codeHash]
	if !ok {
		return storage.ErrOAuthCodeNotFound
	}
	if code.Used {
		return storage.ErrOAuthCodeUsed
	}
	code.Used = true
	return nil
}

// TestRefreshTokenRotation tests that refresh token rotation works correctly:
// - A new refresh token is issued when using a refresh token
// - The old refresh token is revoked
// - The new refresh token is returned in the response
func TestRefreshTokenRotation(t *testing.T) {
	store := newMockOAuthStorage()
	ctx := context.Background()

	// Setup test data
	store.CreateUser(ctx, &model.User{ID: "user1", Username: "test", PasswordHash: "hash", IsActive: true})
	store.CreateOAuthClient(ctx, &model.OAuthClient{
		ID:           "client1",
		Name:         "Test Client",
		RedirectURIs: []string{"http://localhost/cb"},
	})

	// Create OAuth service
	oauthSvc := NewOAuthService(store, nil, "http://localhost")

	// Create initial refresh token
	refreshPlain, refreshHash, err := auth.GenerateOAuthToken()
	if err != nil {
		t.Fatalf("GenerateOAuthToken failed: %v", err)
	}

	refreshToken := &model.OAuthToken{
		TokenType: "refresh",
		TokenHash: refreshHash,
		ClientID:  "client1",
		UserID:    "user1",
		Scope:     "devices:read",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.CreateOAuthToken(ctx, refreshToken); err != nil {
		t.Fatalf("CreateOAuthToken failed: %v", err)
	}

	// Use refresh token to get new access token
	resp, err := oauthSvc.RefreshAccessToken(ctx, &model.OAuthTokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshPlain,
		ClientID:     "client1",
	})
	if err != nil {
		t.Fatalf("RefreshAccessToken failed: %v", err)
	}

	// Verify response contains both access and refresh tokens
	if resp.AccessToken == "" {
		t.Error("expected access token in response")
	}
	if resp.RefreshToken == "" {
		t.Error("expected refresh token in response (rotation)")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("expected token type Bearer, got %s", resp.TokenType)
	}

	// Verify old refresh token is revoked
	oldToken, err := store.GetOAuthTokenByHashIncludingRevoked(ctx, refreshHash)
	if err != nil {
		t.Fatalf("GetOAuthTokenByHashIncludingRevoked failed: %v", err)
	}
	if oldToken.RevokedAt == nil {
		t.Error("expected old refresh token to be revoked")
	}

	// Verify old refresh token can no longer be used normally
	_, err = store.GetOAuthTokenByHash(ctx, refreshHash)
	if err != storage.ErrOAuthTokenRevoked {
		t.Errorf("expected ErrOAuthTokenRevoked for old token, got %v", err)
	}

	// Verify new refresh token is valid
	newToken, err := store.GetOAuthTokenByHash(ctx, auth.HashToken(resp.RefreshToken))
	if err != nil {
		t.Fatalf("GetOAuthTokenByHash for new refresh token failed: %v", err)
	}
	if newToken.RevokedAt != nil {
		t.Error("expected new refresh token to NOT be revoked")
	}
	if newToken.TokenType != "refresh" {
		t.Errorf("expected token type refresh, got %s", newToken.TokenType)
	}
}

// TestRefreshTokenReplayDetection tests that replay attacks are detected:
// - If a revoked refresh token is reused, it's detected as a replay attack
// - The token chain is revoked to prevent further abuse
func TestRefreshTokenReplayDetection(t *testing.T) {
	store := newMockOAuthStorage()
	ctx := context.Background()

	// Setup test data
	store.CreateUser(ctx, &model.User{ID: "user1", Username: "test", PasswordHash: "hash", IsActive: true})
	store.CreateOAuthClient(ctx, &model.OAuthClient{
		ID:           "client1",
		Name:         "Test Client",
		RedirectURIs: []string{"http://localhost/cb"},
	})

	// Create OAuth service
	oauthSvc := NewOAuthService(store, nil, "http://localhost")

	// Create initial refresh token
	refreshPlain, refreshHash, err := auth.GenerateOAuthToken()
	if err != nil {
		t.Fatalf("GenerateOAuthToken failed: %v", err)
	}

	refreshToken := &model.OAuthToken{
		ID:        "refresh-token-1",
		TokenType: "refresh",
		TokenHash: refreshHash,
		ClientID:  "client1",
		UserID:    "user1",
		Scope:     "devices:read",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.CreateOAuthToken(ctx, refreshToken); err != nil {
		t.Fatalf("CreateOAuthToken failed: %v", err)
	}

	// First use - should succeed and rotate token
	resp1, err := oauthSvc.RefreshAccessToken(ctx, &model.OAuthTokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshPlain,
		ClientID:     "client1",
	})
	if err != nil {
		t.Fatalf("First RefreshAccessToken failed: %v", err)
	}
	if resp1.RefreshToken == "" {
		t.Error("expected rotated refresh token in first response")
	}

	// Create an access token that has the old refresh token as parent
	// (simulating what would happen in a real scenario)
	accessToken := &model.OAuthToken{
		ID:            "access-token-1",
		TokenType:     "access",
		TokenHash:     "access-hash-1",
		ClientID:      "client1",
		UserID:        "user1",
		Scope:         "devices:read",
		ExpiresAt:     time.Now().Add(1 * time.Hour),
		ParentTokenID: refreshToken.ID,
	}
	store.CreateOAuthToken(ctx, accessToken)

	// Second use of the same (now revoked) refresh token - should be detected as replay
	_, err = oauthSvc.RefreshAccessToken(ctx, &model.OAuthTokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshPlain,
		ClientID:     "client1",
	})
	if err != storage.ErrOAuthTokenRevoked {
		t.Errorf("expected ErrOAuthTokenRevoked for replay attempt, got %v", err)
	}

	// Verify the access token in the chain was also revoked (replay mitigation)
	revokedAccess, err := store.GetOAuthTokenByHashIncludingRevoked(ctx, "access-hash-1")
	if err != nil {
		t.Fatalf("GetOAuthTokenByHashIncludingRevoked for access token failed: %v", err)
	}
	if revokedAccess.RevokedAt == nil {
		t.Error("expected access token in chain to be revoked after replay detection")
	}
}

// TestRefreshTokenInvalidClient tests that using wrong client ID fails
func TestRefreshTokenInvalidClient(t *testing.T) {
	store := newMockOAuthStorage()
	ctx := context.Background()

	// Setup test data
	store.CreateUser(ctx, &model.User{ID: "user1", Username: "test", PasswordHash: "hash", IsActive: true})
	store.CreateOAuthClient(ctx, &model.OAuthClient{
		ID:           "client1",
		Name:         "Test Client",
		RedirectURIs: []string{"http://localhost/cb"},
	})

	// Create OAuth service
	oauthSvc := NewOAuthService(store, nil, "http://localhost")

	// Create refresh token
	refreshPlain, refreshHash, err := auth.GenerateOAuthToken()
	if err != nil {
		t.Fatalf("GenerateOAuthToken failed: %v", err)
	}

	refreshToken := &model.OAuthToken{
		TokenType: "refresh",
		TokenHash: refreshHash,
		ClientID:  "client1",
		UserID:    "user1",
		Scope:     "devices:read",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.CreateOAuthToken(ctx, refreshToken); err != nil {
		t.Fatalf("CreateOAuthToken failed: %v", err)
	}

	// Try to use with wrong client ID
	_, err = oauthSvc.RefreshAccessToken(ctx, &model.OAuthTokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshPlain,
		ClientID:     "wrong-client",
	})
	if err != ErrOAuthInvalidClient {
		t.Errorf("expected ErrOAuthInvalidClient, got %v", err)
	}
}
