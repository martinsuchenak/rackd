package service

import (
	"context"
	"errors"

	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

type CredentialService struct {
	store credentials.Storage
	rbac  auditedPermissionStore
}

// auditedPermissionStore checks permissions and records audit entries, for
// services whose own store does neither.
type auditedPermissionStore interface {
	permissionStore
	storage.AuditStorage
}

func NewCredentialService(store credentials.Storage, rbac auditedPermissionStore) *CredentialService {
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
		return nil, mapCredentialError(err)
	}
	recordAudit(ctx, s.rbac, "create", "credential", cred.ID, credentialAudit(cred))
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
		return nil, mapCredentialError(err)
	}
	recordAudit(ctx, s.rbac, "update", "credential", id, credentialAudit(cred))
	return cred, nil
}

func (s *CredentialService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.rbac, "credentials", "delete"); err != nil {
		return err
	}

	if err := s.store.Delete(id); err != nil {
		return mapCredentialError(err)
	}
	recordAudit(ctx, s.rbac, "delete", "credential", id, nil)
	return nil
}

func mapCredentialError(err error) error {
	switch {
	case errors.Is(err, credentials.ErrCredentialNotFound):
		return ErrNotFound
	case errors.Is(err, credentials.ErrInvalidCredential):
		return ValidationErrors{{Field: "credential", Message: err.Error()}}
	}
	return NormalizeError(err)
}

// credentialAudit describes a credential for the audit log, without secrets.
func credentialAudit(c *model.Credential) map[string]string {
	return map[string]string{"name": c.Name, "type": c.Type, "datacenter_id": c.DatacenterID}
}
