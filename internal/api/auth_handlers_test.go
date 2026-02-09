//go:build !short

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/internal/storage"
)

func TestLoginReturnsPermissions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Setup in-memory database with migrations
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()

	// Create a test user with operator role
	ctx := t.Context()
	passwordHash, err := auth.HashPassword("testpassword123")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := &model.User{
		Username:     "testuser",
		Email:        "test@example.com",
		FullName:     "Test User",
		PasswordHash: passwordHash,
		IsActive:     true,
		IsAdmin:      false,
	}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Assign operator role to user
	operatorRole, err := store.GetRoleByName(ctx, "operator")
	if err != nil {
		t.Fatalf("failed to get operator role: %v", err)
	}
	if err := store.AssignRoleToUser(ctx, user.ID, operatorRole.ID); err != nil {
		t.Fatalf("failed to assign role: %v", err)
	}

	// Setup handler and server
	sessionManager := auth.NewSessionManager(3600)
	services := service.NewServices(store, sessionManager, nil)
	h := NewHandler(store, nil)
	h.SetSessionManager(sessionManager)
	h.SetServices(services)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Test login
	loginBody := `{"username":"testuser","password":"testpassword123"}`
	resp, err := http.Post(server.URL+"/api/auth/login", "application/json", bytes.NewBufferString(loginBody))
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var loginResp struct {
		User struct {
			ID          string            `json:"id"`
			Username    string            `json:"username"`
			Email       string            `json:"email"`
			Permissions []json.RawMessage `json:"permissions"`
			Roles       []json.RawMessage `json:"roles"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}

	// Verify: response has permissions
	if len(loginResp.User.Permissions) == 0 {
		t.Error("login response should include permissions")
	}

	// Verify: response has roles
	if len(loginResp.User.Roles) == 0 {
		t.Error("login response should include roles")
	}

	// Verify: user details
	if loginResp.User.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", loginResp.User.Username)
	}
	if loginResp.User.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", loginResp.User.Email)
	}
}
