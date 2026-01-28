package storage

import (
	"database/sql"
	"errors"

	"github.com/martinsuchenak/rackd/internal/model"
)

// Predefined errors for storage operations
var (
	ErrDeviceNotFound     = errors.New("device not found")
	ErrInvalidID          = errors.New("invalid ID")
	ErrDatacenterNotFound = errors.New("datacenter not found")
	ErrNetworkNotFound    = errors.New("network not found")
	ErrPoolNotFound       = errors.New("network pool not found")
	ErrDiscoveryNotFound  = errors.New("discovered device not found")
	ErrScanNotFound       = errors.New("scan not found")
	ErrRuleNotFound       = errors.New("discovery rule not found")
	ErrIPNotAvailable     = errors.New("no IP addresses available")
	ErrIPConflict         = errors.New("IP address already in use")
)

// DeviceStorage defines device persistence operations
type DeviceStorage interface {
	GetDevice(id string) (*model.Device, error)
	CreateDevice(device *model.Device) error
	UpdateDevice(device *model.Device) error
	DeleteDevice(id string) error
	ListDevices(filter *model.DeviceFilter) ([]model.Device, error)
	SearchDevices(query string) ([]model.Device, error)
}

// DatacenterStorage defines datacenter persistence operations
type DatacenterStorage interface {
	ListDatacenters(filter *model.DatacenterFilter) ([]model.Datacenter, error)
	GetDatacenter(id string) (*model.Datacenter, error)
	CreateDatacenter(dc *model.Datacenter) error
	UpdateDatacenter(dc *model.Datacenter) error
	DeleteDatacenter(id string) error
	GetDatacenterDevices(datacenterID string) ([]model.Device, error)
}

// NetworkStorage defines network persistence operations
type NetworkStorage interface {
	ListNetworks(filter *model.NetworkFilter) ([]model.Network, error)
	GetNetwork(id string) (*model.Network, error)
	CreateNetwork(network *model.Network) error
	UpdateNetwork(network *model.Network) error
	DeleteNetwork(id string) error
	GetNetworkDevices(networkID string) ([]model.Device, error)
	GetNetworkUtilization(networkID string) (*model.NetworkUtilization, error)
}

// NetworkPoolStorage defines network pool persistence operations
type NetworkPoolStorage interface {
	CreateNetworkPool(pool *model.NetworkPool) error
	UpdateNetworkPool(pool *model.NetworkPool) error
	DeleteNetworkPool(id string) error
	GetNetworkPool(id string) (*model.NetworkPool, error)
	ListNetworkPools(filter *model.NetworkPoolFilter) ([]model.NetworkPool, error)
	GetNextAvailableIP(poolID string) (string, error)
	ValidateIPInPool(poolID, ip string) (bool, error)
	GetPoolHeatmap(poolID string) ([]IPStatus, error)
}

// IPStatus represents the status of an IP in a pool heatmap
type IPStatus struct {
	IP       string `json:"ip"`
	Status   string `json:"status"` // "available", "used", "reserved"
	DeviceID string `json:"device_id,omitempty"`
}

// RelationshipStorage defines device relationship operations
type RelationshipStorage interface {
	AddRelationship(parentID, childID, relationshipType string) error
	RemoveRelationship(parentID, childID, relationshipType string) error
	GetRelationships(deviceID string) ([]model.DeviceRelationship, error)
	GetRelatedDevices(deviceID, relationshipType string) ([]model.Device, error)
}

// DiscoveryStorage defines discovery persistence operations
type DiscoveryStorage interface {
	// Discovered devices
	CreateDiscoveredDevice(device *model.DiscoveredDevice) error
	UpdateDiscoveredDevice(device *model.DiscoveredDevice) error
	GetDiscoveredDevice(id string) (*model.DiscoveredDevice, error)
	GetDiscoveredDeviceByIP(networkID, ip string) (*model.DiscoveredDevice, error)
	ListDiscoveredDevices(networkID string) ([]model.DiscoveredDevice, error)
	DeleteDiscoveredDevice(id string) error
	DeleteDiscoveredDevicesByNetwork(networkID string) error
	PromoteDiscoveredDevice(discoveredID, deviceID string) error

	// Discovery scans
	CreateDiscoveryScan(scan *model.DiscoveryScan) error
	UpdateDiscoveryScan(scan *model.DiscoveryScan) error
	GetDiscoveryScan(id string) (*model.DiscoveryScan, error)
	ListDiscoveryScans(networkID string) ([]model.DiscoveryScan, error)
	DeleteDiscoveryScan(id string) error

	// Discovery rules
	GetDiscoveryRule(id string) (*model.DiscoveryRule, error)
	GetDiscoveryRuleByNetwork(networkID string) (*model.DiscoveryRule, error)
	SaveDiscoveryRule(rule *model.DiscoveryRule) error
	ListDiscoveryRules() ([]model.DiscoveryRule, error)
	DeleteDiscoveryRule(id string) error

	// Cleanup
	CleanupOldDiscoveries(olderThanDays int) error
}

// Storage is the base interface
type Storage interface {
	DeviceStorage
}

// ExtendedStorage combines all storage interfaces
type ExtendedStorage interface {
	Storage
	RelationshipStorage
	DatacenterStorage
	NetworkStorage
	NetworkPoolStorage
	DiscoveryStorage
	Close() error
	DB() *sql.DB
}

// NewStorage creates a new base storage instance
func NewStorage(dataDir string) (Storage, error) {
	return NewSQLiteStorage(dataDir)
}

// NewExtendedStorage creates a new extended storage instance
func NewExtendedStorage(dataDir string) (ExtendedStorage, error) {
	return NewSQLiteStorage(dataDir)
}
