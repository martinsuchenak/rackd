package service

import (
	"context"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type ConflictService struct {
	store storage.ExtendedStorage
}

func NewConflictService(store storage.ExtendedStorage) *ConflictService {
	return &ConflictService{store: store}
}

// List returns all conflicts matching the filter
func (s *ConflictService) List(ctx context.Context, filter *model.ConflictFilter) ([]model.Conflict, error) {
	if err := requirePermission(ctx, s.store, "conflicts", "list"); err != nil {
		return nil, err
	}

	conflicts, err := s.store.ListConflicts(ctx, filter)
	if err != nil {
		return nil, err
	}

	return conflicts, nil
}

// Get returns a single conflict by ID
func (s *ConflictService) Get(ctx context.Context, id string) (*model.Conflict, error) {
	if err := requirePermission(ctx, s.store, "conflicts", "read"); err != nil {
		return nil, err
	}

	conflict, err := s.store.GetConflict(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	return conflict, nil
}

// Resolve resolves a conflict by updating its status and recording who resolved it
func (s *ConflictService) Resolve(ctx context.Context, resolution *model.ConflictResolution) error {
	if err := requirePermission(ctx, s.store, "conflicts", "resolve"); err != nil {
		return err
	}

	if resolution.ConflictID == "" {
		return ValidationErrors{{Field: "conflict_id", Message: "Conflict ID is required"}}
	}

	caller := CallerFrom(ctx)
	resolvedBy := ""
	if caller != nil {
		resolvedBy = caller.UserID
	}

	// Get the conflict to determine its type
	conflict, err := s.store.GetConflict(ctx, resolution.ConflictID)
	if err != nil {
		return ErrNotFound
	}

	// For duplicate IP conflicts, update the addresses of devices that should lose the IP
	if conflict.Type == model.ConflictTypeDuplicateIP && resolution.KeepDeviceID != "" {
		if err := s.resolveDuplicateIP(ctx, resolution, resolvedBy); err != nil {
			return err
		}
	}

	// Update conflict status to resolved
	if err := s.store.UpdateConflictStatus(enrichAuditCtx(ctx), resolution.ConflictID, model.ConflictStatusResolved, resolvedBy, resolution.Notes); err != nil {
		return err
	}

	return nil
}

// resolveDuplicateIP removes the IP from devices that should not have it
func (s *ConflictService) resolveDuplicateIP(ctx context.Context, resolution *model.ConflictResolution, _ string) error {
	conflict, err := s.store.GetConflict(ctx, resolution.ConflictID)
	if err != nil {
		return err
	}

	// For each device that should lose the IP, remove the address
	for _, deviceID := range conflict.DeviceIDs {
		if deviceID == resolution.KeepDeviceID {
			continue // This device keeps the IP
		}

		// Get the device
		device, err := s.store.GetDevice(ctx, deviceID)
		if err != nil {
			continue // Skip if device not found
		}

		// Remove the conflicting IP from addresses
		var newAddresses []model.Address
		for _, addr := range device.Addresses {
			if addr.IP != conflict.IPAddress {
				newAddresses = append(newAddresses, addr)
			}
		}

		device.Addresses = newAddresses

		// Update the device
		if err := s.store.UpdateDevice(enrichAuditCtx(ctx), device); err != nil {
			return err
		}
	}

	return nil
}

// DetectDuplicateIPs scans for and creates conflict records for duplicate IPs
func (s *ConflictService) DetectDuplicateIPs(ctx context.Context) ([]model.Conflict, error) {
	if err := requirePermission(ctx, s.store, "conflicts", "detect"); err != nil {
		return nil, err
	}

	conflicts, err := s.store.FindDuplicateIPs(ctx)
	if err != nil {
		return nil, err
	}

	// Store any new conflicts
	for _, conflict := range conflicts {
		// Check if conflict already exists for this IP
		existing, err := s.store.GetConflictsByIP(ctx, conflict.IPAddress)
		if err == nil && len(existing) > 0 {
			// Update existing conflict if still active
			for _, ex := range existing {
				if ex.Status == model.ConflictStatusActive {
					// Mark as detected (refresh timestamp)
					s.store.UpdateConflictStatus(ctx, ex.ID, ex.Status, "", "")
				}
			}
		} else {
			// Create new conflict record
			if err := s.store.CreateConflict(ctx, &conflict); err != nil {
				return nil, err
			}
		}
	}

	return conflicts, nil
}

// DetectOverlappingSubnets scans for and creates conflict records for overlapping subnets
func (s *ConflictService) DetectOverlappingSubnets(ctx context.Context) ([]model.Conflict, error) {
	if err := requirePermission(ctx, s.store, "conflicts", "detect"); err != nil {
		return nil, err
	}

	conflicts, err := s.store.FindOverlappingSubnets(ctx)
	if err != nil {
		return nil, err
	}

	// Store any new conflicts
	for _, conflict := range conflicts {
		// Check if this exact conflict already exists
		existing, err := s.store.ListConflicts(ctx, &model.ConflictFilter{
			Type:   model.ConflictTypeOverlappingSubnet,
			Status: model.ConflictStatusActive,
		})
		if err == nil {
			// Check for duplicate by comparing network IDs
			isDuplicate := false
			for _, ex := range existing {
				if sameNetworks(ex.NetworkIDs, conflict.NetworkIDs) {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				if err := s.store.CreateConflict(ctx, &conflict); err != nil {
					return nil, err
				}
			}
		} else {
			if err := s.store.CreateConflict(ctx, &conflict); err != nil {
				return nil, err
			}
		}
	}

	return conflicts, nil
}

// sameNetworks checks if two network ID slices contain the same networks
func sameNetworks(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aMap := make(map[string]bool)
	for _, id := range a {
		aMap[id] = true
	}
	for _, id := range b {
		if !aMap[id] {
			return false
		}
	}
	return true
}

// GetConflictsByDevice returns all conflicts involving a specific device
func (s *ConflictService) GetConflictsByDevice(ctx context.Context, deviceID string) ([]model.Conflict, error) {
	if err := requirePermission(ctx, s.store, "conflicts", "read"); err != nil {
		return nil, err
	}

	return s.store.GetConflictsByDeviceID(ctx, deviceID)
}

// Delete removes a conflict (requires conflict:delete permission)
func (s *ConflictService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "conflicts", "delete"); err != nil {
		return err
	}

	if id == "" {
		return ValidationErrors{{Field: "id", Message: "ID is required"}}
	}

	if err := s.store.DeleteConflict(ctx, id); err != nil {
		return err
	}

	return nil
}

// GetSummary returns a summary of active conflicts
func (s *ConflictService) GetSummary(ctx context.Context) (map[string]int, error) {
	if err := requirePermission(ctx, s.store, "conflicts", "read"); err != nil {
		return nil, err
	}

	conflicts, err := s.store.ListConflicts(ctx, &model.ConflictFilter{
		Status: model.ConflictStatusActive,
	})
	if err != nil {
		return nil, err
	}

	summary := make(map[string]int)
	for _, c := range conflicts {
		summary[string(c.Type)]++
	}

	return summary, nil
}
