package service

import (
	"context"
	"errors"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestNATService_CreateRejectsInvalidInputAndDefaultsProtocol(t *testing.T) {
	store := newServiceTestStorage()
	svc := NewNATService(store)
	ctx := SystemContext(context.Background(), "test")

	_, err := svc.Create(ctx, &model.CreateNATRequest{
		Name:       "bad-nat",
		ExternalIP: "not-an-ip",
		InternalIP: "10.0.0.10",
	})
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for invalid external IP, got %v", err)
	}

	mapping, err := svc.Create(ctx, &model.CreateNATRequest{
		Name:         "ssh",
		ExternalIP:   "203.0.113.10",
		ExternalPort: 22,
		InternalIP:   "10.0.0.10",
		InternalPort: 22,
	})
	if err != nil {
		t.Fatalf("expected valid NAT mapping to be created, got %v", err)
	}
	if mapping.Protocol != model.NATProtocolTCP {
		t.Fatalf("expected default protocol tcp, got %q", mapping.Protocol)
	}
	if store.natCreated == nil || store.natCreated.Protocol != model.NATProtocolTCP {
		t.Fatalf("expected persisted NAT mapping with default protocol, got %#v", store.natCreated)
	}
}

func TestNATService_UpdateAndDeleteMapValidationAndNotFound(t *testing.T) {
	store := newServiceTestStorage()
	store.natMappings["nat-1"] = &model.NATMapping{
		ID:         "nat-1",
		Name:       "ssh",
		ExternalIP: "203.0.113.10",
		InternalIP: "10.0.0.10",
		Protocol:   model.NATProtocolTCP,
	}
	store.setPermission("user-1", "nat", "update", true)
	store.setPermission("user-1", "nat", "delete", true)
	svc := NewNATService(store)

	badProtocol := model.NATProtocol("icmp")
	_, err := svc.Update(userContext("user-1"), "nat-1", &model.UpdateNATRequest{Protocol: &badProtocol})
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for invalid protocol, got %v", err)
	}

	err = svc.Delete(userContext("user-1"), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found on missing NAT mapping, got %v", err)
	}
}
