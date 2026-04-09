package storage

import (
	"context"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestCircuitStorageCRUDAndFilters(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	ctx := context.Background()
	dcA := &model.Datacenter{Name: "DC A", Location: "A"}
	dcB := &model.Datacenter{Name: "DC B", Location: "B"}
	if err := storage.CreateDatacenter(ctx, dcA); err != nil {
		t.Fatalf("CreateDatacenter A failed: %v", err)
	}
	if err := storage.CreateDatacenter(ctx, dcB); err != nil {
		t.Fatalf("CreateDatacenter B failed: %v", err)
	}

	circuit := &model.Circuit{
		ID:            "circuit-test-1",
		Name:          "Primary WAN",
		CircuitID:     "WAN-001",
		Provider:      "AcmeTel",
		Type:          "fiber",
		Status:        model.CircuitStatusActive,
		CapacityMbps:  1000,
		DatacenterAID: dcA.ID,
		DatacenterBID: dcB.ID,
		Tags:          []string{"wan", "critical"},
	}
	if err := storage.CreateCircuit(ctx, circuit); err != nil {
		t.Fatalf("CreateCircuit failed: %v", err)
	}

	got, err := storage.GetCircuit(ctx, circuit.ID)
	if err != nil {
		t.Fatalf("GetCircuit failed: %v", err)
	}
	if got.CircuitID != circuit.CircuitID || len(got.Tags) != 2 {
		t.Fatalf("unexpected circuit after create: %+v", got)
	}

	got, err = storage.GetCircuitByCircuitID(ctx, circuit.CircuitID)
	if err != nil {
		t.Fatalf("GetCircuitByCircuitID failed: %v", err)
	}
	if got.ID != circuit.ID {
		t.Fatalf("expected same circuit by circuit_id lookup, got %+v", got)
	}

	list, err := storage.ListCircuits(ctx, &model.CircuitFilter{Provider: "AcmeTel", Status: model.CircuitStatusActive})
	if err != nil {
		t.Fatalf("ListCircuits failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 listed circuit, got %d", len(list))
	}

	circuit.Description = "updated"
	circuit.Status = model.CircuitStatusMaintenance
	if err := storage.UpdateCircuit(ctx, circuit); err != nil {
		t.Fatalf("UpdateCircuit failed: %v", err)
	}

	got, err = storage.GetCircuit(ctx, circuit.ID)
	if err != nil {
		t.Fatalf("GetCircuit after update failed: %v", err)
	}
	if got.Description != "updated" || got.Status != model.CircuitStatusMaintenance {
		t.Fatalf("unexpected circuit after update: %+v", got)
	}

	if err := storage.DeleteCircuit(ctx, circuit.ID); err != nil {
		t.Fatalf("DeleteCircuit failed: %v", err)
	}
	if _, err := storage.GetCircuit(ctx, circuit.ID); err != ErrCircuitNotFound {
		t.Fatalf("expected ErrCircuitNotFound, got %v", err)
	}
}

func TestCircuitStorageNotFoundAndValidation(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	ctx := context.Background()
	if _, err := storage.GetCircuit(ctx, "missing"); err != ErrCircuitNotFound {
		t.Fatalf("expected ErrCircuitNotFound, got %v", err)
	}
	if _, err := storage.GetCircuitByCircuitID(ctx, "missing"); err != ErrCircuitNotFound {
		t.Fatalf("expected ErrCircuitNotFound, got %v", err)
	}
	if err := storage.DeleteCircuit(ctx, "missing"); err != ErrCircuitNotFound {
		t.Fatalf("expected ErrCircuitNotFound, got %v", err)
	}
}
