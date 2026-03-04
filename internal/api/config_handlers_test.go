package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestNewUIConfigBuilder(t *testing.T) {
	b := NewUIConfigBuilder()
	config := b.Build()

	if config.Edition != "oss" {
		t.Errorf("expected edition 'oss', got %q", config.Edition)
	}
	if len(config.Features) != 0 {
		t.Errorf("expected empty features, got %d", len(config.Features))
	}
	if len(config.NavItems) != 0 {
		t.Errorf("expected empty nav items, got %d", len(config.NavItems))
	}
	if config.UserInfo != nil {
		t.Error("expected nil user info")
	}
}

func TestUIConfigBuilder_SetEdition(t *testing.T) {
	b := NewUIConfigBuilder()
	b.SetEdition("oss")
	config := b.Build()

	if config.Edition != "oss" {
		t.Errorf("expected edition 'oss', got %q", config.Edition)
	}
}

func TestUIConfigBuilder_SetUser(t *testing.T) {
	b := NewUIConfigBuilder()
	user := &UserInfo{
		ID:          "user-123",
		Username:    "admin",
		Email:       "admin@example.com",
		Roles:       []model.Role{},
		Permissions: []model.Permission{},
	}
	b.SetUser(user)
	config := b.Build()

	if config.UserInfo == nil {
		t.Fatal("expected user info to be set")
	}
	if config.UserInfo.ID != "user-123" {
		t.Errorf("expected user ID 'user-123', got %q", config.UserInfo.ID)
	}
	if config.UserInfo.Username != "admin" {
		t.Errorf("expected username 'admin', got %q", config.UserInfo.Username)
	}
	if len(config.UserInfo.Roles) != 0 {
		t.Errorf("expected 0 roles, got %d", len(config.UserInfo.Roles))
	}
	if len(config.UserInfo.Permissions) != 0 {
		t.Errorf("expected 0 permissions, got %d", len(config.UserInfo.Permissions))
	}
}

func TestUIConfigBuilder_Handler(t *testing.T) {
	b := NewUIConfigBuilder()
	b.SetEdition("oss")

	handler := b.Handler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", ct)
	}

	var config UIConfig
	if err := json.NewDecoder(w.Body).Decode(&config); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if config.Edition != "oss" {
		t.Errorf("expected edition 'oss', got %q", config.Edition)
	}
}
