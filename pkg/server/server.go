// Package server provides the public server entry point for extensions.
package server

import (
	"context"
	"database/sql"
	"io"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/server"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/rackd"
)

// StorageAdapter wraps internal storage to implement public interfaces
type StorageAdapter struct {
	internal storage.ExtendedStorage
}

func (s *StorageAdapter) DB() *sql.DB {
	return s.internal.DB()
}

func (s *StorageAdapter) Close() error {
	return s.internal.Close()
}

// DeviceStorage methods
func (s *StorageAdapter) GetDevice(id string) (*rackd.Device, error) {
	d, err := s.internal.GetDevice(context.Background(), id)
	if err != nil {
		return nil, err
	}
	return convertDeviceToPublic(d), nil
}

func (s *StorageAdapter) CreateDevice(ctx context.Context, device *rackd.Device) error {
	return s.internal.CreateDevice(ctx, convertDeviceToInternal(device))
}

func (s *StorageAdapter) UpdateDevice(ctx context.Context, device *rackd.Device) error {
	return s.internal.UpdateDevice(ctx, convertDeviceToInternal(device))
}

func (s *StorageAdapter) DeleteDevice(ctx context.Context, id string) error {
	return s.internal.DeleteDevice(ctx, id)
}

func (s *StorageAdapter) ListDevices(filter *rackd.DeviceFilter) ([]rackd.Device, error) {
	var internalFilter *model.DeviceFilter
	if filter != nil {
		internalFilter = &model.DeviceFilter{
			DatacenterID: filter.DatacenterID,
			Tags:         filter.Tags,
			NetworkID:    filter.NetworkID,
		}
	}
	devices, err := s.internal.ListDevices(context.Background(), internalFilter)
	if err != nil {
		return nil, err
	}
	result := make([]rackd.Device, len(devices))
	for i, d := range devices {
		result[i] = *convertDeviceToPublic(&d)
	}
	return result, nil
}

func (s *StorageAdapter) SearchDevices(query string) ([]rackd.Device, error) {
	devices, err := s.internal.SearchDevices(context.Background(), query)
	if err != nil {
		return nil, err
	}
	result := make([]rackd.Device, len(devices))
	for i, d := range devices {
		result[i] = *convertDeviceToPublic(&d)
	}
	return result, nil
}

// NetworkStorage methods
func (s *StorageAdapter) GetNetwork(id string) (*rackd.Network, error) {
	n, err := s.internal.GetNetwork(context.Background(), id)
	if err != nil {
		return nil, err
	}
	return convertNetworkToPublic(n), nil
}

func (s *StorageAdapter) CreateNetwork(ctx context.Context, network *rackd.Network) error {
	return s.internal.CreateNetwork(ctx, convertNetworkToInternal(network))
}

func (s *StorageAdapter) UpdateNetwork(ctx context.Context, network *rackd.Network) error {
	return s.internal.UpdateNetwork(ctx, convertNetworkToInternal(network))
}

func (s *StorageAdapter) DeleteNetwork(ctx context.Context, id string) error {
	return s.internal.DeleteNetwork(ctx, id)
}

func (s *StorageAdapter) ListNetworks(filter *rackd.NetworkFilter) ([]rackd.Network, error) {
	var internalFilter *model.NetworkFilter
	if filter != nil {
		internalFilter = &model.NetworkFilter{
			Name:         filter.Name,
			DatacenterID: filter.DatacenterID,
			VLANID:       filter.VLANID,
		}
	}
	networks, err := s.internal.ListNetworks(context.Background(), internalFilter)
	if err != nil {
		return nil, err
	}
	result := make([]rackd.Network, len(networks))
	for i, n := range networks {
		result[i] = *convertNetworkToPublic(&n)
	}
	return result, nil
}

// DiscoveryStorage methods
func (s *StorageAdapter) CreateDiscoveredDevice(ctx context.Context, device *rackd.DiscoveredDevice) error {
	return s.internal.CreateDiscoveredDevice(ctx, convertDiscoveredDeviceToInternal(device))
}

func (s *StorageAdapter) UpdateDiscoveredDevice(ctx context.Context, device *rackd.DiscoveredDevice) error {
	return s.internal.UpdateDiscoveredDevice(ctx, convertDiscoveredDeviceToInternal(device))
}

func (s *StorageAdapter) GetDiscoveredDevice(id string) (*rackd.DiscoveredDevice, error) {
	d, err := s.internal.GetDiscoveredDevice(context.Background(), id)
	if err != nil {
		return nil, err
	}
	return convertDiscoveredDeviceToPublic(d), nil
}

func (s *StorageAdapter) GetDiscoveredDeviceByIP(networkID, ip string) (*rackd.DiscoveredDevice, error) {
	d, err := s.internal.GetDiscoveredDeviceByIP(context.Background(), networkID, ip)
	if err != nil {
		return nil, err
	}
	return convertDiscoveredDeviceToPublic(d), nil
}

func (s *StorageAdapter) ListDiscoveredDevices(networkID string) ([]rackd.DiscoveredDevice, error) {
	devices, err := s.internal.ListDiscoveredDevices(context.Background(), networkID)
	if err != nil {
		return nil, err
	}
	result := make([]rackd.DiscoveredDevice, len(devices))
	for i, d := range devices {
		result[i] = *convertDiscoveredDeviceToPublic(&d)
	}
	return result, nil
}

func (s *StorageAdapter) DeleteDiscoveredDevice(ctx context.Context, id string) error {
	return s.internal.DeleteDiscoveredDevice(ctx, id)
}

func (s *StorageAdapter) PromoteDiscoveredDevice(ctx context.Context, discoveredID, deviceID string) error {
	return s.internal.PromoteDiscoveredDevice(ctx, discoveredID, deviceID)
}

func (s *StorageAdapter) CreateDiscoveryScan(ctx context.Context, scan *rackd.DiscoveryScan) error {
	return s.internal.CreateDiscoveryScan(ctx, convertDiscoveryScanToInternal(scan))
}

func (s *StorageAdapter) UpdateDiscoveryScan(ctx context.Context, scan *rackd.DiscoveryScan) error {
	return s.internal.UpdateDiscoveryScan(ctx, convertDiscoveryScanToInternal(scan))
}

func (s *StorageAdapter) GetDiscoveryScan(id string) (*rackd.DiscoveryScan, error) {
	scan, err := s.internal.GetDiscoveryScan(context.Background(), id)
	if err != nil {
		return nil, err
	}
	return convertDiscoveryScanToPublic(scan), nil
}

func (s *StorageAdapter) ListDiscoveryScans(networkID string) ([]rackd.DiscoveryScan, error) {
	scans, err := s.internal.ListDiscoveryScans(context.Background(), networkID)
	if err != nil {
		return nil, err
	}
	result := make([]rackd.DiscoveryScan, len(scans))
	for i, scan := range scans {
		result[i] = *convertDiscoveryScanToPublic(&scan)
	}
	return result, nil
}

// Conversion helpers
func convertDeviceToPublic(d *model.Device) *rackd.Device {
	addresses := make([]rackd.Address, len(d.Addresses))
	for i, a := range d.Addresses {
		addresses[i] = rackd.Address{
			IP:         a.IP,
			Port:       a.Port,
			Type:       a.Type,
			Label:      a.Label,
			NetworkID:  a.NetworkID,
			SwitchPort: a.SwitchPort,
			PoolID:     a.PoolID,
		}
	}
	return &rackd.Device{
		ID:           d.ID,
		Name:         d.Name,
		Description:  d.Description,
		MakeModel:    d.MakeModel,
		OS:           d.OS,
		DatacenterID: d.DatacenterID,
		Username:     d.Username,
		Location:     d.Location,
		Tags:         d.Tags,
		Addresses:    addresses,
		Domains:      d.Domains,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}

func convertDeviceToInternal(d *rackd.Device) *model.Device {
	addresses := make([]model.Address, len(d.Addresses))
	for i, a := range d.Addresses {
		addresses[i] = model.Address{
			IP:         a.IP,
			Port:       a.Port,
			Type:       a.Type,
			Label:      a.Label,
			NetworkID:  a.NetworkID,
			SwitchPort: a.SwitchPort,
			PoolID:     a.PoolID,
		}
	}
	return &model.Device{
		ID:           d.ID,
		Name:         d.Name,
		Description:  d.Description,
		MakeModel:    d.MakeModel,
		OS:           d.OS,
		DatacenterID: d.DatacenterID,
		Username:     d.Username,
		Location:     d.Location,
		Tags:         d.Tags,
		Addresses:    addresses,
		Domains:      d.Domains,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}

func convertNetworkToPublic(n *model.Network) *rackd.Network {
	return &rackd.Network{
		ID:           n.ID,
		Name:         n.Name,
		Subnet:       n.Subnet,
		VLANID:       n.VLANID,
		DatacenterID: n.DatacenterID,
		Description:  n.Description,
		CreatedAt:    n.CreatedAt,
		UpdatedAt:    n.UpdatedAt,
	}
}

func convertNetworkToInternal(n *rackd.Network) *model.Network {
	return &model.Network{
		ID:           n.ID,
		Name:         n.Name,
		Subnet:       n.Subnet,
		VLANID:       n.VLANID,
		DatacenterID: n.DatacenterID,
		Description:  n.Description,
		CreatedAt:    n.CreatedAt,
		UpdatedAt:    n.UpdatedAt,
	}
}

func convertDiscoveredDeviceToPublic(d *model.DiscoveredDevice) *rackd.DiscoveredDevice {
	services := make([]rackd.ServiceInfo, len(d.Services))
	for i, s := range d.Services {
		services[i] = rackd.ServiceInfo{
			Port:     s.Port,
			Protocol: s.Protocol,
			Service:  s.Service,
			Version:  s.Version,
		}
	}
	return &rackd.DiscoveredDevice{
		ID:                 d.ID,
		IP:                 d.IP,
		MACAddress:         d.MACAddress,
		Hostname:           d.Hostname,
		NetworkID:          d.NetworkID,
		Status:             d.Status,
		Confidence:         d.Confidence,
		OSGuess:            d.OSGuess,
		Vendor:             d.Vendor,
		OpenPorts:          d.OpenPorts,
		Services:           services,
		FirstSeen:          d.FirstSeen,
		LastSeen:           d.LastSeen,
		PromotedToDeviceID: d.PromotedToDeviceID,
		PromotedAt:         d.PromotedAt,
	}
}

func convertDiscoveredDeviceToInternal(d *rackd.DiscoveredDevice) *model.DiscoveredDevice {
	services := make([]model.ServiceInfo, len(d.Services))
	for i, s := range d.Services {
		services[i] = model.ServiceInfo{
			Port:     s.Port,
			Protocol: s.Protocol,
			Service:  s.Service,
			Version:  s.Version,
		}
	}
	return &model.DiscoveredDevice{
		ID:                 d.ID,
		IP:                 d.IP,
		MACAddress:         d.MACAddress,
		Hostname:           d.Hostname,
		NetworkID:          d.NetworkID,
		Status:             d.Status,
		Confidence:         d.Confidence,
		OSGuess:            d.OSGuess,
		Vendor:             d.Vendor,
		OpenPorts:          d.OpenPorts,
		Services:           services,
		FirstSeen:          d.FirstSeen,
		LastSeen:           d.LastSeen,
		PromotedToDeviceID: d.PromotedToDeviceID,
		PromotedAt:         d.PromotedAt,
	}
}

func convertDiscoveryScanToPublic(s *model.DiscoveryScan) *rackd.DiscoveryScan {
	return &rackd.DiscoveryScan{
		ID:              s.ID,
		NetworkID:       s.NetworkID,
		Status:          s.Status,
		ScanType:        s.ScanType,
		TotalHosts:      s.TotalHosts,
		ScannedHosts:    s.ScannedHosts,
		FoundHosts:      s.FoundHosts,
		ProgressPercent: s.ProgressPercent,
		StartedAt:       s.StartedAt,
		CompletedAt:     s.CompletedAt,
		ErrorMessage:    s.ErrorMessage,
	}
}

func convertDiscoveryScanToInternal(s *rackd.DiscoveryScan) *model.DiscoveryScan {
	return &model.DiscoveryScan{
		ID:              s.ID,
		NetworkID:       s.NetworkID,
		Status:          s.Status,
		ScanType:        s.ScanType,
		TotalHosts:      s.TotalHosts,
		ScannedHosts:    s.ScannedHosts,
		FoundHosts:      s.FoundHosts,
		ProgressPercent: s.ProgressPercent,
		StartedAt:       s.StartedAt,
		CompletedAt:     s.CompletedAt,
		ErrorMessage:    s.ErrorMessage,
	}
}

// Run starts the server
func Run(cfg *config.Config, store *StorageAdapter) error {
	return RunWithCustomRoutes(cfg, store, nil)
}

// RunWithCustomRoutes starts the server with custom route registration
func RunWithCustomRoutes(cfg *config.Config, store *StorageAdapter, registerRoutes func(mux *http.ServeMux)) error {
	return server.RunWithCustomRoutes(cfg, store.internal, registerRoutes)
}

// LoadConfig loads configuration from environment
func LoadConfig() *config.Config {
	return config.Load()
}

// InitLogger initializes the logging system with the specified format, level, and writer
func InitLogger(logFormat, logLevel string, writer io.Writer) {
	log.Init(logFormat, logLevel, writer)
}

// NewStorage creates a new storage instance
func NewStorage(dataDir string) (*StorageAdapter, error) {
	s, err := storage.NewExtendedStorage(dataDir)
	if err != nil {
		return nil, err
	}
	return &StorageAdapter{internal: s}, nil
}
