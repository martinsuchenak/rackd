package service

import (
	"context"
	"sort"
	"strings"

	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
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
		// Taking the IP off the other devices changes them.
		if err := requirePermission(ctx, s.store, "devices", "update"); err != nil {
			return err
		}
		if err := s.resolveDuplicateIP(ctx, resolution, resolvedBy); err != nil {
			return err
		}
	}

	// Update conflict status to resolved
	if err := s.store.UpdateConflictStatus(enrichAuditCtx(ctx), resolution.ConflictID, model.ConflictStatusResolved, resolvedBy, resolution.Notes); err != nil {
		return err
	}
	if resolved, err := s.store.GetConflict(ctx, resolution.ConflictID); err == nil {
		publishEvent(model.EventTypeConflictResolved, resolved)
	}

	recordAudit(ctx, s.store, "resolve", "conflict", resolution.ConflictID, resolution)
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
					_ = s.store.UpdateConflictStatus(ctx, ex.ID, ex.Status, "", "")
				}
			}
		} else {
			// Create new conflict record
			if err := s.store.CreateConflict(ctx, &conflict); err != nil {
				return nil, err
			}
			publishEvent(model.EventTypeConflictDetected, conflict)
		}
	}

	return conflicts, nil
}

// DetectOverlappingSubnets scans for and creates conflict records for overlapping subnets
func (s *ConflictService) DetectOverlappingSubnets(ctx context.Context) ([]model.Conflict, error) {
	if err := requirePermission(ctx, s.store, "conflicts", "detect"); err != nil {
		return nil, err
	}

	// Read the recorded conflicts once, and skip them while searching, so the
	// per-run cap applies to new overlaps only.
	existing, err := s.store.ListConflicts(ctx, &model.ConflictFilter{
		Type:   model.ConflictTypeOverlappingSubnet,
		Status: model.ConflictStatusActive,
	})
	if err != nil {
		return nil, err
	}
	recorded := make(map[string]bool, len(existing))
	for _, ex := range existing {
		recorded[networkPairKey(ex.NetworkIDs)] = true
	}

	conflicts, err := s.store.FindOverlappingSubnets(ctx, func(ids []string) bool {
		return recorded[networkPairKey(ids)]
	})
	if err != nil {
		return nil, err
	}
	for _, conflict := range conflicts {
		if err := s.store.CreateConflict(ctx, &conflict); err != nil {
			return nil, err
		}
		publishEvent(model.EventTypeConflictDetected, conflict)
	}

	return conflicts, nil
}

// networkPairKey identifies a set of networks regardless of order.
func networkPairKey(ids []string) string {
	sorted := append([]string(nil), ids...)
	sort.Strings(sorted)
	return strings.Join(sorted, "\x00")
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

	recordAudit(ctx, s.store, "delete", "conflict", id, nil)
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
