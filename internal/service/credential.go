package service

import (
	"context"
	"errors"

	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/model"
)

type CredentialService struct {
	store credentials.Storage
	rbac  PermissionChecker
}

func NewCredentialService(store credentials.Storage, rbac PermissionChecker) *CredentialService {
	return &CredentialService{store: store, rbac: rbac}
}

func (s *CredentialService) List(ctx context.Context, datacenterID string) ([]model.Credential, error) {
	if err := requirePermission(ctx, s.rbac, "credentials", "list"); err != nil {
		return nil, err
	}

	return s.store.List(datacenterID)
}

func (s *CredentialService) Create(ctx context.Context, input *model.CredentialInput) (*model.Credential, error) {
	if err := requirePermission(ctx, s.rbac, "credentials", "create"); err != nil {
		return nil, err
	}

	cred := input.ToCredential()
	if err := s.store.Create(cred); err != nil {
		if errors.Is(err, credentials.ErrInvalidCredential) {
			return nil, ValidationErrors{{Field: "credential", Message: err.Error()}}
		}
		return nil, err
	}

	return cred, nil
}

func (s *CredentialService) Get(ctx context.Context, id string) (*model.Credential, error) {
	if err := requirePermission(ctx, s.rbac, "credentials", "read"); err != nil {
		return nil, err
	}

	cred, err := s.store.Get(id)
	if errors.Is(err, credentials.ErrCredentialNotFound) {
		return nil, ErrNotFound
	}
	return cred, err
}

func (s *CredentialService) Update(ctx context.Context, id string, input *model.CredentialInput) (*model.Credential, error) {
	if err := requirePermission(ctx, s.rbac, "credentials", "update"); err != nil {
		return nil, err
	}

	cred := input.ToCredential()
	cred.ID = id
	if err := s.store.Update(cred); err != nil {
		if errors.Is(err, credentials.ErrCredentialNotFound) {
			return nil, ErrNotFound
		}
		if errors.Is(err, credentials.ErrInvalidCredential) {
			return nil, ValidationErrors{{Field: "credential", Message: err.Error()}}
		}
		return nil, err
	}

	return cred, nil
}

func (s *CredentialService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.rbac, "credentials", "delete"); err != nil {
		return err
	}

	err := s.store.Delete(id)
	if errors.Is(err, credentials.ErrCredentialNotFound) {
		return ErrNotFound
	}
	return err
}
