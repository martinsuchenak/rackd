package service

import (
	"context"
	"errors"
	"testing"
	"time"

	discoverypkg "github.com/martinsuchenak/rackd/internal/discovery"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type discoveryTestStorage struct {
	storage.ExtendedStorage
	permissions map[string]bool
	networks    map[string]*model.Network
	discovered  map[string]*model.DiscoveredDevice
	created     *model.Device
	promotedID  string
	promotedTo  string
}

func newDiscoveryTestStorage() *discoveryTestStorage {
	return &discoveryTestStorage{
		permissions: make(map[string]bool),
		networks:    make(map[string]*model.Network),
		discovered:  make(map[string]*model.DiscoveredDevice),
	}
}

func (s *discoveryTestStorage) setPermission(userID, resource, action string, allowed bool) {
	s.permissions[userID+":"+resource+":"+action] = allowed
}

func (s *discoveryTestStorage) HasPermission(_ context.Context, userID, resource, action string) (bool, error) {
	return s.permissions[userID+":"+resource+":"+action], nil
}

func (s *discoveryTestStorage) GetNetwork(_ context.Context, id string) (*model.Network, error) {
	network, ok := s.networks[id]
	if !ok {
		return nil, storage.ErrNetworkNotFound
	}
	cloned := *network
	return &cloned, nil
}

func (s *discoveryTestStorage) GetDiscoveredDevice(_ context.Context, id string) (*model.DiscoveredDevice, error) {
	device, ok := s.discovered[id]
	if !ok {
		return nil, storage.ErrDiscoveryNotFound
	}
	cloned := *device
	return &cloned, nil
}

func (s *discoveryTestStorage) CreateDevice(_ context.Context, device *model.Device) error {
	cloned := *device
	s.created = &cloned
	return nil
}

func (s *discoveryTestStorage) PromoteDiscoveredDevice(_ context.Context, discoveredID, deviceID string) error {
	s.promotedID = discoveredID
	s.promotedTo = deviceID
	return nil
}

type scannerStub struct {
	scanFn   func(ctx context.Context, network *model.Network, scanType string) (*model.DiscoveryScan, error)
	cancelFn func(ctx context.Context, scanID string) error
}

func (s *scannerStub) Scan(ctx context.Context, network *model.Network, scanType string) (*model.DiscoveryScan, error) {
	return s.scanFn(ctx, network, scanType)
}

func (s *scannerStub) GetScanStatus(_ context.Context, _ string) (*model.DiscoveryScan, error) {
	return nil, nil
}

func (s *scannerStub) CancelScan(ctx context.Context, scanID string) error {
	return s.cancelFn(ctx, scanID)
}

func TestDiscoveryService_StartScanDefaultsInvalidTypeToQuick(t *testing.T) {
	store := newDiscoveryTestStorage()
	store.setPermission("user-1", "discovery", "create", true)
	store.networks["net-1"] = &model.Network{ID: "net-1", Name: "prod-net", Subnet: "10.0.0.0/24"}

	var capturedType string
	scanner := &scannerStub{
		scanFn: func(_ context.Context, _ *model.Network, scanType string) (*model.DiscoveryScan, error) {
			capturedType = scanType
			return &model.DiscoveryScan{ID: "scan-1", ScanType: scanType}, nil
		},
	}

	svc := NewDiscoveryService(store, scanner)
	scan, err := svc.StartScan(userContext("user-1"), "net-1", "unexpected")
	if err != nil {
		t.Fatalf("StartScan returned unexpected error: %v", err)
	}
	if capturedType != model.ScanTypeQuick || scan.ScanType != model.ScanTypeQuick {
		t.Fatalf("expected invalid scan type to default to quick, got %q / %#v", capturedType, scan)
	}
}

func TestDiscoveryService_CancelScanMapsRunningAndMissingErrors(t *testing.T) {
	store := newDiscoveryTestStorage()
	store.setPermission("user-1", "discovery", "delete", true)
	svc := NewDiscoveryService(store, &scannerStub{
		cancelFn: func(_ context.Context, scanID string) error {
			if scanID == "missing" {
				return discoverypkg.ErrScanNotFound
			}
			return discoverypkg.ErrScanNotRunning
		},
	})

	err := svc.CancelScan(userContext("user-1"), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found for missing scan, got %v", err)
	}

	err = svc.CancelScan(userContext("user-1"), "done")
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for non-running scan, got %v", err)
	}
}

func TestDiscoveryService_PromoteDeviceCarriesOverDiscoveredFields(t *testing.T) {
	store := newDiscoveryTestStorage()
	store.setPermission("user-1", "discovery", "create", true)
	now := time.Now()
	store.discovered["disc-1"] = &model.DiscoveredDevice{
		ID:         "disc-1",
		IP:         "10.0.0.15",
		Hostname:   "ap-1",
		Vendor:     "Ubiquiti",
		OSGuess:    "EdgeOS",
		MACAddress: "aa:bb:cc:dd:ee:ff",
		OpenPorts:  []int{22, 443},
		Services: []model.ServiceInfo{
			{Port: 22, Protocol: "tcp", Service: "ssh", Version: "OpenSSH"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	svc := NewDiscoveryService(store, nil)
	device, err := svc.PromoteDevice(userContext("user-1"), "disc-1", &model.Device{Name: "edge-router"})
	if err != nil {
		t.Fatalf("PromoteDevice returned unexpected error: %v", err)
	}
	if len(device.Addresses) != 1 || device.Addresses[0].IP != "10.0.0.15" {
		t.Fatalf("expected discovered IP to be carried over, got %#v", device.Addresses)
	}
	if device.Hostname != "ap-1" || device.MakeModel != "Ubiquiti" || device.OS != "EdgeOS" {
		t.Fatalf("expected discovered metadata to be carried over, got %#v", device)
	}
	if store.created == nil || store.promotedID != "disc-1" || store.promotedTo == "" {
		t.Fatalf("expected create and promote side effects, got created=%#v promoted=%q->%q", store.created, store.promotedID, store.promotedTo)
	}
}

func TestDiscoveryService_RuleValidationAndDeleteMapping(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "discovery", "create", true)
	store.setPermission("user-1", "discovery", "update", true)
	store.setPermission("user-1", "discovery", "delete", true)
	svc := NewDiscoveryService(store, nil)

	err := svc.CreateRule(userContext("user-1"), &model.DiscoveryRule{})
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for missing network ID, got %v", err)
	}

	err = svc.UpdateRule(userContext("user-1"), &model.DiscoveryRule{NetworkID: "net-1"})
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for missing rule ID, got %v", err)
	}

	err = svc.DeleteRule(userContext("user-1"), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found for missing rule, got %v", err)
	}
}

func TestDiscoveryService_ListGetAndDeleteWrappers(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "discovery", "list", true)
	store.setPermission("user-1", "discovery", "read", true)
	store.setPermission("user-1", "discovery", "delete", true)
	store.discoveryScans["scan-1"] = &model.DiscoveryScan{ID: "scan-1", NetworkID: "net-1"}
	store.discoveredByNetwork["net-1"] = []model.DiscoveredDevice{{ID: "disc-1", NetworkID: "net-1"}}
	store.rules["rule-1"] = &model.DiscoveryRule{ID: "rule-1", NetworkID: "net-1"}
	svc := NewDiscoveryService(store, nil)

	if _, err := svc.ListScans(userContext("user-1"), "net-1"); err != nil {
		t.Fatalf("ListScans returned unexpected error: %v", err)
	}
	if _, err := svc.GetScan(userContext("user-1"), "scan-1"); err != nil {
		t.Fatalf("GetScan returned unexpected error: %v", err)
	}
	if _, err := svc.ListDevices(userContext("user-1"), "net-1"); err != nil {
		t.Fatalf("ListDevices returned unexpected error: %v", err)
	}
	if _, err := svc.GetDevice(userContext("user-1"), "disc-1"); err != nil {
		t.Fatalf("GetDevice returned unexpected error: %v", err)
	}
	if _, err := svc.ListRules(userContext("user-1")); err != nil {
		t.Fatalf("ListRules returned unexpected error: %v", err)
	}
	if _, err := svc.GetRule(userContext("user-1"), "rule-1"); err != nil {
		t.Fatalf("GetRule returned unexpected error: %v", err)
	}
	if err := svc.DeleteScan(userContext("user-1"), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found deleting missing scan, got %v", err)
	}
	if err := svc.DeleteDevice(userContext("user-1"), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found deleting missing discovered device, got %v", err)
	}
}
