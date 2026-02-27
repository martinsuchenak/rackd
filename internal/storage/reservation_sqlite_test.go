package storage

import (
	"context"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

// ============================================================================
// Reservation Operations Tests
// ============================================================================

func TestReservationOperations_CreateAndGet(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool first
	network := &model.Network{Name: "Test Network", Subnet: "192.168.1.0/24"}
	if err := storage.CreateNetwork(context.Background(), network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Test Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	if err := storage.CreateNetworkPool(context.Background(), pool); err != nil {
		t.Fatalf("CreateNetworkPool failed: %v", err)
	}

	reservation := &model.Reservation{
		PoolID:     pool.ID,
		IPAddress:  "192.168.1.100",
		Hostname:   "server1.example.com",
		Purpose:    "Web server",
		ReservedBy: "admin",
		Status:     model.ReservationStatusActive,
	}

	// Create reservation
	err := storage.CreateReservation(context.Background(), reservation)
	if err != nil {
		t.Fatalf("CreateReservation failed: %v", err)
	}

	if reservation.ID == "" {
		t.Error("reservation ID should be set after creation")
	}
	if reservation.ReservedAt.IsZero() {
		t.Error("reserved_at should be set after creation")
	}

	// Get reservation
	retrieved, err := storage.GetReservation(reservation.ID)
	if err != nil {
		t.Fatalf("GetReservation failed: %v", err)
	}

	if retrieved.PoolID != reservation.PoolID {
		t.Errorf("expected pool_id %s, got %s", reservation.PoolID, retrieved.PoolID)
	}
	if retrieved.IPAddress != reservation.IPAddress {
		t.Errorf("expected ip_address %s, got %s", reservation.IPAddress, retrieved.IPAddress)
	}
	if retrieved.Hostname != reservation.Hostname {
		t.Errorf("expected hostname %s, got %s", reservation.Hostname, retrieved.Hostname)
	}
	if retrieved.Purpose != reservation.Purpose {
		t.Errorf("expected purpose %s, got %s", reservation.Purpose, retrieved.Purpose)
	}
	if retrieved.ReservedBy != reservation.ReservedBy {
		t.Errorf("expected reserved_by %s, got %s", reservation.ReservedBy, retrieved.ReservedBy)
	}
	if retrieved.Status != reservation.Status {
		t.Errorf("expected status %s, got %s", reservation.Status, retrieved.Status)
	}
}

func TestReservationOperations_CreateWithExpiration(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Test Network", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Test Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	expiresAt := time.Now().Add(7 * 24 * time.Hour).UTC()
	reservation := &model.Reservation{
		PoolID:     pool.ID,
		IPAddress:  "192.168.1.101",
		ReservedBy: "admin",
		ExpiresAt:  &expiresAt,
		Status:     model.ReservationStatusActive,
	}

	err := storage.CreateReservation(context.Background(), reservation)
	if err != nil {
		t.Fatalf("CreateReservation failed: %v", err)
	}

	retrieved, err := storage.GetReservation(reservation.ID)
	if err != nil {
		t.Fatalf("GetReservation failed: %v", err)
	}

	if retrieved.ExpiresAt == nil {
		t.Error("expires_at should be set")
	}
}

func TestReservationOperations_GetNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetReservation("non-existent-id")
	if err == nil {
		t.Error("expected error for non-existent reservation")
	}
	if err != ErrReservationNotFound {
		t.Errorf("expected ErrReservationNotFound, got %v", err)
	}
}

func TestReservationOperations_GetInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetReservation("")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestReservationOperations_CreateInvalidPool(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	reservation := &model.Reservation{
		PoolID:     "non-existent-pool",
		IPAddress:  "192.168.1.100",
		ReservedBy: "admin",
		Status:     model.ReservationStatusActive,
	}

	err := storage.CreateReservation(context.Background(), reservation)
	if err == nil {
		t.Error("expected error for non-existent pool")
	}
	if err != ErrPoolNotFound {
		t.Errorf("expected ErrPoolNotFound, got %v", err)
	}
}

func TestReservationOperations_CreateIPNotInPoolRange(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Test Network", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Test Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	reservation := &model.Reservation{
		PoolID:     pool.ID,
		IPAddress:  "10.0.0.1", // Not in pool range
		ReservedBy: "admin",
		Status:     model.ReservationStatusActive,
	}

	err := storage.CreateReservation(context.Background(), reservation)
	if err == nil {
		t.Error("expected error for IP not in pool range")
	}
}

func TestReservationOperations_CreateDuplicateIP(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Test Network", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Test Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Create first reservation
	reservation1 := &model.Reservation{
		PoolID:     pool.ID,
		IPAddress:  "192.168.1.100",
		ReservedBy: "admin",
		Status:     model.ReservationStatusActive,
	}
	if err := storage.CreateReservation(context.Background(), reservation1); err != nil {
		t.Fatalf("CreateReservation 1 failed: %v", err)
	}

	// Try to create second reservation with same IP
	reservation2 := &model.Reservation{
		PoolID:     pool.ID,
		IPAddress:  "192.168.1.100",
		ReservedBy: "user2",
		Status:     model.ReservationStatusActive,
	}
	err := storage.CreateReservation(context.Background(), reservation2)
	if err == nil {
		t.Error("expected error for duplicate IP reservation")
	}
	if err != ErrIPAlreadyReserved {
		t.Errorf("expected ErrIPAlreadyReserved, got %v", err)
	}
}

func TestReservationOperations_ListAll(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Test Network", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Test Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Create multiple reservations
	storage.CreateReservation(context.Background(), &model.Reservation{
		PoolID:     pool.ID,
		IPAddress:  "192.168.1.100",
		ReservedBy: "admin",
		Status:     model.ReservationStatusActive,
	})
	storage.CreateReservation(context.Background(), &model.Reservation{
		PoolID:     pool.ID,
		IPAddress:  "192.168.1.101",
		ReservedBy: "user1",
		Status:     model.ReservationStatusActive,
	})

	// List all reservations
	reservations, err := storage.ListReservations(nil)
	if err != nil {
		t.Fatalf("ListReservations failed: %v", err)
	}

	if len(reservations) != 2 {
		t.Errorf("expected 2 reservations, got %d", len(reservations))
	}
}

func TestReservationOperations_ListWithPoolFilter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pools
	network := &model.Network{Name: "Test Network", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool1 := &model.NetworkPool{NetworkID: network.ID, Name: "Pool 1", StartIP: "192.168.1.100", EndIP: "192.168.1.150"}
	pool2 := &model.NetworkPool{NetworkID: network.ID, Name: "Pool 2", StartIP: "192.168.1.151", EndIP: "192.168.1.200"}
	storage.CreateNetworkPool(context.Background(), pool1)
	storage.CreateNetworkPool(context.Background(), pool2)

	// Create reservations in different pools
	storage.CreateReservation(context.Background(), &model.Reservation{
		PoolID:     pool1.ID,
		IPAddress:  "192.168.1.100",
		ReservedBy: "admin",
		Status:     model.ReservationStatusActive,
	})
	storage.CreateReservation(context.Background(), &model.Reservation{
		PoolID:     pool2.ID,
		IPAddress:  "192.168.1.151",
		ReservedBy: "user1",
		Status:     model.ReservationStatusActive,
	})

	// Filter by pool1
	reservations, err := storage.ListReservations(&model.ReservationFilter{PoolID: pool1.ID})
	if err != nil {
		t.Fatalf("ListReservations failed: %v", err)
	}

	if len(reservations) != 1 {
		t.Errorf("expected 1 reservation in pool1, got %d", len(reservations))
	}
	if reservations[0].PoolID != pool1.ID {
		t.Errorf("expected pool1 ID, got %s", reservations[0].PoolID)
	}
}

func TestReservationOperations_ListWithStatusFilter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Test Network", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Test Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Create reservations with different statuses
	storage.CreateReservation(context.Background(), &model.Reservation{
		PoolID:     pool.ID,
		IPAddress:  "192.168.1.100",
		ReservedBy: "admin",
		Status:     model.ReservationStatusActive,
	})
	reservation2 := &model.Reservation{
		PoolID:     pool.ID,
		IPAddress:  "192.168.1.101",
		ReservedBy: "admin",
		Status:     model.ReservationStatusActive,
	}
	storage.CreateReservation(context.Background(), reservation2)

	// Expire one reservation
	storage.UpdateReservation(context.Background(), &model.Reservation{
		ID:     reservation2.ID,
		Status: model.ReservationStatusExpired,
	})

	// Filter by active status
	reservations, err := storage.ListReservations(&model.ReservationFilter{Status: model.ReservationStatusActive})
	if err != nil {
		t.Fatalf("ListReservations failed: %v", err)
	}

	if len(reservations) != 1 {
		t.Errorf("expected 1 active reservation, got %d", len(reservations))
	}
	if reservations[0].Status != model.ReservationStatusActive {
		t.Errorf("expected status active, got %s", reservations[0].Status)
	}
}

func TestReservationOperations_Update(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Test Network", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Test Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Create reservation
	reservation := &model.Reservation{
		PoolID:     pool.ID,
		IPAddress:  "192.168.1.100",
		Hostname:   "old-hostname",
		Purpose:    "Old purpose",
		ReservedBy: "admin",
		Status:     model.ReservationStatusActive,
	}
	storage.CreateReservation(context.Background(), reservation)

	// Update reservation
	reservation.Hostname = "new-hostname"
	reservation.Purpose = "New purpose"
	reservation.Notes = "Updated notes"

	err := storage.UpdateReservation(context.Background(), reservation)
	if err != nil {
		t.Fatalf("UpdateReservation failed: %v", err)
	}

	// Verify update
	retrieved, err := storage.GetReservation(reservation.ID)
	if err != nil {
		t.Fatalf("GetReservation failed: %v", err)
	}

	if retrieved.Hostname != "new-hostname" {
		t.Errorf("expected hostname 'new-hostname', got %s", retrieved.Hostname)
	}
	if retrieved.Purpose != "New purpose" {
		t.Errorf("expected purpose 'New purpose', got %s", retrieved.Purpose)
	}
	if retrieved.Notes != "Updated notes" {
		t.Errorf("expected notes 'Updated notes', got %s", retrieved.Notes)
	}
}

func TestReservationOperations_Delete(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Test Network", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Test Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Create reservation
	reservation := &model.Reservation{
		PoolID:     pool.ID,
		IPAddress:  "192.168.1.100",
		ReservedBy: "admin",
		Status:     model.ReservationStatusActive,
	}
	storage.CreateReservation(context.Background(), reservation)

	// Delete reservation
	err := storage.DeleteReservation(context.Background(), reservation.ID)
	if err != nil {
		t.Fatalf("DeleteReservation failed: %v", err)
	}

	// Verify deletion
	_, err = storage.GetReservation(reservation.ID)
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestReservationOperations_DeleteInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.DeleteReservation(context.Background(), "")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestReservationOperations_GetByPool(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Test Network", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Test Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Create reservations
	storage.CreateReservation(context.Background(), &model.Reservation{
		PoolID:     pool.ID,
		IPAddress:  "192.168.1.100",
		ReservedBy: "admin",
		Status:     model.ReservationStatusActive,
	})
	storage.CreateReservation(context.Background(), &model.Reservation{
		PoolID:     pool.ID,
		IPAddress:  "192.168.1.101",
		ReservedBy: "user1",
		Status:     model.ReservationStatusActive,
	})

	// Get reservations by pool
	reservations, err := storage.GetReservationsByPool(pool.ID)
	if err != nil {
		t.Fatalf("GetReservationsByPool failed: %v", err)
	}

	if len(reservations) != 2 {
		t.Errorf("expected 2 reservations, got %d", len(reservations))
	}
}

func TestReservationOperations_GetByPoolInvalidID(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetReservationsByPool("")
	if err != ErrInvalidID {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestReservationOperations_GetByUser(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Test Network", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Test Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Create reservations by different users
	storage.CreateReservation(context.Background(), &model.Reservation{
		PoolID:     pool.ID,
		IPAddress:  "192.168.1.100",
		ReservedBy: "admin",
		Status:     model.ReservationStatusActive,
	})
	storage.CreateReservation(context.Background(), &model.Reservation{
		PoolID:     pool.ID,
		IPAddress:  "192.168.1.101",
		ReservedBy: "user1",
		Status:     model.ReservationStatusActive,
	})
	storage.CreateReservation(context.Background(), &model.Reservation{
		PoolID:     pool.ID,
		IPAddress:  "192.168.1.102",
		ReservedBy: "admin",
		Status:     model.ReservationStatusActive,
	})

	// Get reservations by user
	reservations, err := storage.GetReservationsByUser("admin")
	if err != nil {
		t.Fatalf("GetReservationsByUser failed: %v", err)
	}

	if len(reservations) != 2 {
		t.Errorf("expected 2 reservations by admin, got %d", len(reservations))
	}
}

func TestReservationOperations_IsIPReserved(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Test Network", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Test Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Check IP not reserved
	isReserved, err := storage.IsIPReserved(pool.ID, "192.168.1.100")
	if err != nil {
		t.Fatalf("IsIPReserved failed: %v", err)
	}
	if isReserved {
		t.Error("IP should not be reserved")
	}

	// Create reservation
	storage.CreateReservation(context.Background(), &model.Reservation{
		PoolID:     pool.ID,
		IPAddress:  "192.168.1.100",
		ReservedBy: "admin",
		Status:     model.ReservationStatusActive,
	})

	// Check IP is now reserved
	isReserved, err = storage.IsIPReserved(pool.ID, "192.168.1.100")
	if err != nil {
		t.Fatalf("IsIPReserved failed: %v", err)
	}
	if !isReserved {
		t.Error("IP should be reserved")
	}
}

func TestReservationOperations_ExpireReservations(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Test Network", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Test Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Create reservation that has expired
	pastTime := time.Now().Add(-24 * time.Hour).UTC()
	storage.CreateReservation(context.Background(), &model.Reservation{
		PoolID:     pool.ID,
		IPAddress:  "192.168.1.100",
		ReservedBy: "admin",
		Status:     model.ReservationStatusActive,
		ExpiresAt:  &pastTime,
	})

	// Create reservation that has not expired
	futureTime := time.Now().Add(24 * time.Hour).UTC()
	storage.CreateReservation(context.Background(), &model.Reservation{
		PoolID:     pool.ID,
		IPAddress:  "192.168.1.101",
		ReservedBy: "admin",
		Status:     model.ReservationStatusActive,
		ExpiresAt:  &futureTime,
	})

	// Expire reservations
	count, err := storage.ExpireReservations(context.Background())
	if err != nil {
		t.Fatalf("ExpireReservations failed: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 reservation expired, got %d", count)
	}

	// Verify the expired reservation
	reservations, _ := storage.ListReservations(&model.ReservationFilter{Status: model.ReservationStatusExpired})
	if len(reservations) != 1 {
		t.Errorf("expected 1 expired reservation, got %d", len(reservations))
	}
}

func TestReservationOperations_GetReservationByIP(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Test Network", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Test Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	// Create reservation
	storage.CreateReservation(context.Background(), &model.Reservation{
		PoolID:     pool.ID,
		IPAddress:  "192.168.1.100",
		Hostname:   "server1",
		ReservedBy: "admin",
		Status:     model.ReservationStatusActive,
	})

	// Get by IP
	reservation, err := storage.GetReservationByIP(pool.ID, "192.168.1.100")
	if err != nil {
		t.Fatalf("GetReservationByIP failed: %v", err)
	}

	if reservation.Hostname != "server1" {
		t.Errorf("expected hostname 'server1', got %s", reservation.Hostname)
	}
}

func TestReservationOperations_GetReservationByIPNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create network and pool
	network := &model.Network{Name: "Test Network", Subnet: "192.168.1.0/24"}
	storage.CreateNetwork(context.Background(), network)

	pool := &model.NetworkPool{
		NetworkID: network.ID,
		Name:      "Test Pool",
		StartIP:   "192.168.1.100",
		EndIP:     "192.168.1.200",
	}
	storage.CreateNetworkPool(context.Background(), pool)

	_, err := storage.GetReservationByIP(pool.ID, "192.168.1.100")
	if err != ErrReservationNotFound {
		t.Errorf("expected ErrReservationNotFound, got %v", err)
	}
}

func TestReservationOperations_CreateNil(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.CreateReservation(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil reservation")
	}
}
