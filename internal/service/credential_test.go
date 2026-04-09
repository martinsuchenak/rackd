package service

import (
	"errors"
	"testing"

	credstore "github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/model"
)

type credentialStoreStub struct {
	items     map[string]*model.Credential
	createErr error
	updateErr error
}

func (s *credentialStoreStub) Create(cred *model.Credential) error {
	if s.createErr != nil {
		return s.createErr
	}
	if s.items == nil {
		s.items = make(map[string]*model.Credential)
	}
	cloned := *cred
	s.items[cred.ID] = &cloned
	return nil
}

func (s *credentialStoreStub) Update(cred *model.Credential) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	if _, ok := s.items[cred.ID]; !ok {
		return credstore.ErrCredentialNotFound
	}
	cloned := *cred
	s.items[cred.ID] = &cloned
	return nil
}

func (s *credentialStoreStub) Get(id string) (*model.Credential, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, credstore.ErrCredentialNotFound
	}
	cloned := *item
	return &cloned, nil
}

func (s *credentialStoreStub) List(_ string) ([]model.Credential, error) {
	var items []model.Credential
	for _, item := range s.items {
		items = append(items, *item)
	}
	return items, nil
}

func (s *credentialStoreStub) Delete(id string) error {
	if _, ok := s.items[id]; !ok {
		return credstore.ErrCredentialNotFound
	}
	delete(s.items, id)
	return nil
}

func TestCredentialService_CreateAndUpdateMapInvalidAndMissingErrors(t *testing.T) {
	store := &credentialStoreStub{items: map[string]*model.Credential{
		"cred-1": {ID: "cred-1", Name: "ssh", Type: "ssh_key", SSHUsername: "admin", SSHKeyID: "key-1"},
	}}
	rbac := newServiceTestStorage()
	rbac.setPermission("user-1", "credentials", "create", true)
	rbac.setPermission("user-1", "credentials", "update", true)
	svc := NewCredentialService(store, rbac)

	store.createErr = credstore.ErrInvalidCredential
	_, err := svc.Create(userContext("user-1"), &model.CredentialInput{Name: "bad", Type: "invalid"})
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation-wrapped create error, got %v", err)
	}

	store.createErr = nil
	_, err = svc.Update(userContext("user-1"), "missing", &model.CredentialInput{Name: "ssh", Type: "ssh_key", SSHUsername: "admin", SSHKeyID: "key-1"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found on missing credential update, got %v", err)
	}
}
