package service

import (
	"context"
	"errors"
	"fmt"
	"net"
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
	// All zones: a PTR goes to its reverse zone whether or not that zone
	// auto-syncs. Forward records only come from auto-sync zones.
	allZones, err := s.store.ListDNSZones(ctx, &model.DNSZoneFilter{})
	if err != nil {
		return err
	}
	var zones []model.DNSZone
	for _, z := range allZones {
		if z.AutoSync {
			zones = append(zones, z)
		}
	}

	// For each zone linked to device's network
	for _, zone := range zones {
		if zone.NetworkID == nil || !deviceHasNetworkID(device, *zone.NetworkID) {
			continue
		}
		fqdn := device.Hostname + "." + zone.Name
		for _, addr := range device.Addresses {
			if addr.NetworkID != *zone.NetworkID || addr.IP == "" {
				continue
			}
			rtype := addressRecordType(addr.IP)
			if rtype == "" {
				log.Warn("Skipping DNS record for unparseable address", "device_id", device.ID, "ip", addr.IP)
				continue
			}
			req := &model.CreateDNSRecordRequest{
				ZoneID:   zone.ID,
				DeviceID: &device.ID,
				Name:     device.Hostname,
				Type:     rtype,
				Value:    addr.IP,
				TTL:      zone.TTL,
			}
			if _, err := s.dns.CreateRecord(ctx, req); err != nil {
				// Individual record failures shouldn't block the rest
				log.Warn("Failed to create DNS record", "device_id", device.ID, "zone", zone.Name, "type", rtype, "error", err)
				continue
			}

			if zone.CreatePTR && zone.PTRZone != nil && *zone.PTRZone != "" {
				s.createPTRRecord(ctx, device, allZones, *zone.PTRZone, addr.IP, fqdn, zone.TTL)
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

// createPTRRecord adds the reverse record for ip to the rackd zone named
// ptrZoneName. The PTR lives in that reverse zone, not the forward zone, and
// is synced by it, so the reverse zone has to exist in rackd as well.
func (s *DeviceService) createPTRRecord(ctx context.Context, device *model.Device, zones []model.DNSZone, ptrZoneName, ip, target string, ttl int) {
	ptrZone := findZoneByName(zones, ptrZoneName)
	if ptrZone == nil {
		log.Warn("PTR zone is not a zone in rackd; skipping PTR record", "device_id", device.ID, "ptr_zone", ptrZoneName)
		return
	}
	name, ok := ptrRecordName(ip, ptrZone.Name)
	if !ok {
		log.Warn("Address is outside the PTR zone; skipping PTR record", "device_id", device.ID, "ip", ip, "ptr_zone", ptrZone.Name)
		return
	}
	req := &model.CreateDNSRecordRequest{
		ZoneID:   ptrZone.ID,
		DeviceID: &device.ID,
		Name:     name,
		Type:     "PTR",
		Value:    target,
		TTL:      ttl,
	}
	if _, err := s.dns.CreateRecord(ctx, req); err != nil {
		log.Warn("Failed to create PTR record", "device_id", device.ID, "ptr_zone", ptrZone.Name, "error", err)
	}
}

// addressRecordType returns "A" for an IPv4 address, "AAAA" for IPv6 and ""
// for anything that doesn't parse.
func addressRecordType(ipStr string) string {
	ip := net.ParseIP(ipStr)
	switch {
	case ip == nil:
		return ""
	case ip.To4() != nil:
		return "A"
	default:
		return "AAAA"
	}
}

// reverseDNSName returns the full reverse lookup name for an address:
// 192.168.1.10 -> 10.1.168.192.in-addr.arpa, and the 32-nibble ip6.arpa
// form for IPv6.
func reverseDNSName(ipStr string) (string, bool) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", false
	}
	if v4 := ip.To4(); v4 != nil {
		return fmt.Sprintf("%d.%d.%d.%d.in-addr.arpa", v4[3], v4[2], v4[1], v4[0]), true
	}
	const hexDigits = "0123456789abcdef"
	var b strings.Builder
	for i := len(ip) - 1; i >= 0; i-- {
		b.WriteByte(hexDigits[ip[i]&0x0f])
		b.WriteByte('.')
		b.WriteByte(hexDigits[ip[i]>>4])
		b.WriteByte('.')
	}
	b.WriteString("ip6.arpa")
	return b.String(), true
}

// ptrRecordName returns the PTR record name for ip relative to the reverse
// zone (e.g. "10" for 192.168.1.10 in 1.168.192.in-addr.arpa), and false if
// the address doesn't fall inside that zone.
func ptrRecordName(ip, zoneName string) (string, bool) {
	full, ok := reverseDNSName(ip)
	if !ok {
		return "", false
	}
	suffix := "." + strings.ToLower(strings.TrimSuffix(zoneName, "."))
	if !strings.HasSuffix(full, suffix) {
		return "", false
	}
	return strings.TrimSuffix(full, suffix), true
}

// findZoneByName returns the zone with the given name, ignoring case and a
// trailing dot, or nil.
func findZoneByName(zones []model.DNSZone, name string) *model.DNSZone {
	name = strings.TrimSuffix(name, ".")
	for i := range zones {
		if strings.EqualFold(strings.TrimSuffix(zones[i].Name, "."), name) {
			return &zones[i]
		}
	}
	return nil
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
