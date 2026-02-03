package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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

func TestUIConfigBuilder_AddFeature(t *testing.T) {
	b := NewUIConfigBuilder()
	b.AddFeature("advanced_scanning")
	b.AddFeature("sso")
	config := b.Build()

	if len(config.Features) != 2 {
		t.Fatalf("expected 2 features, got %d", len(config.Features))
	}
	if config.Features[0] != "advanced_scanning" {
		t.Errorf("expected first feature 'advanced_scanning', got %q", config.Features[0])
	}
	if config.Features[1] != "sso" {
		t.Errorf("expected second feature 'sso', got %q", config.Features[1])
	}
}

func TestUIConfigBuilder_AddNavItem(t *testing.T) {
	b := NewUIConfigBuilder()
	b.AddNavItem(NavItem{Label: "Credentials", Path: "/credentials", Icon: "key", Order: 10})
	b.AddNavItem(NavItem{Label: "Profiles", Path: "/profiles", Icon: "settings", Order: 20})
	config := b.Build()

	if len(config.NavItems) != 2 {
		t.Fatalf("expected 2 nav items, got %d", len(config.NavItems))
	}
	if config.NavItems[0].Label != "Credentials" {
		t.Errorf("expected first nav item label 'Credentials', got %q", config.NavItems[0].Label)
	}
	if config.NavItems[1].Order != 20 {
		t.Errorf("expected second nav item order 20, got %d", config.NavItems[1].Order)
	}
}

func TestUIConfigBuilder_SetUser(t *testing.T) {
	b := NewUIConfigBuilder()
	user := &UserInfo{
		ID:       "user-123",
		Username: "admin",
		Email:    "admin@example.com",
		Roles:    []string{"admin", "viewer"},
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
	if len(config.UserInfo.Roles) != 2 {
		t.Errorf("expected 2 roles, got %d", len(config.UserInfo.Roles))
	}
}

func TestUIConfigBuilder_Handler(t *testing.T) {
	b := NewUIConfigBuilder()
	b.SetEdition("oss")
	b.AddFeature("advanced_scanning")
	b.AddNavItem(NavItem{Label: "Profiles", Path: "/profiles", Icon: "settings", Order: 100})

	handler := b.Handler()
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
	if len(config.Features) != 1 || config.Features[0] != "advanced_scanning" {
		t.Errorf("unexpected features: %v", config.Features)
	}
	if len(config.NavItems) != 1 || config.NavItems[0].Label != "SSO" {
		t.Errorf("unexpected nav items: %v", config.NavItems)
	}
}

func TestGetConfig(t *testing.T) {
	h := NewHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()

	h.getConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var config UIConfig
	if err := json.NewDecoder(w.Body).Decode(&config); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if config.Edition != "oss" {
		t.Errorf("expected edition 'oss', got %q", config.Edition)
	}
	if len(config.Features) != 0 {
		t.Errorf("expected empty features for OSS, got %v", config.Features)
	}
}
