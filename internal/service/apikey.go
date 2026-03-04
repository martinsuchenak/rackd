package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type APIKeyService struct {
	store storage.ExtendedStorage
}

func NewAPIKeyService(store storage.ExtendedStorage) *APIKeyService {
	return &APIKeyService{store: store}
}

func (s *APIKeyService) List(ctx context.Context, filter *model.APIKeyFilter) ([]model.APIKey, error) {
	if err := requirePermission(ctx, s.store, "apikeys", "list"); err != nil {
		return nil, err
	}

	caller := CallerFrom(ctx)
	if filter == nil {
		filter = &model.APIKeyFilter{}
	}

	// Non-admin users can only see their own keys
	if caller != nil && !caller.IsSystem() {
		isAdmin, _ := auth.IsAdmin(ctx, s.store, caller.UserID)
		if !isAdmin {
			filter.UserID = caller.UserID
		}
	}

	return s.store.ListAPIKeys(filter)
}

func (s *APIKeyService) Create(ctx context.Context, key *model.APIKey) (string, error) {
	if err := requirePermission(ctx, s.store, "apikeys", "create"); err != nil {
		return "", err
	}

	if key.Name == "" {
		return "", ValidationErrors{{Field: "name", Message: "Name is required"}}
	}

	// Assign the key to the creating user
	caller := CallerFrom(ctx)
	if caller != nil && caller.UserID != "" {
		key.UserID = caller.UserID
	}

	key.ID = uuid.Must(uuid.NewV7()).String()
	plaintextKey := uuid.Must(uuid.NewV7()).String()
	key.Key = auth.HashToken(plaintextKey)
	key.CreatedAt = time.Now()

	if err := s.store.CreateAPIKey(key); err != nil {
		return "", err
	}

	return plaintextKey, nil
}

func (s *APIKeyService) Get(ctx context.Context, id string) (*model.APIKey, error) {
	if err := requirePermission(ctx, s.store, "apikeys", "read"); err != nil {
		return nil, err
	}

	key, err := s.store.GetAPIKey(id)
	if err != nil {
		return nil, err
	}

	// Non-admin users can only see their own keys
	if err := s.requireOwnership(ctx, key); err != nil {
		return nil, err
	}

	return key, nil
}

func (s *APIKeyService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "apikeys", "delete"); err != nil {
		return err
	}

	key, err := s.store.GetAPIKey(id)
	if err != nil {
		return err
	}

	// Non-admin users can only delete their own keys
	if err := s.requireOwnership(ctx, key); err != nil {
		return err
	}

	return s.store.DeleteAPIKey(id)
}

// requireOwnership verifies the caller owns the key or is an admin.
func (s *APIKeyService) requireOwnership(ctx context.Context, key *model.APIKey) error {
	caller := CallerFrom(ctx)
	if caller == nil {
		return ErrUnauthenticated
	}
	if caller.IsSystem() {
		return nil
	}

	// Owner can always access their own key
	if key.UserID != "" && key.UserID == caller.UserID {
		return nil
	}

	// Admin can access any key
	isAdmin, _ := auth.IsAdmin(ctx, s.store, caller.UserID)
	if isAdmin {
		return nil
	}

	return ErrForbidden
}
