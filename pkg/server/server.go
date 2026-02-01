// Package server provides the public server entry point for enterprise extension.
package server

import (
	"database/sql"
	"io"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/api"
	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/martinsuchenak/rackd/internal/log"
	intmcp "github.com/martinsuchenak/rackd/internal/mcp"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/server"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/rackd"
)

// featureAdapter wraps a public Feature to implement internal Feature
type featureAdapter struct {
	f rackd.Feature
}

func (a *featureAdapter) Name() string {
	return a.f.Name()
}

func (a *featureAdapter) RegisterRoutes(mux *http.ServeMux) {
	a.f.RegisterRoutes(mux)
}

func (a *featureAdapter) RegisterMCPTools(mcpServer *intmcp.Server) {
	a.f.RegisterMCPTools(mcpServer.Inner())
}

func (a *featureAdapter) ConfigureUI(builder *api.UIConfigBuilder) {
	a.f.ConfigureUI(&uiBuilderAdapter{builder})
}

// uiBuilderAdapter wraps internal UIConfigBuilder to implement public interface
type uiBuilderAdapter struct {
	b *api.UIConfigBuilder
}

func (a *uiBuilderAdapter) SetEdition(edition string) {
	a.b.SetEdition(edition)
}

func (a *uiBuilderAdapter) AddFeature(name string) {
	a.b.AddFeature(name)
}

func (a *uiBuilderAdapter) AddNavItem(item rackd.NavItem) {
	a.b.AddNavItem(api.NavItem{
		Label: item.Label,
		Path:  item.Path,
		Icon:  item.Icon,
		Order: item.Order,
	})
}

func (a *uiBuilderAdapter) SetUser(user *rackd.UserInfo) {
	if user == nil {
		a.b.SetUser(nil)
		return
	}
	a.b.SetUser(&api.UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Roles:    user.Roles,
	})
}

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
	d, err := s.internal.GetDevice(id)
	if err != nil {
		return nil, err
	}
	return convertDeviceToPublic(d), nil
}

func (s *StorageAdapter) CreateDevice(device *rackd.Device) error {
	return s.internal.CreateDevice(convertDeviceToInternal(device))
}

func (s *StorageAdapter) UpdateDevice(device *rackd.Device) error {
	return s.internal.UpdateDevice(convertDeviceToInternal(device))
}

func (s *StorageAdapter) DeleteDevice(id string) error {
	return s.internal.DeleteDevice(id)
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
	devices, err := s.internal.ListDevices(internalFilter)
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
	devices, err := s.internal.SearchDevices(query)
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
	n, err := s.internal.GetNetwork(id)
	if err != nil {
		return nil, err
	}
	return convertNetworkToPublic(n), nil
}

func (s *StorageAdapter) CreateNetwork(network *rackd.Network) error {
	return s.internal.CreateNetwork(convertNetworkToInternal(network))
}

func (s *StorageAdapter) UpdateNetwork(network *rackd.Network) error {
	return s.internal.UpdateNetwork(convertNetworkToInternal(network))
}

func (s *StorageAdapter) DeleteNetwork(id string) error {
	return s.internal.DeleteNetwork(id)
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
	networks, err := s.internal.ListNetworks(internalFilter)
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
func (s *StorageAdapter) CreateDiscoveredDevice(device *rackd.DiscoveredDevice) error {
	return s.internal.CreateDiscoveredDevice(convertDiscoveredDeviceToInternal(device))
}

func (s *StorageAdapter) UpdateDiscoveredDevice(device *rackd.DiscoveredDevice) error {
	return s.internal.UpdateDiscoveredDevice(convertDiscoveredDeviceToInternal(device))
}

func (s *StorageAdapter) GetDiscoveredDevice(id string) (*rackd.DiscoveredDevice, error) {
	d, err := s.internal.GetDiscoveredDevice(id)
	if err != nil {
		return nil, err
	}
	return convertDiscoveredDeviceToPublic(d), nil
}

func (s *StorageAdapter) GetDiscoveredDeviceByIP(networkID, ip string) (*rackd.DiscoveredDevice, error) {
	d, err := s.internal.GetDiscoveredDeviceByIP(networkID, ip)
	if err != nil {
		return nil, err
	}
	return convertDiscoveredDeviceToPublic(d), nil
}

func (s *StorageAdapter) ListDiscoveredDevices(networkID string) ([]rackd.DiscoveredDevice, error) {
	devices, err := s.internal.ListDiscoveredDevices(networkID)
	if err != nil {
		return nil, err
	}
	result := make([]rackd.DiscoveredDevice, len(devices))
	for i, d := range devices {
		result[i] = *convertDiscoveredDeviceToPublic(&d)
	}
	return result, nil
}

func (s *StorageAdapter) DeleteDiscoveredDevice(id string) error {
	return s.internal.DeleteDiscoveredDevice(id)
}

func (s *StorageAdapter) PromoteDiscoveredDevice(discoveredID, deviceID string) error {
	return s.internal.PromoteDiscoveredDevice(discoveredID, deviceID)
}

func (s *StorageAdapter) CreateDiscoveryScan(scan *rackd.DiscoveryScan) error {
	return s.internal.CreateDiscoveryScan(convertDiscoveryScanToInternal(scan))
}

func (s *StorageAdapter) UpdateDiscoveryScan(scan *rackd.DiscoveryScan) error {
	return s.internal.UpdateDiscoveryScan(convertDiscoveryScanToInternal(scan))
}

func (s *StorageAdapter) GetDiscoveryScan(id string) (*rackd.DiscoveryScan, error) {
	scan, err := s.internal.GetDiscoveryScan(id)
	if err != nil {
		return nil, err
	}
	return convertDiscoveryScanToPublic(scan), nil
}

func (s *StorageAdapter) ListDiscoveryScans(networkID string) ([]rackd.DiscoveryScan, error) {
	scans, err := s.internal.ListDiscoveryScans(networkID)
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

// Run starts the server with optional features
func Run(cfg *config.Config, store *StorageAdapter, features ...rackd.Feature) error {
	return RunWithCustomRoutes(cfg, store, nil, features...)
}

// RunWithCustomRoutes starts the server with optional features and custom route registration
func RunWithCustomRoutes(cfg *config.Config, store *StorageAdapter, registerRoutes func(mux *http.ServeMux), features ...rackd.Feature) error {
	// Convert public features to internal features
	internalFeatures := make([]server.Feature, len(features))
	for i, f := range features {
		internalFeatures[i] = &featureAdapter{f}
	}
	return server.RunWithCustomRoutes(cfg, store.internal, registerRoutes, internalFeatures...)
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
