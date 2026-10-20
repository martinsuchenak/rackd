package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
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

	return s.store.ListAPIKeys(ctx, filter)
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
	scopes, err := s.checkScopes(ctx, caller, key.Scopes)
	if err != nil {
		return "", err
	}
	key.Scopes = scopes

	key.ID = uuid.Must(uuid.NewV7()).String()
	// API keys are long-lived bearer credentials: use 32 bytes of crypto/rand
	// (256 bits), not a UUIDv7 (74 random bits + timestamp structure).
	plaintextKey, err := auth.GenerateKey()
	if err != nil {
		return "", err
	}
	key.Key = auth.HashToken(plaintextKey)
	key.CreatedAt = time.Now().UTC()

	if err := s.store.CreateAPIKey(ctx, key); err != nil {
		return "", err
	}
	recordAudit(ctx, s.store, "create", "apikey", key.ID, map[string]any{"name": key.Name, "user_id": key.UserID, "expires_at": key.ExpiresAt, "scopes": key.Scopes})

	return plaintextKey, nil
}

func (s *APIKeyService) Get(ctx context.Context, id string) (*model.APIKey, error) {
	if err := requirePermission(ctx, s.store, "apikeys", "read"); err != nil {
		return nil, err
	}

	key, err := s.store.GetAPIKey(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrAPIKeyNotFound) {
			return nil, ErrNotFound
		}
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

	key, err := s.store.GetAPIKey(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrAPIKeyNotFound) {
			return ErrNotFound
		}
		return err
	}

	// Non-admin users can only delete their own keys
	if err := s.requireOwnership(ctx, key); err != nil {
		return err
	}

	if err := s.store.DeleteAPIKey(ctx, id); err != nil {
		if errors.Is(err, storage.ErrAPIKeyNotFound) {
			return ErrNotFound
		}
		return err
	}
	recordAudit(ctx, s.store, "delete", "apikey", id, map[string]string{"name": key.Name, "user_id": key.UserID})

	return nil
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

// checkScopes validates a new key's scopes, each a resource:action permission
// the creator holds (a key can't do more than its owner), and returns them
// without duplicates, in order. None means the key gets all of its owner's
// permissions.
func (s *APIKeyService) checkScopes(ctx context.Context, caller *Caller, scopes []string) ([]string, error) {
	if len(scopes) == 0 {
		return nil, nil
	}
	if caller == nil || caller.UserID == "" {
		return nil, ErrUnauthenticated
	}
	// A key made with a scoped credential can't escape that credential's
	// scopes either.
	allowed := map[string]bool{}
	perms, err := s.store.GetUserPermissions(ctx, caller.UserID)
	if err != nil {
		return nil, err
	}
	for _, p := range perms {
		scope := p.Resource + ":" + p.Action
		if caller.Scopes == nil || slices.Contains(caller.Scopes, scope) || slices.Contains(caller.Scopes, "*") {
			allowed[scope] = true
		}
	}
	var out, refused []string
	seen := map[string]bool{}
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" || seen[scope] {
			continue
		}
		seen[scope] = true
		if !allowed[scope] {
			refused = append(refused, scope)
			continue
		}
		out = append(out, scope)
	}
	if len(refused) > 0 {
		return nil, ValidationErrors{{Field: "scopes", Message: "not permissions you hold: " + strings.Join(refused, ", ")}}
	}
	if len(out) == 0 {
		return nil, ValidationErrors{{Field: "scopes", Message: "give at least one resource:action scope, or none for a key with all your permissions"}}
	}
	slices.Sort(out)
	return out, nil
}
