package service

import (
	"context"
	"time"

	"github.com/google/uuid"
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
	return s.store.ListAPIKeys(filter)
}

func (s *APIKeyService) Create(ctx context.Context, key *model.APIKey) (string, error) {
	if err := requirePermission(ctx, s.store, "apikeys", "create"); err != nil {
		return "", err
	}

	if key.Name == "" {
		return "", ValidationErrors{{Field: "name", Message: "Name is required"}}
	}

	key.ID = uuid.Must(uuid.NewV7()).String()
	key.Key = uuid.Must(uuid.NewV7()).String()
	key.CreatedAt = time.Now()

	if err := s.store.CreateAPIKey(key); err != nil {
		return "", err
	}

	return key.Key, nil
}

func (s *APIKeyService) Get(ctx context.Context, id string) (*model.APIKey, error) {
	if err := requirePermission(ctx, s.store, "apikeys", "read"); err != nil {
		return nil, err
	}

	key, err := s.store.GetAPIKey(id)
	if err != nil {
		return nil, err
	}

	return key, nil
}

func (s *APIKeyService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "apikeys", "delete"); err != nil {
		return err
	}

	if err := s.store.DeleteAPIKey(id); err != nil {
		return err
	}

	return nil
}
