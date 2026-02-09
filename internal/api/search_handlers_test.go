package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/internal/storage"
)

func TestSearch(t *testing.T) {
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	handler := NewHandler(store, nil)
	handler.SetServices(service.NewServices(store, nil, nil))
	ctx := context.Background()

	// Create test data
	dc := &model.Datacenter{Name: "NYC Datacenter", Location: "New York"}
	if err := store.CreateDatacenter(ctx, dc); err != nil {
		t.Fatalf("Failed to create datacenter: %v", err)
	}

	net := &model.Network{Name: "Production Network", Subnet: "10.0.0.0/24"}
	if err := store.CreateNetwork(ctx, net); err != nil {
		t.Fatalf("Failed to create network: %v", err)
	}

	dev := &model.Device{Name: "web-server", Description: "Production server"}
	if err := store.CreateDevice(ctx, dev); err != nil {
		t.Fatalf("Failed to create device: %v", err)
	}

	// Test search — inject system caller so service layer RBAC allows access
	req := httptest.NewRequest("GET", "/api/search?q=Production", nil)
	req = req.WithContext(service.SystemContext(req.Context(), "test"))
	w := httptest.NewRecorder()

	handler.search(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response SearchResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	t.Logf("Search results: %d", len(response.Results))
	for _, r := range response.Results {
		t.Logf("  - Type: %s", r.Type)
	}

	// Should find at least the network and device
	if len(response.Results) < 2 {
		t.Errorf("Expected at least 2 results, got %d", len(response.Results))
	}

	// Check we have different types
	types := make(map[string]bool)
	for _, r := range response.Results {
		types[r.Type] = true
	}

	if !types["device"] {
		t.Error("Expected to find device in results")
	}
	if !types["network"] {
		t.Error("Expected to find network in results")
	}
}
