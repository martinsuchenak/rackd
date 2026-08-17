package service

import (
	"context"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/model"
)

// TestOAuthScopeClampToUserPermissions covers the pentest finding: a
// zero-permission user completing consent for a client requesting privileged
// scopes must end up with a token whose scope is empty of anything they don't
// hold — never an echo of the requested strings.
func TestOAuthScopeClampToUserPermissions(t *testing.T) {
	store := newMockOAuthStorage()
	ctx := context.Background()

	store.CreateUser(ctx, &model.User{ID: "user1", Username: "test", PasswordHash: "hash", IsActive: true})
	// user1 holds NO permissions (zero-privilege account).

	// Client registered with a scope inside the advertised catalog.
	store.CreateOAuthClient(ctx, &model.OAuthClient{
		ID:           "client1",
		Name:         "Test Client",
		RedirectURIs: []string{"http://localhost/cb"},
		Scope:        "devices:read users:delete",
	})

	svc := NewOAuthService(store, nil, "http://localhost")

	// 1. Consent must refuse to issue a code when the intersection with the
	// user's (empty) permission set is empty.
	_, err := svc.CreateAuthorizationCode(ctx, "client1", "user1", "http://localhost/cb",
		"devices:read users:delete", "verifier", "S256")
	if err != ErrOAuthScopeRequired {
		t.Errorf("zero-permission user consent: expected ErrOAuthScopeRequired, got %v", err)
	}

	// 2. A user holding a subset must only delegate that subset.
	store.userPerms["user2"] = []string{"devices:read"}
	store.CreateUser(ctx, &model.User{ID: "user2", Username: "limited", PasswordHash: "hash", IsActive: true})
	code, err := svc.CreateAuthorizationCode(ctx, "client1", "user2", "http://localhost/cb",
		"devices:read users:delete", "verifier", "S256")
	if err != nil {
		t.Fatalf("limited user consent failed: %v", err)
	}
	storedCode, err := store.GetAuthorizationCode(ctx, auth.HashToken(code))
	if err != nil {
		t.Fatalf("code lookup failed: %v", err)
	}
	if storedCode.Scope != "devices:read" {
		t.Errorf("expected scope clamped to 'devices:read', got %q", storedCode.Scope)
	}
}

// TestOAuthRegistrationRejectsUnknownScopes covers the DCR half: registering
// with scopes outside scopes_supported (or empty/wildcard) must be rejected.
func TestOAuthRegistrationRejectsUnknownScopes(t *testing.T) {
	store := newMockOAuthStorage()
	ctx := context.Background()
	svc := NewOAuthService(store, nil, "http://localhost")

	cases := []struct {
		name  string
		scope string
	}{
		{"unknown scope", "credentials:read"},               // not in catalog
		{"partially unknown", "devices:read madeup:action"}, // mixed
		{"wildcard", "*"},
		{"empty", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.RegisterClient(ctx, &model.OAuthClientRegistrationRequest{
				ClientName:   "cl-" + tc.name,
				RedirectURIs: []string{"https://client.example/cb"},
				Scope:        tc.scope,
			})
			if err == nil {
				t.Fatal("expected rejection")
			}
			if err != ErrOAuthScopeNotAllowed && err != ErrOAuthScopeRequired {
				t.Errorf("expected scope policy error, got %v", err)
			}
		})
	}

	// A fully valid registration succeeds.
	resp, err := svc.RegisterClient(ctx, &model.OAuthClientRegistrationRequest{
		ClientName:   "good-client",
		RedirectURIs: []string{"https://client.example/cb"},
		Scope:        "devices:read devices:list",
	})
	if err != nil {
		t.Fatalf("valid registration rejected: %v", err)
	}
	if resp.ClientID == "" {
		t.Fatal("expected client ID in response")
	}
}

// TestOAuthRefreshReclampsScope verifies that a refresh token minted before a
// permission revocation loses the revoked scopes at the next refresh.
func TestOAuthRefreshReclampsScope(t *testing.T) {
	store := newMockOAuthStorage()
	ctx := context.Background()

	store.CreateUser(ctx, &model.User{ID: "user1", Username: "test", PasswordHash: "hash", IsActive: true})
	store.userPerms["user1"] = []string{"devices:read", "users:delete"}
	store.CreateOAuthClient(ctx, &model.OAuthClient{
		ID:           "client1",
		Name:         "Test Client",
		RedirectURIs: []string{"http://localhost/cb"},
	})

	svc := NewOAuthService(store, nil, "http://localhost")

	// Token issued while the user held both scopes.
	refreshPlain, refreshHash, err := auth.GenerateOAuthToken()
	if err != nil {
		t.Fatalf("GenerateOAuthToken: %v", err)
	}
	if err := store.CreateOAuthToken(ctx, &model.OAuthToken{
		TokenType: "refresh",
		TokenHash: refreshHash,
		ClientID:  "client1",
		UserID:    "user1",
		Scope:     "devices:read users:delete",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}); err != nil {
		t.Fatalf("CreateOAuthToken: %v", err)
	}

	// User loses users:delete (e.g. role change).
	store.userPerms["user1"] = []string{"devices:read"}

	resp, err := svc.RefreshAccessToken(ctx, &model.OAuthTokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshPlain,
		ClientID:     "client1",
	})
	if err != nil {
		t.Fatalf("RefreshAccessToken: %v", err)
	}
	if resp.Scope != "devices:read" {
		t.Errorf("expected refresh to re-clamp scope to 'devices:read', got %q", resp.Scope)
	}
}
