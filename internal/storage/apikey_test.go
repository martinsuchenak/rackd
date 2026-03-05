package storage

import (
	"context"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestAPIKeyOperations(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	ctx := context.Background()

	// Create API key
	key := &model.APIKey{
		Name:        "test-key",
		Key:         "test-token-123",
		Description: "Test API key",
	}

	if err := store.CreateAPIKey(ctx, key); err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	if key.ID == "" {
		t.Error("Expected ID to be set")
	}

	// Get by ID
	retrieved, err := store.GetAPIKey(ctx, key.ID)
	if err != nil {
		t.Fatalf("GetAPIKey failed: %v", err)
	}

	if retrieved.Name != key.Name {
		t.Errorf("Expected name %s, got %s", key.Name, retrieved.Name)
	}

	// Get by key string
	byKey, err := store.GetAPIKeyByKey(ctx, key.Key)
	if err != nil {
		t.Fatalf("GetAPIKeyByKey failed: %v", err)
	}

	if byKey.ID != key.ID {
		t.Errorf("Expected ID %s, got %s", key.ID, byKey.ID)
	}

	// List
	keys, err := store.ListAPIKeys(ctx, nil)
	if err != nil {
		t.Fatalf("ListAPIKeys failed: %v", err)
	}

	if len(keys) != 1 {
		t.Errorf("Expected 1 key, got %d", len(keys))
	}

	// Update last used
	now := time.Now()
	if err := store.UpdateAPIKeyLastUsed(ctx, key.ID, now); err != nil {
		t.Fatalf("UpdateAPIKeyLastUsed failed: %v", err)
	}

	updated, err := store.GetAPIKey(ctx, key.ID)
	if err != nil {
		t.Fatalf("GetAPIKey failed: %v", err)
	}

	if updated.LastUsedAt == nil {
		t.Error("Expected LastUsedAt to be set")
	}

	// Delete
	if err := store.DeleteAPIKey(ctx, key.ID); err != nil {
		t.Fatalf("DeleteAPIKey failed: %v", err)
	}

	// Verify deleted
	_, err = store.GetAPIKey(ctx, key.ID)
	if err == nil {
		t.Error("Expected error when getting deleted key")
	}
}

func TestAPIKeyExpiration(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	expired := time.Now().Add(-1 * time.Hour)
	key := &model.APIKey{
		Name:      "expired-key",
		Key:       "expired-token",
		ExpiresAt: &expired,
	}

	ctx := context.Background()
	if err := store.CreateAPIKey(ctx, key); err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	retrieved, err := store.GetAPIKey(ctx, key.ID)
	if err != nil {
		t.Fatalf("GetAPIKey failed: %v", err)
	}

	if retrieved.ExpiresAt == nil {
		t.Error("Expected ExpiresAt to be set")
	}

	if !retrieved.ExpiresAt.Before(time.Now()) {
		t.Error("Expected key to be expired")
	}
}
