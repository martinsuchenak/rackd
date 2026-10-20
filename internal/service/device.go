package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

type DeviceService struct {
	store           storage.ExtendedStorage
	conflictService *ConflictService
	dns             *DNSService
}

func NewDeviceService(store storage.ExtendedStorage) *DeviceService {
	return &DeviceService{store: store}
}

func (s *DeviceService) setDNSService(dns *DNSService) {
	s.dns = dns
}

// claimReservations marks the reservation for each address the device holds
// as claimed: the address is now in use, not just set aside. That includes
// addresses bound to no pool, which the web UI's device form creates. A
// failure doesn't undo the device write.
func (s *DeviceService) claimReservations(ctx context.Context, device *model.Device) {
	for _, addr := range device.Addresses {
		if addr.IP == "" {
			continue
		}
		if _, err := s.store.ClaimReservation(enrichAuditCtx(ctx), addr); err != nil {
			log.Warn("Failed to claim reservation", "device_id", device.ID, "ip", addr.IP, "error", err)
		}
	}
}

// boolPtr returns a pointer to the given bool value
func boolPtr(v bool) *bool {
	return &v
}

// syncDeviceDNSAsync runs syncDeviceDNS in a background goroutine so a slow
// or unresponsive DNS provider cannot block device create/update (or run
// into the global request timeout). The local DNS record is still created
// within milliseconds; provider-side failures land on the record's sync
// status and in the server log.
func (s *DeviceService) syncDeviceDNSAsync(ctx context.Context, device *model.Device) {
	if s.dns == nil {
		return
	}
	snapshot := *device
	go func() {
		bgCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Minute)
		defer cancel()
		if err := s.syncDeviceDNS(bgCtx, &snapshot); err != nil {
			log.Warn("DNS sync failed after device change", "device_id", snapshot.ID, "error", err)
		}
	}()
}

// syncDeviceDNS creates/updates DNS records for a device when it has a hostname
func (s *DeviceService) syncDeviceDNS(ctx context.Context, device *model.Device) error {
	if s.dns == nil {
		return nil
	}
	if device.Hostname == "" {
		return nil
	}

	// Find zones for this device's networks with auto_sync enabled
	zones, err := s.store.ListDNSZones(ctx, &model.DNSZoneFilter{
		AutoSync: boolPtr(true),
	})
	if err != nil {
		return err
	}

	// For each zone linked to device's network
	for _, zone := range zones {
		if zone.NetworkID != nil && deviceHasNetworkID(device, *zone.NetworkID) {
			// Create/update A record for each IP in this network
			for _, addr := range device.Addresses {
				if addr.NetworkID == *zone.NetworkID && addr.IP != "" {
					req := &model.CreateDNSRecordRequest{
						ZoneID:   zone.ID,
						DeviceID: &device.ID,
						Name:     device.Hostname,
						Type:     "A",
						Value:    addr.IP,
						TTL:      zone.TTL,
					}
					if _, err := s.dns.CreateRecord(ctx, req); err != nil {
						// Log but don't fail - individual record failures shouldn't block
						continue
					}

					// Create PTR record if enabled
					if zone.CreatePTR && zone.PTRZone != nil && *zone.PTRZone != "" {
						ptrReq := &model.CreateDNSRecordRequest{
							ZoneID:   zone.ID,
							DeviceID: &device.ID,
							Name:     extractPTRName(addr.IP),
							Type:     "PTR",
							Value:    device.Hostname + "." + zone.Name,
							TTL:      zone.TTL,
						}
						if _, err := s.dns.CreateRecord(ctx, ptrReq); err != nil {
							// Log but don't fail - individual record failures shouldn't block
							continue
						}
					}
				}
			}
		}
	}

	// Create CNAME records for each device domain that matches a zone
	for _, domain := range device.Domains {
		zone, prefix := MatchZoneForDomain(domain, zones)
		if zone == nil {
			continue // no matching zone
		}
		cnameValue := device.Hostname + "." + zone.Name
		req := &model.CreateDNSRecordRequest{
			ZoneID:   zone.ID,
			DeviceID: &device.ID,
			Name:     prefix,
			Type:     "CNAME",
			Value:    cnameValue,
			TTL:      zone.TTL,
		}
		if _, err := s.dns.CreateRecord(ctx, req); err != nil {
			// Log but don't fail - individual record failures shouldn't block
			continue
		}
	}

	return nil
}

// deviceHasNetworkID checks if a device has an address in the specified network
func deviceHasNetworkID(device *model.Device, networkID string) bool {
	for _, addr := range device.Addresses {
		if addr.NetworkID == networkID {
			return true
		}
	}
	return false
}

// extractPTRName extracts a PTR record name from an IP address
func extractPTRName(ipStr string) string {
	parts := strings.Split(ipStr, ".")
	if len(parts) != 4 {
		return ""
	}
	// Reverse the IP for in-addr.arpa format
	return fmt.Sprintf("%s.%s.%s.in-addr.arpa", parts[3], parts[2], parts[1])
}

// checkForIPConflicts checks if any IP addresses are duplicates and creates conflict records
func (s *DeviceService) checkForIPConflicts(ctx context.Context, device *model.Device) {
	if s.conflictService == nil {
		return
	}

	checked := map[string]bool{}
	for _, addr := range device.Addresses {
		if addr.IP == "" || checked[addr.IP] {
			continue
		}
		checked[addr.IP] = true

		// An IP with an active or ignored conflict is left to that record,
		// so ignored conflicts stay ignored. A resolved one was dealt with:
		// a duplicate now is new.
		conflicts, err := s.store.GetConflictsByIP(ctx, addr.IP)
		if err != nil {
			log.Warn("Duplicate-IP check failed", "ip", addr.IP, "error", err)
			continue
		}
		if hasOpenConflict(conflicts) {
			continue
		}

		holders, err := s.store.DevicesWithIP(ctx, addr.IP)
		if err != nil {
			log.Warn("Duplicate-IP check failed", "ip", addr.IP, "error", err)
			continue
		}
		if len(holders) < 2 {
			continue
		}
		conflict := &model.Conflict{
			Type:        model.ConflictTypeDuplicateIP,
			Status:      model.ConflictStatusActive,
			Description: "IP address assigned to multiple devices",
			IPAddress:   addr.IP,
		}
		for _, d := range holders {
			conflict.DeviceIDs = append(conflict.DeviceIDs, d.ID)
			conflict.DeviceNames = append(conflict.DeviceNames, d.Name)
		}
		if err := s.conflictService.store.CreateConflict(ctx, conflict); err != nil {
			// A concurrent write recorded it first.
			if !errors.Is(err, storage.ErrConflictRecorded) {
				log.Warn("Failed to record duplicate-IP conflict", "ip", addr.IP, "error", err)
			}
			continue
		}
		publishEvent(model.EventTypeConflictDetected, conflict)
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
	return s.store.ListDevices(ctx, filter)
}

func (s *DeviceService) Create(ctx context.Context, device *model.Device) error {
	if err := requirePermission(ctx, s.store, "devices", "create"); err != nil {
		return err
	}

	if errs := ValidateDevice(device); len(errs) > 0 {
		return errs
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
	publishEvent(model.EventTypeDeviceCreated, deviceSnapshot(device))

	// The device now holds its addresses: claim their reservations, then
	// check for IP conflicts.
	s.claimReservations(ctx, device)
	s.checkForIPConflicts(ctx, device)

	// Sync DNS records if device has a hostname. Run in the background:
	// provider sync can block for the full HTTP timeout (~30s) against an
	// unresponsive DNS provider, which would otherwise hold the request open
	// until the global request timeout aborts it. Local record creation
	// happens within milliseconds; provider failures are recorded on the
	// record's sync status.
	s.syncDeviceDNSAsync(ctx, device)

	return nil
}

func (s *DeviceService) Get(ctx context.Context, id string) (*model.Device, error) {
	if err := requirePermission(ctx, s.store, "devices", "read"); err != nil {
		return nil, err
	}

	device, err := s.store.GetDevice(ctx, id)
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

	if errs := ValidateDevice(device); len(errs) > 0 {
		return errs
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
	publishEvent(model.EventTypeDeviceUpdated, deviceSnapshot(device))

	// Claim reservations for the device's addresses, then check for IP
	// conflicts.
	s.claimReservations(ctx, device)
	s.checkForIPConflicts(ctx, device)

	s.syncDeviceDNSAsync(ctx, device)

	return nil
}

func (s *DeviceService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "devices", "delete"); err != nil {
		return err
	}

	deleted := s.deletedDevice(ctx, id)
	if err := s.store.DeleteDevice(enrichAuditCtx(ctx), id); err != nil {
		if errors.Is(err, storage.ErrDeviceNotFound) {
			return ErrNotFound
		}
		return err
	}
	publishEvent(model.EventTypeDeviceDeleted, deleted)
	return nil
}

func (s *DeviceService) Search(ctx context.Context, query string) ([]model.Device, error) {
	if err := requirePermission(ctx, s.store, "devices", "list"); err != nil {
		return nil, err
	}

	return s.store.SearchDevices(ctx, query)
}

// GetStatusCounts returns the count of devices by status
func (s *DeviceService) GetStatusCounts(ctx context.Context) (map[model.DeviceStatus]int, error) {
	if err := requirePermission(ctx, s.store, "devices", "list"); err != nil {
		return nil, err
	}

	return s.store.GetDeviceStatusCounts(ctx)
}
