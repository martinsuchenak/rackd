package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/discovery"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type DiscoveryService struct {
	store   storage.ExtendedStorage
	scanner discovery.Scanner
}

func NewDiscoveryService(store storage.ExtendedStorage, scanner discovery.Scanner) *DiscoveryService {
	return &DiscoveryService{
		store:   store,
		scanner: scanner,
	}
}

func (s *DiscoveryService) StartScan(ctx context.Context, networkID, scanType string) (*model.DiscoveryScan, error) {
	if err := requirePermission(ctx, s.store, "discovery", "create"); err != nil {
		return nil, err
	}

	if networkID == "" {
		return nil, ValidationErrors{{Field: "network_id", Message: "Network ID is required"}}
	}

	network, err := s.store.GetNetwork(networkID)
	if err != nil {
		return nil, err
	}

	if scanType != model.ScanTypeQuick && scanType != model.ScanTypeFull && scanType != model.ScanTypeDeep {
		scanType = model.ScanTypeQuick
	}

	if s.scanner != nil {
		scan, err := s.scanner.Scan(ctx, network, scanType)
		if err != nil {
			return nil, err
		}
		return scan, nil
	}

	return nil, ErrValidation
}

func (s *DiscoveryService) ListScans(ctx context.Context, networkID string) ([]model.DiscoveryScan, error) {
	if err := requirePermission(ctx, s.store, "discovery", "list"); err != nil {
		return nil, err
	}
	return s.store.ListDiscoveryScans(networkID)
}

func (s *DiscoveryService) GetScan(ctx context.Context, id string) (*model.DiscoveryScan, error) {
	if err := requirePermission(ctx, s.store, "discovery", "read"); err != nil {
		return nil, err
	}

	scan, err := s.store.GetDiscoveryScan(id)
	if err != nil {
		if errors.Is(err, storage.ErrScanNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return scan, nil
}

func (s *DiscoveryService) CancelScan(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "discovery", "delete"); err != nil {
		return err
	}

	if s.scanner != nil {
		if err := s.scanner.CancelScan(id); err != nil {
			if err == discovery.ErrScanNotFound {
				return ErrNotFound
			}
			return err
		}
		return nil
	}

	return ErrValidation
}

func (s *DiscoveryService) DeleteScan(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "discovery", "delete"); err != nil {
		return err
	}

	if err := s.store.DeleteDiscoveryScan(enrichAuditCtx(ctx), id); err != nil {
		if errors.Is(err, storage.ErrScanNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *DiscoveryService) ListDevices(ctx context.Context, networkID string) ([]model.DiscoveredDevice, error) {
	if err := requirePermission(ctx, s.store, "discovery", "list"); err != nil {
		return nil, err
	}
	return s.store.ListDiscoveredDevices(networkID)
}

func (s *DiscoveryService) GetDevice(ctx context.Context, id string) (*model.DiscoveredDevice, error) {
	if err := requirePermission(ctx, s.store, "discovery", "read"); err != nil {
		return nil, err
	}

	device, err := s.store.GetDiscoveredDevice(id)
	if err != nil {
		if errors.Is(err, storage.ErrDiscoveryNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return device, nil
}

func (s *DiscoveryService) DeleteDevicesByNetwork(ctx context.Context, networkID string) error {
	if err := requirePermission(ctx, s.store, "discovery", "delete"); err != nil {
		return err
	}

	return s.store.DeleteDiscoveredDevicesByNetwork(enrichAuditCtx(ctx), networkID)
}

func (s *DiscoveryService) DeleteDevice(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "discovery", "delete"); err != nil {
		return err
	}

	if err := s.store.DeleteDiscoveredDevice(enrichAuditCtx(ctx), id); err != nil {
		if errors.Is(err, storage.ErrDiscoveryNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *DiscoveryService) PromoteDevice(ctx context.Context, discoveredID string, device *model.Device) (*model.Device, error) {
	if err := requirePermission(ctx, s.store, "discovery", "create"); err != nil {
		return nil, err
	}

	if discoveredID == "" {
		return nil, ValidationErrors{{Field: "discovered_id", Message: "Discovered device ID is required"}}
	}

	if device.Name == "" {
		return nil, ValidationErrors{{Field: "name", Message: "Device name is required"}}
	}

	discovered, err := s.store.GetDiscoveredDevice(discoveredID)
	if err != nil {
		if errors.Is(err, storage.ErrDiscoveryNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	device.ID = uuid.Must(uuid.NewV7()).String()
	if device.Addresses == nil {
		device.Addresses = []model.Address{{IP: discovered.IP, Type: "ipv4"}}
	} else {
		device.Addresses = append(device.Addresses, model.Address{IP: discovered.IP, Type: "ipv4"})
	}

	if discovered.Hostname != "" && device.Domains == nil {
		device.Domains = []string{discovered.Hostname}
	}

	if err := s.store.CreateDevice(enrichAuditCtx(ctx), device); err != nil {
		return nil, err
	}

	if err := s.store.PromoteDiscoveredDevice(enrichAuditCtx(ctx), discoveredID, device.ID); err != nil {
		return nil, err
	}

	return device, nil
}

func (s *DiscoveryService) ListRules(ctx context.Context) ([]model.DiscoveryRule, error) {
	if err := requirePermission(ctx, s.store, "discovery", "list"); err != nil {
		return nil, err
	}
	return s.store.ListDiscoveryRules()
}

func (s *DiscoveryService) GetRule(ctx context.Context, id string) (*model.DiscoveryRule, error) {
	if err := requirePermission(ctx, s.store, "discovery", "read"); err != nil {
		return nil, err
	}

	rule, err := s.store.GetDiscoveryRule(id)
	if err != nil {
		if errors.Is(err, storage.ErrRuleNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return rule, nil
}

func (s *DiscoveryService) CreateRule(ctx context.Context, rule *model.DiscoveryRule) error {
	if err := requirePermission(ctx, s.store, "discovery", "create"); err != nil {
		return err
	}

	if rule.NetworkID == "" {
		return ValidationErrors{{Field: "network_id", Message: "Network ID is required"}}
	}

	return s.store.SaveDiscoveryRule(enrichAuditCtx(ctx), rule)
}

func (s *DiscoveryService) UpdateRule(ctx context.Context, rule *model.DiscoveryRule) error {
	if err := requirePermission(ctx, s.store, "discovery", "update"); err != nil {
		return err
	}

	if rule.ID == "" {
		return ValidationErrors{{Field: "id", Message: "ID is required"}}
	}

	if rule.NetworkID == "" {
		return ValidationErrors{{Field: "network_id", Message: "Network ID is required"}}
	}

	return s.store.SaveDiscoveryRule(enrichAuditCtx(ctx), rule)
}

func (s *DiscoveryService) DeleteRule(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "discovery", "delete"); err != nil {
		return err
	}

	if err := s.store.DeleteDiscoveryRule(enrichAuditCtx(ctx), id); err != nil {
		if errors.Is(err, storage.ErrRuleNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}
