package service

import (
	"context"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type ReservationService struct {
	store storage.ExtendedStorage
}

func NewReservationService(store storage.ExtendedStorage) *ReservationService {
	return &ReservationService{store: store}
}

// List returns all reservations matching the filter
func (s *ReservationService) List(ctx context.Context, filter *model.ReservationFilter) ([]model.Reservation, error) {
	if err := requirePermission(ctx, s.store, "reservations", "list"); err != nil {
		return nil, err
	}

	reservations, err := s.store.ListReservations(ctx, filter)
	if err != nil {
		return nil, err
	}

	return reservations, nil
}

// Get returns a single reservation by ID
func (s *ReservationService) Get(ctx context.Context, id string) (*model.Reservation, error) {
	if err := requirePermission(ctx, s.store, "reservations", "read"); err != nil {
		return nil, err
	}

	reservation, err := s.store.GetReservation(ctx, id)
	if err != nil {
		if err == storage.ErrReservationNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return reservation, nil
}

// GetByIP returns a reservation by pool ID and IP address
func (s *ReservationService) GetByIP(ctx context.Context, poolID, ip string) (*model.Reservation, error) {
	if err := requirePermission(ctx, s.store, "reservations", "read"); err != nil {
		return nil, err
	}

	reservation, err := s.store.GetReservationByIP(ctx, poolID, ip)
	if err != nil {
		if err == storage.ErrReservationNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return reservation, nil
}

// Create creates a new IP reservation
func (s *ReservationService) Create(ctx context.Context, req *model.CreateReservationRequest) (*model.Reservation, error) {
	if err := requirePermission(ctx, s.store, "reservations", "create"); err != nil {
		return nil, err
	}

	// Validate required fields
	if req.PoolID == "" {
		return nil, ValidationErrors{{Field: "pool_id", Message: "Pool ID is required"}}
	}

	// Get caller info for reserved_by field
	caller := CallerFrom(ctx)
	reservedBy := "system"
	if caller != nil && caller.UserID != "" {
		reservedBy = caller.UserID
	}

	// If IP not specified, auto-assign one
	autoAssign := req.IPAddress == ""
	var ipAddress string
	var reservation *model.Reservation

	maxRetries := 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		ipAddress = req.IPAddress
		if autoAssign {
			var err error
			ipAddress, err = s.store.GetNextAvailableIP(ctx, req.PoolID)
			if err != nil {
				if err == storage.ErrIPNotAvailable {
					return nil, ValidationErrors{{Field: "ip_address", Message: "No IP addresses available in pool"}}
				}
				return nil, err
			}
		}

		reservation = &model.Reservation{
			PoolID:     req.PoolID,
			IPAddress:  ipAddress,
			Hostname:   req.Hostname,
			Purpose:    req.Purpose,
			ReservedBy: reservedBy,
			ReservedAt: time.Now().UTC(),
			ExpiresAt:  req.ExpiresAt,
			Status:     model.ReservationStatusActive,
			Notes:      req.Notes,
		}

		err := s.store.CreateReservation(enrichAuditCtx(ctx), reservation)
		if err == nil {
			return reservation, nil
		}

		if err == storage.ErrIPConflict {
			if !autoAssign {
				return nil, ValidationErrors{{Field: "ip_address", Message: "IP address is already in use by a device"}}
			}
			continue // try again if auto-assigned
		}
		if err == storage.ErrIPAlreadyReserved {
			if !autoAssign {
				return nil, ValidationErrors{{Field: "ip_address", Message: "IP address is already reserved"}}
			}
			continue // try again if auto-assigned
		}
		if err == storage.ErrPoolNotFound {
			return nil, ValidationErrors{{Field: "pool_id", Message: "Pool not found"}}
		}
		return nil, err
	}

	return nil, ValidationErrors{{Field: "ip_address", Message: "Failed to automatically allocate IP due to high contention, please try again later"}}
}

// Update updates an existing reservation
func (s *ReservationService) Update(ctx context.Context, id string, req *model.UpdateReservationRequest) (*model.Reservation, error) {
	if err := requirePermission(ctx, s.store, "reservations", "update"); err != nil {
		return nil, err
	}

	if id == "" {
		return nil, ValidationErrors{{Field: "id", Message: "ID is required"}}
	}

	// Get existing reservation
	reservation, err := s.store.GetReservation(ctx, id)
	if err != nil {
		if err == storage.ErrReservationNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Check if reservation is still active
	if reservation.Status != model.ReservationStatusActive {
		return nil, ValidationErrors{{Field: "status", Message: "Cannot update a non-active reservation"}}
	}

	// Update fields
	if req.Hostname != "" {
		reservation.Hostname = req.Hostname
	}
	if req.Purpose != "" {
		reservation.Purpose = req.Purpose
	}
	if req.ExpiresAt != nil {
		reservation.ExpiresAt = req.ExpiresAt
	}
	if req.Notes != "" {
		reservation.Notes = req.Notes
	}

	if err := s.store.UpdateReservation(enrichAuditCtx(ctx), reservation); err != nil {
		return nil, err
	}

	return reservation, nil
}

// Delete removes a reservation
func (s *ReservationService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "reservations", "delete"); err != nil {
		return err
	}

	if id == "" {
		return ValidationErrors{{Field: "id", Message: "ID is required"}}
	}

	// Check if reservation exists
	_, err := s.store.GetReservation(ctx, id)
	if err != nil {
		if err == storage.ErrReservationNotFound {
			return ErrNotFound
		}
		return err
	}

	return s.store.DeleteReservation(enrichAuditCtx(ctx), id)
}

// Release releases a reservation (sets status to released)
func (s *ReservationService) Release(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "reservations", "update"); err != nil {
		return err
	}

	if id == "" {
		return ValidationErrors{{Field: "id", Message: "ID is required"}}
	}

	// Get existing reservation
	reservation, err := s.store.GetReservation(ctx, id)
	if err != nil {
		if err == storage.ErrReservationNotFound {
			return ErrNotFound
		}
		return err
	}

	// Update status to released
	reservation.Status = model.ReservationStatusReleased

	return s.store.UpdateReservation(enrichAuditCtx(ctx), reservation)
}

// Claim marks a reservation as claimed (used when a device is assigned the IP)
func (s *ReservationService) Claim(ctx context.Context, poolID, ip string) error {
	// This is called internally when a device is assigned a reserved IP
	// No permission check needed as it's an internal operation

	reservation, err := s.store.GetReservationByIP(ctx, poolID, ip)
	if err != nil {
		if err == storage.ErrReservationNotFound {
			return nil // No reservation for this IP, that's fine
		}
		return err
	}

	// Update status to claimed
	reservation.Status = model.ReservationStatusClaimed

	return s.store.UpdateReservation(enrichAuditCtx(ctx), reservation)
}

// GetByPool returns all active reservations for a pool
func (s *ReservationService) GetByPool(ctx context.Context, poolID string) ([]model.Reservation, error) {
	if err := requirePermission(ctx, s.store, "reservations", "list"); err != nil {
		return nil, err
	}

	return s.store.GetReservationsByPool(ctx, poolID)
}

// GetByUser returns all reservations made by a user
func (s *ReservationService) GetByUser(ctx context.Context, userID string) ([]model.Reservation, error) {
	if err := requirePermission(ctx, s.store, "reservations", "list"); err != nil {
		return nil, err
	}

	return s.store.GetReservationsByUser(ctx, userID)
}

// ExpireExpired marks all expired reservations as expired
func (s *ReservationService) ExpireExpired(ctx context.Context) (int64, error) {
	// This is a system operation, no permission check needed
	return s.store.ExpireReservations(ctx)
}

// IsIPReserved checks if an IP is reserved in a pool
func (s *ReservationService) IsIPReserved(poolID, ip string) (bool, error) {
	return s.store.IsIPReserved(context.Background(), poolID, ip)
}

// GetNextAvailableIP gets the next available IP that is not used or reserved
func (s *ReservationService) GetNextAvailableIP(ctx context.Context, poolID string) (string, error) {
	if err := requirePermission(ctx, s.store, "reservations", "create"); err != nil {
		return "", err
	}

	// Verify pool exists
	_, err := s.store.GetNetworkPool(ctx, poolID)
	if err != nil {
		return "", err
	}

	// Get pool heatmap which already handles used and reserved IPs
	heatmap, err := s.store.GetPoolHeatmap(ctx, poolID)
	if err != nil {
		return "", err
	}

	// Find first available IP
	for _, ip := range heatmap {
		if ip.Status == "available" {
			return ip.IP, nil
		}
	}

	return "", storage.ErrIPNotAvailable
}
