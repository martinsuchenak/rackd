package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
			if err == discovery.ErrScanNotRunning {
				return ValidationErrors{{Field: "scan", Message: err.Error()}}
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

	// Carry over all discovered device data to supported Device fields
	device.ID = uuid.Must(uuid.NewV7()).String()

	// Set IP address from discovered device
	if device.Addresses == nil {
		device.Addresses = []model.Address{{IP: discovered.IP, Type: "ipv4"}}
	} else {
		// Add discovered IP to existing addresses
		device.Addresses = append(device.Addresses, model.Address{IP: discovered.IP, Type: "ipv4"})
	}

	// Set hostname from discovered device
	if discovered.Hostname != "" && device.Hostname == "" {
		device.Hostname = discovered.Hostname
	}

	// Set OS guess from discovered device
	if discovered.OSGuess != "" && device.OS == "" {
		device.OS = discovered.OSGuess
	}

	// Set vendor from discovered device (use MakeModel field)
	if discovered.Vendor != "" && device.MakeModel == "" {
		device.MakeModel = discovered.Vendor
	}

	// Store discovered device info in description for reference
	if discovered.MACAddress != "" || len(discovered.Services) > 0 || len(discovered.OpenPorts) > 0 {
		infoParts := []string{}
		if discovered.MACAddress != "" {
			infoParts = append(infoParts, fmt.Sprintf("MAC: %s", discovered.MACAddress))
		}
		if discovered.Vendor != "" {
			infoParts = append(infoParts, fmt.Sprintf("Vendor: %s", discovered.Vendor))
		}
		if discovered.OSGuess != "" {
			infoParts = append(infoParts, fmt.Sprintf("OS: %s", discovered.OSGuess))
		}
		if len(discovered.OpenPorts) > 0 {
			infoParts = append(infoParts, fmt.Sprintf("Ports: %v", discovered.OpenPorts))
		}
		if len(discovered.Services) > 0 {
			servicesInfo := make([]string, len(discovered.Services))
			for i, svc := range discovered.Services {
				if svc.Version != "" {
					servicesInfo[i] = fmt.Sprintf("%s@%d (%s %s)", svc.Service, svc.Port, svc.Protocol, svc.Version)
				} else {
					servicesInfo[i] = fmt.Sprintf("%s@%d (%s)", svc.Service, svc.Port, svc.Protocol)
				}
			}
			infoParts = append(infoParts, fmt.Sprintf("Services: %s", strings.Join(servicesInfo, ", ")))
		}

		if device.Description == "" {
			device.Description = strings.Join(infoParts, " | ")
		}
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
