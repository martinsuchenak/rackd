package service

import (
	"context"
	"errors"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type DeviceService struct {
	store           storage.ExtendedStorage
	conflictService *ConflictService
}

func NewDeviceService(store storage.ExtendedStorage) *DeviceService {
	return &DeviceService{store: store}
}

func (s *DeviceService) setConflictService(cs *ConflictService) {
	s.conflictService = cs
}

// checkForIPConflicts checks if any IP addresses are duplicates and creates conflict records
func (s *DeviceService) checkForIPConflicts(ctx context.Context, device *model.Device) {
	if s.conflictService == nil {
		return
	}

	// Check each IP address for duplicates
	for _, addr := range device.Addresses {
		if addr.IP == "" {
			continue
		}

		// Look for existing devices with this IP
		conflicts, err := s.store.GetConflictsByIP(addr.IP)
		if err != nil {
			continue
		}

		// Check if we already have a conflict for this IP
		hasConflict := false
		for _, c := range conflicts {
			if c.Status == model.ConflictStatusActive {
				hasConflict = true
				break
			}
		}

		// If no active conflict exists and we found multiple devices with this IP, create one
		if !hasConflict && len(conflicts) == 0 {
			// Find all devices with this IP (including the current one)
			allDevices, err := s.store.ListDevices(&model.DeviceFilter{})
			if err != nil {
				continue
			}

			var deviceIDs []string
			var deviceNames []string
			for _, d := range allDevices {
				for _, a := range d.Addresses {
					if a.IP == addr.IP {
						deviceIDs = append(deviceIDs, d.ID)
						deviceNames = append(deviceNames, d.Name)
						break
					}
				}
			}

			// Only create conflict if multiple devices have this IP
			if len(deviceIDs) > 1 {
				conflict := &model.Conflict{
					Type:        model.ConflictTypeDuplicateIP,
					Status:      model.ConflictStatusActive,
					Description: "IP address assigned to multiple devices",
					IPAddress:   addr.IP,
					DeviceIDs:   deviceIDs,
					DeviceNames: deviceNames,
				}
				s.conflictService.store.CreateConflict(ctx, conflict)
			}
		}
	}
}

// validateStatus validates the device status
func validateStatus(status model.DeviceStatus) error {
	if status != "" && !status.IsValid() {
		return ValidationErrors{{Field: "status", Message: "Invalid status. Must be one of: planned, active, maintenance, decommissioned"}}
	}
	return nil
}

// setStatusChangedBy sets the StatusChangedBy field from the context
func setStatusChangedBy(ctx context.Context, device *model.Device) {
	caller := CallerFrom(ctx)
	if caller != nil && caller.UserID != "" {
		device.StatusChangedBy = caller.UserID
	}
}

func (s *DeviceService) List(ctx context.Context, filter *model.DeviceFilter) ([]model.Device, error) {
	if err := requirePermission(ctx, s.store, "devices", "list"); err != nil {
		return nil, err
	}
	return s.store.ListDevices(filter)
}

func (s *DeviceService) Create(ctx context.Context, device *model.Device) error {
	if err := requirePermission(ctx, s.store, "devices", "create"); err != nil {
		return err
	}

	if device.Name == "" {
		return ValidationErrors{{Field: "name", Message: "Name is required"}}
	}

	// Validate status
	if err := validateStatus(device.Status); err != nil {
		return err
	}

	// Set status changed by from context
	setStatusChangedBy(ctx, device)

	err := s.store.CreateDevice(enrichAuditCtx(ctx), device)
	if err != nil {
		return err
	}

	// Check for IP conflicts after creation
	s.checkForIPConflicts(ctx, device)

	return nil
}

func (s *DeviceService) Get(ctx context.Context, id string) (*model.Device, error) {
	if err := requirePermission(ctx, s.store, "devices", "read"); err != nil {
		return nil, err
	}

	device, err := s.store.GetDevice(id)
	if err != nil {
		if errors.Is(err, storage.ErrDeviceNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return device, nil
}

func (s *DeviceService) Update(ctx context.Context, device *model.Device) error {
	if err := requirePermission(ctx, s.store, "devices", "update"); err != nil {
		return err
	}

	if device.ID == "" {
		return ValidationErrors{{Field: "id", Message: "ID is required"}}
	}

	if device.Name == "" {
		return ValidationErrors{{Field: "name", Message: "Name is required"}}
	}

	// Validate status
	if err := validateStatus(device.Status); err != nil {
		return err
	}

	// Set status changed by from context
	setStatusChangedBy(ctx, device)

	err := s.store.UpdateDevice(enrichAuditCtx(ctx), device)
	if err != nil {
		return err
	}

	// Check for IP conflicts after update
	s.checkForIPConflicts(ctx, device)

	return nil
}

func (s *DeviceService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "devices", "delete"); err != nil {
		return err
	}

	if err := s.store.DeleteDevice(enrichAuditCtx(ctx), id); err != nil {
		if errors.Is(err, storage.ErrDeviceNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *DeviceService) Search(ctx context.Context, query string) ([]model.Device, error) {
	if err := requirePermission(ctx, s.store, "devices", "search"); err != nil {
		return nil, err
	}

	return s.store.SearchDevices(query)
}

// GetStatusCounts returns the count of devices by status
func (s *DeviceService) GetStatusCounts(ctx context.Context) (map[model.DeviceStatus]int, error) {
	if err := requirePermission(ctx, s.store, "devices", "list"); err != nil {
		return nil, err
	}

	return s.store.GetDeviceStatusCounts()
}
