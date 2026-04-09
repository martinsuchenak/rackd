package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	dnspkg "github.com/martinsuchenak/rackd/internal/dns"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
	"pgregory.net/rapid"
)

// stubStorage implements the subset of storage.ExtendedStorage needed by
// syncDeviceDNS → DNSService.CreateRecord. Methods not exercised by these
// tests panic so that unexpected calls are caught immediately.
type stubStorage struct {
	storage.ExtendedStorage // embed to satisfy interface; unused methods will panic via nil pointer

	zones          []model.DNSZone
	createdRecords []model.DNSRecord
	devices        map[string]*model.Device
}

func newStubStorage(zones []model.DNSZone) *stubStorage {
	return &stubStorage{
		zones:   zones,
		devices: make(map[string]*model.Device),
	}
}

func (s *stubStorage) ListDNSZones(_ context.Context, filter *model.DNSZoneFilter) ([]model.DNSZone, error) {
	var out []model.DNSZone
	for _, z := range s.zones {
		if filter.AutoSync != nil && *filter.AutoSync != z.AutoSync {
			continue
		}
		out = append(out, z)
	}
	return out, nil
}

func (s *stubStorage) GetDNSZone(_ context.Context, id string) (*model.DNSZone, error) {
	for i := range s.zones {
		if s.zones[i].ID == id {
			return &s.zones[i], nil
		}
	}
	return nil, storage.ErrDNSZoneNotFound
}

func (s *stubStorage) GetDevice(_ context.Context, id string) (*model.Device, error) {
	if d, ok := s.devices[id]; ok {
		return d, nil
	}
	return nil, storage.ErrDeviceNotFound
}

func (s *stubStorage) GetDNSRecordByName(_ context.Context, zoneID, name string, recordType string) (*model.DNSRecord, error) {
	return nil, storage.ErrDNSRecordNotFound
}

func (s *stubStorage) CreateDNSRecord(_ context.Context, record *model.DNSRecord) error {
	s.createdRecords = append(s.createdRecords, *record)
	return nil
}

func (s *stubStorage) UpdateDNSRecord(_ context.Context, _ *model.DNSRecord) error {
	return nil
}

func (s *stubStorage) GetDNSProvider(_ context.Context, id string) (*model.DNSProviderConfig, error) {
	return nil, storage.ErrDNSProviderNotFound
}

func (s *stubStorage) HasPermission(_ context.Context, _, _, _ string) (bool, error) {
	return true, nil
}

func (s *stubStorage) DB() *sql.DB  { return nil }
func (s *stubStorage) Close() error { return nil }

// buildTestServices creates a DeviceService wired to a DNSService backed by
// the given stubStorage.
func buildTestServices(ss *stubStorage) *DeviceService {
	dnsService := &DNSService{
		store:         ss,
		providerCache: make(map[string]dnspkg.Provider),
	}
	ds := &DeviceService{
		store: ss,
		dns:   dnsService,
	}
	return ds
}

// systemCtx returns a context that bypasses permission checks.
func systemCtx() context.Context {
	return SystemContext(context.Background(), "test")
}

// --- Generators ---

// genHostname generates a valid hostname label (1-10 lowercase alpha chars).
func genHostname(t *rapid.T) string {
	n := rapid.IntRange(1, 10).Draw(t, "hostnameLen")
	chars := make([]byte, n)
	for i := range chars {
		chars[i] = byte(rapid.IntRange('a', 'z').Draw(t, "char"))
	}
	return string(chars)
}

// genLabel generates a DNS label (1-10 lowercase alpha chars).
func genLabel(t *rapid.T) string {
	n := rapid.IntRange(1, 10).Draw(t, "labelLen")
	chars := make([]byte, n)
	for i := range chars {
		chars[i] = byte(rapid.IntRange('a', 'z').Draw(t, "char"))
	}
	return string(chars)
}

// genZone generates a zone name like "example.com" (2-3 labels).
func genZone(t *rapid.T) string {
	numLabels := rapid.IntRange(2, 3).Draw(t, "numLabels")
	labels := make([]string, numLabels)
	for i := range labels {
		labels[i] = genLabel(t)
	}
	return strings.Join(labels, ".")
}

// Feature: dns-device-linking, Property 3: CNAME sync produces correct records for matching domains
// Validates: Requirements 3.1, 3.2, 3.3, 3.5
func TestSyncDeviceDNS_CNAMECorrectRecords(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a zone
		zoneName := genZone(t)
		zone := model.DNSZone{
			ID:       "zone-1",
			Name:     zoneName,
			AutoSync: true,
			TTL:      300,
		}

		// Generate a device with a hostname and 1-3 domains that match the zone
		hostname := genHostname(t)
		numDomains := rapid.IntRange(1, 3).Draw(t, "numDomains")
		var domains []string
		var expectedPrefixes []string
		for i := 0; i < numDomains; i++ {
			prefix := genLabel(t)
			domain := prefix + "." + zoneName
			domains = append(domains, domain)
			expectedPrefixes = append(expectedPrefixes, prefix)
		}

		device := &model.Device{
			ID:       "dev-1",
			Name:     "test-device",
			Hostname: hostname,
			Domains:  domains,
		}

		ss := newStubStorage([]model.DNSZone{zone})
		ss.devices[device.ID] = device
		ds := buildTestServices(ss)

		err := ds.syncDeviceDNS(systemCtx(), device)
		if err != nil {
			t.Fatalf("syncDeviceDNS returned error: %v", err)
		}

		// Filter only CNAME records from created records
		var cnameRecords []model.DNSRecord
		for _, r := range ss.createdRecords {
			if r.Type == "CNAME" {
				cnameRecords = append(cnameRecords, r)
			}
		}

		// Should have exactly one CNAME per domain
		if len(cnameRecords) != numDomains {
			t.Fatalf("expected %d CNAME records, got %d", numDomains, len(cnameRecords))
		}

		// Verify each CNAME record
		for i, rec := range cnameRecords {
			expectedName := expectedPrefixes[i]
			expectedValue := hostname + "." + zoneName

			if rec.Name != expectedName {
				t.Errorf("CNAME[%d] Name: expected %q, got %q", i, expectedName, rec.Name)
			}
			if rec.Value != expectedValue {
				t.Errorf("CNAME[%d] Value: expected %q, got %q", i, expectedValue, rec.Value)
			}
			if rec.Type != "CNAME" {
				t.Errorf("CNAME[%d] Type: expected CNAME, got %q", i, rec.Type)
			}
			if rec.DeviceID == nil || *rec.DeviceID != device.ID {
				t.Errorf("CNAME[%d] DeviceID: expected %q, got %v", i, device.ID, rec.DeviceID)
			}
			if rec.ZoneID != zone.ID {
				t.Errorf("CNAME[%d] ZoneID: expected %q, got %q", i, zone.ID, rec.ZoneID)
			}
		}
	})
}

// Feature: dns-device-linking, Property 4: No CNAME sync without hostname
// Validates: Requirements 3.3
func TestSyncDeviceDNS_NoCNAMEWithoutHostname(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a zone
		zoneName := genZone(t)
		zone := model.DNSZone{
			ID:       "zone-1",
			Name:     zoneName,
			AutoSync: true,
			TTL:      300,
		}

		// Generate domains that would match the zone
		numDomains := rapid.IntRange(0, 5).Draw(t, "numDomains")
		var domains []string
		for i := 0; i < numDomains; i++ {
			prefix := genLabel(t)
			domains = append(domains, prefix+"."+zoneName)
		}

		// Device with empty hostname
		device := &model.Device{
			ID:      "dev-1",
			Name:    "test-device",
			Domains: domains,
			// Hostname is intentionally empty
		}

		ss := newStubStorage([]model.DNSZone{zone})
		ds := buildTestServices(ss)

		err := ds.syncDeviceDNS(systemCtx(), device)
		if err != nil {
			t.Fatalf("syncDeviceDNS returned error: %v", err)
		}

		// No records should be created at all (not just CNAMEs)
		if len(ss.createdRecords) != 0 {
			t.Fatalf("expected 0 records for device without hostname, got %d", len(ss.createdRecords))
		}
	})
}

func TestDeviceService_CreateValidatesStatusAndSetsStatusChangedBy(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "devices", "create", true)
	svc := NewDeviceService(store)

	err := svc.Create(userContext("user-1"), &model.Device{
		ID:     "dev-1",
		Name:   "switch-1",
		Status: model.DeviceStatus("broken"),
	})
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for invalid status, got %v", err)
	}

	device := &model.Device{
		ID:     "dev-2",
		Name:   "switch-2",
		Status: model.DeviceStatusActive,
	}
	if err := svc.Create(userContext("user-1"), device); err != nil {
		t.Fatalf("expected valid device create to succeed, got %v", err)
	}
	if store.deviceCreated == nil || store.deviceCreated.StatusChangedBy != "user-1" {
		t.Fatalf("expected status_changed_by to be set from caller, got %#v", store.deviceCreated)
	}
}

func TestDeviceService_SearchUsesListPermission(t *testing.T) {
	store := newServiceTestStorage()
	store.devices["dev-1"] = &model.Device{ID: "dev-1", Name: "router-1"}
	store.setPermission("user-1", "devices", "list", true)
	svc := NewDeviceService(store)

	results, err := svc.Search(userContext("user-1"), "router-1")
	if err != nil {
		t.Fatalf("Search returned unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].ID != "dev-1" {
		t.Fatalf("expected search result for router-1, got %#v", results)
	}
}

func TestDeviceService_DeleteMapsMissingDeviceToNotFound(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "devices", "delete", true)
	svc := NewDeviceService(store)

	err := svc.Delete(userContext("user-1"), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestValidateStatusAndExtractPTRNameHelpers(t *testing.T) {
	if err := validateStatus(model.DeviceStatus("invalid")); err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for invalid device status, got %v", err)
	}
	if err := validateStatus(model.DeviceStatusActive); err != nil {
		t.Fatalf("expected active status to be valid, got %v", err)
	}
	if ptr := extractPTRName("10.20.30.40"); ptr != "40.30.20.in-addr.arpa" {
		t.Fatalf("unexpected PTR name %q", ptr)
	}
	if ptr := extractPTRName("bad-ip"); ptr != "" {
		t.Fatalf("expected invalid IP PTR extraction to return empty string, got %q", ptr)
	}
}
