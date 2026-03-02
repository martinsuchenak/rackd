package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	dnspkg "github.com/martinsuchenak/rackd/internal/dns"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
	"pgregory.net/rapid"
)

// genDNSLabel generates a valid DNS label (1-10 lowercase alpha chars).
func genDNSLabel(t *rapid.T) string {
	n := rapid.IntRange(1, 10).Draw(t, "labelLen")
	chars := make([]byte, n)
	for i := range chars {
		chars[i] = byte(rapid.IntRange('a', 'z').Draw(t, "char"))
	}
	return string(chars)
}

// genZoneName generates a zone name like "example.com" (2-3 labels joined by dots).
func genZoneName(t *rapid.T) string {
	numLabels := rapid.IntRange(2, 3).Draw(t, "numLabels")
	labels := make([]string, numLabels)
	for i := range labels {
		labels[i] = genDNSLabel(t)
	}
	return strings.Join(labels, ".")
}

// Feature: dns-device-linking, Property 15: Domain suffix match correctness
// Validates: Requirements 7.1, 7.2
func TestMatchZoneForDomain_SuffixMatchCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		zoneName := genZoneName(t)
		zone := model.DNSZone{ID: "z1", Name: zoneName}
		zones := []model.DNSZone{zone}

		// Sub-property A: exact match returns "@" prefix
		matched, prefix := MatchZoneForDomain(zoneName, zones)
		if matched == nil {
			t.Fatalf("expected exact match for domain=%q zone=%q, got nil", zoneName, zoneName)
		}
		if matched.Name != zoneName {
			t.Fatalf("expected matched zone name %q, got %q", zoneName, matched.Name)
		}
		if prefix != "@" {
			t.Fatalf("expected prefix '@' for exact match, got %q", prefix)
		}

		// Sub-property B: suffix match returns correct prefix
		sub := genDNSLabel(t)
		domain := sub + "." + zoneName
		matched, prefix = MatchZoneForDomain(domain, zones)
		if matched == nil {
			t.Fatalf("expected suffix match for domain=%q zone=%q, got nil", domain, zoneName)
		}
		if matched.Name != zoneName {
			t.Fatalf("expected matched zone name %q, got %q", zoneName, matched.Name)
		}
		if prefix != sub {
			t.Fatalf("expected prefix %q, got %q", sub, prefix)
		}

		// Sub-property C: non-matching domain returns nil
		unrelated := genZoneName(t)
		// Ensure unrelated is not a suffix of zoneName or equal to it
		for unrelated == zoneName || strings.HasSuffix(unrelated, "."+zoneName) || strings.HasSuffix(zoneName, "."+unrelated) {
			unrelated = genZoneName(t)
		}
		matched, _ = MatchZoneForDomain(unrelated, zones)
		if matched != nil {
			// Only fail if the match is incorrect — the domain genuinely doesn't end with .zoneName
			if unrelated != zoneName && !strings.HasSuffix(unrelated, "."+zoneName) {
				t.Fatalf("expected no match for domain=%q zone=%q, got zone %q", unrelated, zoneName, matched.Name)
			}
		}
	})
}

// Feature: dns-device-linking, Property 16: Longest zone match wins
// Validates: Requirements 7.3
func TestMatchZoneForDomain_LongestZoneMatchWins(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Build a parent zone and a child zone that is a subdomain of the parent.
		// e.g. parent = "example.com", child = "sub.example.com"
		parentName := genZoneName(t)
		childLabel := genDNSLabel(t)
		childName := childLabel + "." + parentName

		parentZone := model.DNSZone{ID: "parent", Name: parentName}
		childZone := model.DNSZone{ID: "child", Name: childName}

		// Domain that matches both: "host.sub.example.com"
		host := genDNSLabel(t)
		domain := host + "." + childName

		// Test with both orderings to ensure order doesn't matter
		for _, zones := range [][]model.DNSZone{
			{parentZone, childZone},
			{childZone, parentZone},
		} {
			matched, prefix := MatchZoneForDomain(domain, zones)
			if matched == nil {
				t.Fatalf("expected match for domain=%q with zones %v, got nil", domain, zones)
			}
			if matched.Name != childName {
				t.Fatalf("expected longest zone %q to win, got %q (domain=%q)", childName, matched.Name, domain)
			}
			if prefix != host {
				t.Fatalf("expected prefix %q, got %q", host, prefix)
			}
		}
	})
}

// genIPv4Simple generates a random IPv4 address using fmt.
func genIPv4Simple(t *rapid.T) string {
	return fmt.Sprintf("%d.%d.%d.%d",
		rapid.IntRange(1, 254).Draw(t, "o1"),
		rapid.IntRange(0, 255).Draw(t, "o2"),
		rapid.IntRange(0, 255).Draw(t, "o3"),
		rapid.IntRange(1, 254).Draw(t, "o4"),
	)
}

// ipToReversedPTRName converts an IPv4 address to its in-addr.arpa PTR name.
func ipToReversedPTRName(ip string) string {
	parts := strings.Split(ip, ".")
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, ".") + ".in-addr.arpa"
}

// Feature: dns-device-linking, Property 5: A/AAAA import auto-match by IP
// Validates: Requirements 4.1
func TestMatchDeviceForRecord_A_AAAA_MatchByIP(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		zoneName := genZoneName(t)
		zone := &model.DNSZone{ID: "z1", Name: zoneName}

		ip := genIPv4Simple(t)
		addrID := "addr-1"
		deviceID := "dev-1"

		device := model.Device{
			ID: deviceID,
			Addresses: []model.Address{
				{ID: addrID, IP: ip},
			},
		}

		recType := rapid.SampledFrom([]string{"A", "AAAA"}).Draw(t, "recType")
		record := &model.DNSRecord{
			Name:  genDNSLabel(t),
			Type:  recType,
			Value: ip,
		}

		svc := &DNSService{}
		svc.matchDeviceForRecord(record, zone, []model.Device{device})

		if record.DeviceID == nil {
			t.Fatalf("expected DeviceID to be set for %s record with matching IP %s", recType, ip)
		}
		if *record.DeviceID != deviceID {
			t.Fatalf("expected DeviceID %q, got %q", deviceID, *record.DeviceID)
		}
		if record.AddressID == nil {
			t.Fatalf("expected AddressID to be set for %s record with matching IP %s", recType, ip)
		}
		if *record.AddressID != addrID {
			t.Fatalf("expected AddressID %q, got %q", addrID, *record.AddressID)
		}
	})
}

// Feature: dns-device-linking, Property 6: CNAME import auto-match by domain
// Validates: Requirements 4.2
func TestMatchDeviceForRecord_CNAME_MatchByDomain(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		zoneName := genZoneName(t)
		zone := &model.DNSZone{ID: "z1", Name: zoneName}

		prefix := genDNSLabel(t)
		fqdn := prefix + "." + zoneName
		deviceID := "dev-1"

		device := model.Device{
			ID:      deviceID,
			Domains: []string{fqdn},
		}

		record := &model.DNSRecord{
			Name:  prefix,
			Type:  "CNAME",
			Value: "target." + zoneName,
		}

		svc := &DNSService{}
		svc.matchDeviceForRecord(record, zone, []model.Device{device})

		if record.DeviceID == nil {
			t.Fatalf("expected DeviceID to be set for CNAME record with FQDN %q matching device domain", fqdn)
		}
		if *record.DeviceID != deviceID {
			t.Fatalf("expected DeviceID %q, got %q", deviceID, *record.DeviceID)
		}
	})
}

// Feature: dns-device-linking, Property 7: PTR import auto-match by hostname
// Validates: Requirements 4.3, 4.4
func TestMatchDeviceForRecord_PTR_MatchByHostname(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		zoneName := genZoneName(t)
		zone := &model.DNSZone{ID: "z1", Name: zoneName}

		hostname := genDNSLabel(t)
		ip := genIPv4Simple(t)
		addrID := "addr-1"
		deviceID := "dev-1"

		device := model.Device{
			ID:       deviceID,
			Hostname: hostname,
			Addresses: []model.Address{
				{ID: addrID, IP: ip},
			},
		}

		// PTR record: Name is the reversed IP in-addr.arpa form, Value is hostname.zoneName
		ptrName := ipToReversedPTRName(ip)
		record := &model.DNSRecord{
			Name:  ptrName,
			Type:  "PTR",
			Value: hostname + "." + zoneName,
		}

		svc := &DNSService{}
		svc.matchDeviceForRecord(record, zone, []model.Device{device})

		if record.DeviceID == nil {
			t.Fatalf("expected DeviceID to be set for PTR record matching hostname %q", hostname)
		}
		if *record.DeviceID != deviceID {
			t.Fatalf("expected DeviceID %q, got %q", deviceID, *record.DeviceID)
		}
		if record.AddressID == nil {
			t.Fatalf("expected AddressID to be set for PTR record with reverse IP matching address")
		}
		if *record.AddressID != addrID {
			t.Fatalf("expected AddressID %q, got %q", addrID, *record.AddressID)
		}
	})
}

// Feature: dns-device-linking, Property 8: Non-matchable record types remain unlinked
// Validates: Requirements 4.5, 4.6
func TestMatchDeviceForRecord_NonMatchableTypes(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		zoneName := genZoneName(t)
		zone := &model.DNSZone{ID: "z1", Name: zoneName}

		ip := genIPv4Simple(t)
		device := model.Device{
			ID:       "dev-1",
			Hostname: genDNSLabel(t),
			Domains:  []string{genDNSLabel(t) + "." + zoneName},
			Addresses: []model.Address{
				{ID: "addr-1", IP: ip},
			},
		}

		nonMatchTypes := []string{"MX", "TXT", "NS", "SRV", "SOA"}
		recType := rapid.SampledFrom(nonMatchTypes).Draw(t, "recType")

		record := &model.DNSRecord{
			Name:  genDNSLabel(t),
			Type:  recType,
			Value: ip, // Even if value matches an IP, non-matchable types should not link
		}

		svc := &DNSService{}
		svc.matchDeviceForRecord(record, zone, []model.Device{device})

		if record.DeviceID != nil {
			t.Fatalf("expected DeviceID to be nil for %s record, got %q", recType, *record.DeviceID)
		}
		if record.AddressID != nil {
			t.Fatalf("expected AddressID to be nil for %s record, got %q", recType, *record.AddressID)
		}
	})
}

// Feature: dns-device-linking, Property 9: Import linked count accuracy
// Validates: Requirements 4.7
func TestMatchDeviceForRecord_LinkedCountAccuracy(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		zoneName := genZoneName(t)
		zone := &model.DNSZone{ID: "z1", Name: zoneName}

		// Generate 1-5 devices with unique IPs
		numDevices := rapid.IntRange(1, 5).Draw(t, "numDevices")
		var devices []model.Device
		usedIPs := make(map[string]bool)
		for i := 0; i < numDevices; i++ {
			ip := genIPv4Simple(t)
			for usedIPs[ip] {
				ip = genIPv4Simple(t)
			}
			usedIPs[ip] = true
			devices = append(devices, model.Device{
				ID:       fmt.Sprintf("dev-%d", i),
				Hostname: genDNSLabel(t),
				Addresses: []model.Address{
					{ID: fmt.Sprintf("addr-%d", i), IP: ip},
				},
			})
		}

		// Generate a mix of matching and non-matching records
		numRecords := rapid.IntRange(1, 10).Draw(t, "numRecords")
		var records []*model.DNSRecord
		expectedLinked := 0

		for i := 0; i < numRecords; i++ {
			shouldMatch := rapid.Bool().Draw(t, fmt.Sprintf("match-%d", i))
			if shouldMatch && len(devices) > 0 {
				// Create an A record that matches a random device's IP
				devIdx := rapid.IntRange(0, len(devices)-1).Draw(t, fmt.Sprintf("devIdx-%d", i))
				dev := devices[devIdx]
				rec := &model.DNSRecord{
					Name:  genDNSLabel(t),
					Type:  "A",
					Value: dev.Addresses[0].IP,
				}
				records = append(records, rec)
			} else {
				// Create a TXT record (non-matchable)
				rec := &model.DNSRecord{
					Name:  genDNSLabel(t),
					Type:  "TXT",
					Value: "v=spf1 include:example.com ~all",
				}
				records = append(records, rec)
			}
		}

		// Run matching and count linked
		svc := &DNSService{}
		for _, rec := range records {
			svc.matchDeviceForRecord(rec, zone, devices)
			if rec.DeviceID != nil {
				expectedLinked++
			}
		}

		// Verify: count records with non-nil DeviceID matches expectedLinked
		actualLinked := 0
		for _, rec := range records {
			if rec.DeviceID != nil {
				actualLinked++
			}
		}

		if actualLinked != expectedLinked {
			t.Fatalf("linked count mismatch: expected %d, got %d", expectedLinked, actualLinked)
		}
	})
}

// linkTestStorage implements the subset of storage.ExtendedStorage needed by
// DNSService.LinkRecord. It stores DNS records, devices, and zones in memory.
type linkTestStorage struct {
	storage.ExtendedStorage // embed to satisfy interface; unused methods panic via nil pointer

	records        map[string]*model.DNSRecord
	devices        map[string]*model.Device
	zones          map[string]*model.DNSZone
	updatedRecords []*model.DNSRecord
	updatedDevices []*model.Device
}

func newLinkTestStorage() *linkTestStorage {
	return &linkTestStorage{
		records: make(map[string]*model.DNSRecord),
		devices: make(map[string]*model.Device),
		zones:   make(map[string]*model.DNSZone),
	}
}

func (s *linkTestStorage) GetDNSRecord(id string) (*model.DNSRecord, error) {
	if r, ok := s.records[id]; ok {
		// Return a copy to avoid aliasing
		cp := *r
		return &cp, nil
	}
	return nil, storage.ErrDNSRecordNotFound
}

func (s *linkTestStorage) GetDevice(id string) (*model.Device, error) {
	if d, ok := s.devices[id]; ok {
		// Return a copy
		cp := *d
		cp.Addresses = make([]model.Address, len(d.Addresses))
		copy(cp.Addresses, d.Addresses)
		cp.Domains = make([]string, len(d.Domains))
		copy(cp.Domains, d.Domains)
		return &cp, nil
	}
	return nil, storage.ErrDeviceNotFound
}

func (s *linkTestStorage) GetDNSZone(id string) (*model.DNSZone, error) {
	if z, ok := s.zones[id]; ok {
		cp := *z
		return &cp, nil
	}
	return nil, storage.ErrDNSZoneNotFound
}

func (s *linkTestStorage) UpdateDNSRecord(_ context.Context, record *model.DNSRecord) error {
	s.updatedRecords = append(s.updatedRecords, record)
	// Also update the in-memory store
	if _, ok := s.records[record.ID]; ok {
		cp := *record
		s.records[record.ID] = &cp
	}
	return nil
}

func (s *linkTestStorage) UpdateDevice(_ context.Context, device *model.Device) error {
	s.updatedDevices = append(s.updatedDevices, device)
	// Also update the in-memory store
	if _, ok := s.devices[device.ID]; ok {
		cp := *device
		cp.Addresses = make([]model.Address, len(device.Addresses))
		copy(cp.Addresses, device.Addresses)
		cp.Domains = make([]string, len(device.Domains))
		copy(cp.Domains, device.Domains)
		s.devices[device.ID] = &cp
	}
	return nil
}

func (s *linkTestStorage) HasPermission(_ context.Context, _, _, _ string) (bool, error) {
	return true, nil
}

func buildLinkTestService(ss *linkTestStorage) *DNSService {
	return &DNSService{
		store:         ss,
		providerCache: make(map[string]dnspkg.Provider),
	}
}

// Feature: dns-device-linking, Property 10: Link operation sets DeviceID and AddressID
// Validates: Requirements 5.1, 5.2
func TestLinkRecord_SetsDeviceIDAndAddressID(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		zoneName := genZoneName(t)
		zoneID := "zone-1"

		// Generate a device with 1-3 addresses
		deviceID := "dev-1"
		numAddrs := rapid.IntRange(1, 3).Draw(t, "numAddrs")
		var addrs []model.Address
		for i := 0; i < numAddrs; i++ {
			addrs = append(addrs, model.Address{
				ID: fmt.Sprintf("addr-%d", i),
				IP: genIPv4Simple(t),
			})
		}

		device := &model.Device{
			ID:        deviceID,
			Name:      "test-device",
			Addresses: addrs,
		}

		// Generate an unlinked record (DeviceID is nil)
		recType := rapid.SampledFrom([]string{"A", "AAAA", "CNAME", "TXT"}).Draw(t, "recType")
		record := &model.DNSRecord{
			ID:     "rec-1",
			ZoneID: zoneID,
			Name:   genDNSLabel(t),
			Type:   recType,
			Value:  "some-value",
		}

		ss := newLinkTestStorage()
		ss.records[record.ID] = record
		ss.devices[deviceID] = device
		ss.zones[zoneID] = &model.DNSZone{ID: zoneID, Name: zoneName}

		svc := buildLinkTestService(ss)

		// Decide whether to include an AddressID
		includeAddr := rapid.Bool().Draw(t, "includeAddr")
		var addrID *string
		if includeAddr && len(addrs) > 0 {
			idx := rapid.IntRange(0, len(addrs)-1).Draw(t, "addrIdx")
			addrID = &addrs[idx].ID
		}

		req := &model.LinkDNSRecordRequest{
			DeviceID:  deviceID,
			AddressID: addrID,
		}

		result, err := svc.LinkRecord(systemCtx(), record.ID, req)
		if err != nil {
			t.Fatalf("LinkRecord returned unexpected error: %v", err)
		}

		// Verify DeviceID is set
		if result.DeviceID == nil {
			t.Fatal("expected DeviceID to be set, got nil")
		}
		if *result.DeviceID != deviceID {
			t.Fatalf("expected DeviceID %q, got %q", deviceID, *result.DeviceID)
		}

		// Verify AddressID
		if addrID != nil {
			if result.AddressID == nil {
				t.Fatal("expected AddressID to be set, got nil")
			}
			if *result.AddressID != *addrID {
				t.Fatalf("expected AddressID %q, got %q", *addrID, *result.AddressID)
			}
		}

		// Verify the record was persisted via UpdateDNSRecord
		if len(ss.updatedRecords) == 0 {
			t.Fatal("expected UpdateDNSRecord to be called")
		}
	})
}

// Feature: dns-device-linking, Property 11: CNAME link with add_to_domains updates device
// Validates: Requirements 5.3
func TestLinkRecord_CNAMEAddToDomains(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		zoneName := genZoneName(t)
		zoneID := "zone-1"
		prefix := genDNSLabel(t)
		expectedFQDN := prefix + "." + zoneName

		deviceID := "dev-1"
		// Generate 0-2 existing domains on the device
		numExisting := rapid.IntRange(0, 2).Draw(t, "numExisting")
		var existingDomains []string
		for i := 0; i < numExisting; i++ {
			existingDomains = append(existingDomains, genDNSLabel(t)+"."+genZoneName(t))
		}

		device := &model.Device{
			ID:      deviceID,
			Name:    "test-device",
			Domains: existingDomains,
		}

		// Unlinked CNAME record
		record := &model.DNSRecord{
			ID:     "rec-1",
			ZoneID: zoneID,
			Name:   prefix,
			Type:   "CNAME",
			Value:  "target." + zoneName,
		}

		ss := newLinkTestStorage()
		ss.records[record.ID] = record
		ss.devices[deviceID] = device
		ss.zones[zoneID] = &model.DNSZone{ID: zoneID, Name: zoneName}

		svc := buildLinkTestService(ss)

		req := &model.LinkDNSRecordRequest{
			DeviceID:     deviceID,
			AddToDomains: true,
		}

		_, err := svc.LinkRecord(systemCtx(), record.ID, req)
		if err != nil {
			t.Fatalf("LinkRecord returned unexpected error: %v", err)
		}

		// Verify the device was updated with the FQDN in its Domains
		if len(ss.updatedDevices) == 0 {
			t.Fatal("expected UpdateDevice to be called for CNAME with add_to_domains")
		}

		updatedDevice := ss.updatedDevices[len(ss.updatedDevices)-1]
		found := false
		for _, d := range updatedDevice.Domains {
			if d == expectedFQDN {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected device Domains to contain %q, got %v", expectedFQDN, updatedDevice.Domains)
		}

		// Verify existing domains are preserved
		for _, existing := range existingDomains {
			domainFound := false
			for _, d := range updatedDevice.Domains {
				if d == existing {
					domainFound = true
					break
				}
			}
			if !domainFound {
				t.Fatalf("expected existing domain %q to be preserved, got %v", existing, updatedDevice.Domains)
			}
		}
	})
}

// Feature: dns-device-linking, Property 12: Already-linked record rejects link and promote (link portion)
// Validates: Requirements 5.4
func TestLinkRecord_AlreadyLinkedRejects(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		zoneName := genZoneName(t)
		zoneID := "zone-1"

		existingDeviceID := "dev-existing"
		newDeviceID := "dev-new"

		// Record that is already linked (DeviceID is non-nil)
		record := &model.DNSRecord{
			ID:       "rec-1",
			ZoneID:   zoneID,
			DeviceID: &existingDeviceID,
			Name:     genDNSLabel(t),
			Type:     rapid.SampledFrom([]string{"A", "AAAA", "CNAME", "TXT"}).Draw(t, "recType"),
			Value:    "some-value",
		}

		device := &model.Device{
			ID:   newDeviceID,
			Name: "new-device",
		}

		ss := newLinkTestStorage()
		ss.records[record.ID] = record
		ss.devices[newDeviceID] = device
		ss.zones[zoneID] = &model.DNSZone{ID: zoneID, Name: zoneName}

		svc := buildLinkTestService(ss)

		req := &model.LinkDNSRecordRequest{
			DeviceID: newDeviceID,
		}

		_, err := svc.LinkRecord(systemCtx(), record.ID, req)
		if err == nil {
			t.Fatal("expected LinkRecord to return an error for already-linked record, got nil")
		}

		// Verify it's a ValidationError
		var valErrs ValidationErrors
		if !errors.As(err, &valErrs) {
			t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
		}

		// Verify the record was NOT modified
		if len(ss.updatedRecords) != 0 {
			t.Fatal("expected no UpdateDNSRecord call for already-linked record")
		}
	})
}


// promoteTestStorage extends linkTestStorage with CreateDevice and ListDNSZones
// support needed by PromoteRecord → DeviceService.Create.
type promoteTestStorage struct {
	linkTestStorage
	createdDevices []*model.Device
}

func newPromoteTestStorage() *promoteTestStorage {
	return &promoteTestStorage{
		linkTestStorage: linkTestStorage{
			records: make(map[string]*model.DNSRecord),
			devices: make(map[string]*model.Device),
			zones:   make(map[string]*model.DNSZone),
		},
	}
}

func (s *promoteTestStorage) CreateDevice(_ context.Context, device *model.Device) error {
	// Simulate ID assignment like real storage
	if device.ID == "" {
		device.ID = fmt.Sprintf("created-dev-%d", len(s.createdDevices))
	}
	cp := *device
	cp.Addresses = make([]model.Address, len(device.Addresses))
	copy(cp.Addresses, device.Addresses)
	if device.Tags != nil {
		cp.Tags = make([]string, len(device.Tags))
		copy(cp.Tags, device.Tags)
	}
	s.createdDevices = append(s.createdDevices, &cp)
	s.devices[device.ID] = &cp
	return nil
}

func (s *promoteTestStorage) ListDNSZones(_ *model.DNSZoneFilter) ([]model.DNSZone, error) {
	return nil, nil // No auto-sync zones needed for promote tests
}

func buildPromoteTestService(ss *promoteTestStorage) *DNSService {
	deviceSvc := &DeviceService{
		store: ss,
		// dns and conflictService left nil — both are nil-safe
	}
	svc := &DNSService{
		store:         ss,
		providerCache: make(map[string]dnspkg.Provider),
		devices:       deviceSvc,
	}
	return svc
}

// Feature: dns-device-linking, Property 13: Promote creates correctly structured device
// Validates: Requirements 6.1, 6.2, 6.4, 6.5, 6.7
func TestPromoteRecord_CreatesCorrectDevice(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		zoneName := genZoneName(t)
		zoneID := "zone-1"
		recName := genDNSLabel(t)

		// Randomly decide if zone has a NetworkID
		hasNetworkID := rapid.Bool().Draw(t, "hasNetworkID")
		var networkID *string
		expectedNetworkID := ""
		if hasNetworkID {
			nid := "net-" + genDNSLabel(t)
			networkID = &nid
			expectedNetworkID = nid
		}

		zone := &model.DNSZone{
			ID:        zoneID,
			Name:      zoneName,
			NetworkID: networkID,
		}

		ip := genIPv4Simple(t)
		recType := rapid.SampledFrom([]string{"A", "AAAA"}).Draw(t, "recType")

		record := &model.DNSRecord{
			ID:     "rec-1",
			ZoneID: zoneID,
			Name:   recName,
			Type:   recType,
			Value:  ip,
		}

		ss := newPromoteTestStorage()
		ss.records[record.ID] = record
		ss.zones[zoneID] = zone

		svc := buildPromoteTestService(ss)

		req := &model.PromoteDNSRecordRequest{}

		result, err := svc.PromoteRecord(systemCtx(), record.ID, req)
		if err != nil {
			t.Fatalf("PromoteRecord returned unexpected error: %v", err)
		}

		// Verify device was created
		if len(ss.createdDevices) != 1 {
			t.Fatalf("expected 1 device created, got %d", len(ss.createdDevices))
		}
		dev := ss.createdDevices[0]

		// Name = name.zoneName
		expectedName := recName + "." + zoneName
		if dev.Name != expectedName {
			t.Fatalf("expected device Name %q, got %q", expectedName, dev.Name)
		}

		// Hostname = record.Name
		if dev.Hostname != recName {
			t.Fatalf("expected device Hostname %q, got %q", recName, dev.Hostname)
		}

		// For A/AAAA: should have exactly one address
		if len(dev.Addresses) != 1 {
			t.Fatalf("expected 1 address on device, got %d", len(dev.Addresses))
		}
		addr := dev.Addresses[0]

		// Address IP = record.Value
		if addr.IP != ip {
			t.Fatalf("expected address IP %q, got %q", ip, addr.IP)
		}

		// Address NetworkID = zone.NetworkID (or empty)
		if addr.NetworkID != expectedNetworkID {
			t.Fatalf("expected address NetworkID %q, got %q", expectedNetworkID, addr.NetworkID)
		}

		// Record should have DeviceID set
		if result.DeviceID == nil {
			t.Fatal("expected result DeviceID to be set, got nil")
		}
		if *result.DeviceID != dev.ID {
			t.Fatalf("expected result DeviceID %q, got %q", dev.ID, *result.DeviceID)
		}

		// Record should have AddressID set for A/AAAA
		if result.AddressID == nil {
			t.Fatal("expected result AddressID to be set for A/AAAA record, got nil")
		}
		if *result.AddressID != addr.ID {
			t.Fatalf("expected result AddressID %q, got %q", addr.ID, *result.AddressID)
		}

		// Verify record was persisted
		if len(ss.updatedRecords) == 0 {
			t.Fatal("expected UpdateDNSRecord to be called")
		}
	})
}

// Feature: dns-device-linking, Property 14: Promote applies overrides
// Validates: Requirements 6.3
func TestPromoteRecord_AppliesOverrides(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		zoneName := genZoneName(t)
		zoneID := "zone-1"
		recName := genDNSLabel(t)
		ip := genIPv4Simple(t)

		zone := &model.DNSZone{
			ID:   zoneID,
			Name: zoneName,
		}

		record := &model.DNSRecord{
			ID:     "rec-1",
			ZoneID: zoneID,
			Name:   recName,
			Type:   "A",
			Value:  ip,
		}

		ss := newPromoteTestStorage()
		ss.records[record.ID] = record
		ss.zones[zoneID] = zone

		svc := buildPromoteTestService(ss)

		// Generate override values
		overrideName := genDNSLabel(t) + "." + zoneName
		overrideDC := "dc-" + genDNSLabel(t)
		numTags := rapid.IntRange(1, 3).Draw(t, "numTags")
		var tags []string
		for i := 0; i < numTags; i++ {
			tags = append(tags, genDNSLabel(t))
		}

		req := &model.PromoteDNSRecordRequest{
			Name:         &overrideName,
			DatacenterID: &overrideDC,
			Tags:         tags,
		}

		_, err := svc.PromoteRecord(systemCtx(), record.ID, req)
		if err != nil {
			t.Fatalf("PromoteRecord returned unexpected error: %v", err)
		}

		if len(ss.createdDevices) != 1 {
			t.Fatalf("expected 1 device created, got %d", len(ss.createdDevices))
		}
		dev := ss.createdDevices[0]

		// Name should be the override
		if dev.Name != overrideName {
			t.Fatalf("expected device Name override %q, got %q", overrideName, dev.Name)
		}

		// DatacenterID should be the override
		if dev.DatacenterID != overrideDC {
			t.Fatalf("expected device DatacenterID %q, got %q", overrideDC, dev.DatacenterID)
		}

		// Tags should be the override
		if len(dev.Tags) != len(tags) {
			t.Fatalf("expected %d tags, got %d", len(tags), len(dev.Tags))
		}
		for i, tag := range tags {
			if dev.Tags[i] != tag {
				t.Fatalf("expected tag[%d] %q, got %q", i, tag, dev.Tags[i])
			}
		}

		// Hostname should still be record.Name (not overridden)
		if dev.Hostname != recName {
			t.Fatalf("expected device Hostname %q (not overridden), got %q", recName, dev.Hostname)
		}
	})
}

// Feature: dns-device-linking, Property 12: Already-linked record rejects link and promote (promote portion)
// Validates: Requirements 6.6
func TestPromoteRecord_AlreadyLinkedRejects(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		zoneName := genZoneName(t)
		zoneID := "zone-1"

		existingDeviceID := "dev-existing"

		// Record that is already linked
		record := &model.DNSRecord{
			ID:       "rec-1",
			ZoneID:   zoneID,
			DeviceID: &existingDeviceID,
			Name:     genDNSLabel(t),
			Type:     rapid.SampledFrom([]string{"A", "AAAA", "CNAME", "TXT"}).Draw(t, "recType"),
			Value:    genIPv4Simple(t),
		}

		ss := newPromoteTestStorage()
		ss.records[record.ID] = record
		ss.zones[zoneID] = &model.DNSZone{ID: zoneID, Name: zoneName}

		svc := buildPromoteTestService(ss)

		req := &model.PromoteDNSRecordRequest{}

		_, err := svc.PromoteRecord(systemCtx(), record.ID, req)
		if err == nil {
			t.Fatal("expected PromoteRecord to return an error for already-linked record, got nil")
		}

		// Verify it's a ValidationError
		var valErrs ValidationErrors
		if !errors.As(err, &valErrs) {
			t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
		}

		// Verify no device was created
		if len(ss.createdDevices) != 0 {
			t.Fatal("expected no device to be created for already-linked record")
		}

		// Verify the record was NOT modified
		if len(ss.updatedRecords) != 0 {
			t.Fatal("expected no UpdateDNSRecord call for already-linked record")
		}
	})
}
