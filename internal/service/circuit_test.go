package service

import (
	"errors"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestCircuitService_CreateDefaultsStatusAndUpdateValidatesStatus(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "circuits", "create", true)
	store.setPermission("user-1", "circuits", "update", true)
	store.circuits["circuit-1"] = &model.Circuit{
		ID:        "circuit-1",
		Name:      "primary",
		CircuitID: "ckt-1",
		Provider:  "isp",
		Status:    model.CircuitStatusActive,
	}
	svc := NewCircuitService(store)

	circuit, err := svc.Create(userContext("user-1"), &model.CreateCircuitRequest{
		Name:      "wan-a",
		CircuitID: "ckt-a",
		Provider:  "isp",
	})
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}
	if circuit.Status != model.CircuitStatusActive {
		t.Fatalf("expected default circuit status active, got %q", circuit.Status)
	}
	if circuit.ID == "" {
		t.Fatal("expected created circuit to receive an ID")
	}

	invalidStatus := model.CircuitStatus("bad")
	_, err = svc.Update(userContext("user-1"), "circuit-1", &model.UpdateCircuitRequest{Status: &invalidStatus})
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for invalid circuit status, got %v", err)
	}
}
