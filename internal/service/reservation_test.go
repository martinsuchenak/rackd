package service

import (
	"context"
	"errors"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

func TestReservationService_CreateRetriesAutoAssignedConflictsAndTracksCaller(t *testing.T) {
	store := newServiceTestStorage()
	store.pools["pool-1"] = true
	store.nextIPs = []string{"10.0.0.10", "10.0.0.11"}
	store.createReservationErrs = []error{storage.ErrIPAlreadyReserved, nil}
	store.setPermission("user-1", "reservations", "create", true)
	svc := NewReservationService(store)

	reservation, err := svc.Create(userContext("user-1"), &model.CreateReservationRequest{
		PoolID:   "pool-1",
		Hostname: "printer-1",
	})
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}
	if store.nextIPCalls != 2 {
		t.Fatalf("expected 2 next-IP attempts, got %d", store.nextIPCalls)
	}
	if reservation.IPAddress != "10.0.0.11" {
		t.Fatalf("expected second auto-assigned IP to succeed, got %q", reservation.IPAddress)
	}
	if reservation.ReservedBy != "user-1" {
		t.Fatalf("expected reservation to track caller user ID, got %q", reservation.ReservedBy)
	}
}

func TestReservationService_ReleaseAndClaimUpdateStatuses(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "reservations", "update", true)
	store.reservations["res-1"] = &model.Reservation{
		ID:         "res-1",
		PoolID:     "pool-1",
		IPAddress:  "10.0.0.10",
		ReservedBy: "user-1",
		Status:     model.ReservationStatusActive,
	}
	svc := NewReservationService(store)

	if err := svc.Release(userContext("user-1"), "res-1"); err != nil {
		t.Fatalf("Release returned unexpected error: %v", err)
	}
	if store.reservationSet == nil || store.reservationSet.Status != model.ReservationStatusReleased {
		t.Fatalf("expected released reservation status, got %#v", store.reservationSet)
	}

	store.reservationSet = nil
	if err := svc.Claim(context.Background(), "pool-1", "10.0.0.10"); err != nil {
		t.Fatalf("Claim returned unexpected error: %v", err)
	}
	if store.reservationSet == nil || store.reservationSet.Status != model.ReservationStatusClaimed {
		t.Fatalf("expected claimed reservation status, got %#v", store.reservationSet)
	}
}

func TestReservationService_GetNextAvailableIPReturnsFirstAvailableAndMapsExhaustion(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "reservations", "create", true)
	store.pools["pool-1"] = true
	store.poolHeatmap = []storage.IPStatus{
		{IP: "10.0.0.10", Status: "used"},
		{IP: "10.0.0.11", Status: "available"},
	}
	svc := NewReservationService(store)

	ip, err := svc.GetNextAvailableIP(userContext("user-1"), "pool-1")
	if err != nil {
		t.Fatalf("GetNextAvailableIP returned unexpected error: %v", err)
	}
	if ip != "10.0.0.11" {
		t.Fatalf("expected first available IP, got %q", ip)
	}

	store.poolHeatmap = []storage.IPStatus{{IP: "10.0.0.12", Status: "used"}}
	_, err = svc.GetNextAvailableIP(userContext("user-1"), "pool-1")
	if !errors.Is(err, storage.ErrIPNotAvailable) {
		t.Fatalf("expected storage.ErrIPNotAvailable, got %v", err)
	}
}

func TestReservationService_UpdateAndDeleteMapValidationAndNotFound(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "reservations", "update", true)
	store.setPermission("user-1", "reservations", "delete", true)
	store.reservations["res-1"] = &model.Reservation{
		ID:         "res-1",
		PoolID:     "pool-1",
		IPAddress:  "10.0.0.10",
		ReservedBy: "user-1",
		Status:     model.ReservationStatusClaimed,
	}
	svc := NewReservationService(store)

	_, err := svc.Update(userContext("user-1"), "res-1", &model.UpdateReservationRequest{Hostname: "new-name"})
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for non-active reservation update, got %v", err)
	}

	err = svc.Delete(userContext("user-1"), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found on delete, got %v", err)
	}
}

func TestReservationService_MutationsEnforceOwnership(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "reservations", "update", true)
	store.setPermission("user-1", "reservations", "delete", true)
	store.setPermission("user-2", "reservations", "update", true)
	store.setPermission("user-2", "reservations", "delete", true)
	store.setPermission("admin-1", "reservations", "update", true)
	store.setPermission("admin-1", "reservations", "delete", true)
	store.userRoles["admin-1"] = []model.Role{{ID: "admin", Name: "admin"}}
	store.reservations["res-1"] = &model.Reservation{
		ID:         "res-1",
		PoolID:     "pool-1",
		IPAddress:  "10.0.0.10",
		ReservedBy: "user-1",
		Status:     model.ReservationStatusActive,
	}
	svc := NewReservationService(store)

	// Non-owner with the same permissions is rejected on all three mutations
	_, err := svc.Update(userContext("user-2"), "res-1", &model.UpdateReservationRequest{Hostname: "hijack"})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden for non-owner update, got %v", err)
	}
	if err := svc.Release(userContext("user-2"), "res-1"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden for non-owner release, got %v", err)
	}
	if err := svc.Delete(userContext("user-2"), "res-1"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden for non-owner delete, got %v", err)
	}
	if store.reservations["res-1"].Status != model.ReservationStatusActive {
		t.Fatalf("expected reservation untouched by non-owner, got %q", store.reservations["res-1"].Status)
	}

	// Owner can still update and release their own reservation
	if _, err := svc.Update(userContext("user-1"), "res-1", &model.UpdateReservationRequest{Hostname: "new-name"}); err != nil {
		t.Fatalf("owner update returned unexpected error: %v", err)
	}
	if err := svc.Release(userContext("user-1"), "res-1"); err != nil {
		t.Fatalf("owner release returned unexpected error: %v", err)
	}

	// Admin can mutate reservations owned by others
	store.reservations["res-2"] = &model.Reservation{
		ID:         "res-2",
		PoolID:     "pool-1",
		IPAddress:  "10.0.0.11",
		ReservedBy: "user-1",
		Status:     model.ReservationStatusActive,
	}
	if _, err := svc.Update(userContext("admin-1"), "res-2", &model.UpdateReservationRequest{Hostname: "admin-edit"}); err != nil {
		t.Fatalf("admin update returned unexpected error: %v", err)
	}
	if err := svc.Delete(userContext("admin-1"), "res-2"); err != nil {
		t.Fatalf("admin delete returned unexpected error: %v", err)
	}
}
