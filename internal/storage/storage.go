package storage

import (
	"context"
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
	ErrAuditLogNotFound   = errors.New("audit log not found")
	ErrUserNotFound         = errors.New("user not found")
	ErrOAuthClientNotFound  = errors.New("oauth client not found")
	ErrOAuthCodeNotFound    = errors.New("oauth authorization code not found")
	ErrOAuthCodeExpired     = errors.New("oauth authorization code expired")
	ErrOAuthCodeUsed        = errors.New("oauth authorization code already used")
	ErrOAuthTokenNotFound   = errors.New("oauth token not found")
	ErrOAuthTokenRevoked    = errors.New("oauth token revoked")
	ErrOAuthTokenExpired    = errors.New("oauth token expired")
	ErrReservationNotFound  = errors.New("reservation not found")
	ErrReservationExpired   = errors.New("reservation has expired")
	ErrIPAlreadyReserved    = errors.New("IP address is already reserved")
	ErrWebhookNotFound      = errors.New("webhook not found")
	ErrDeliveryNotFound     = errors.New("webhook delivery not found")
	ErrCustomFieldNotFound  = errors.New("custom field definition not found")
	ErrDuplicateFieldKey    = errors.New("custom field key already exists")
	ErrCircuitNotFound      = errors.New("circuit not found")
	ErrNATNotFound          = errors.New("NAT mapping not found")
)

// DeviceStorage defines device persistence operations
type DeviceStorage interface {
	GetDevice(id string) (*model.Device, error)
	CreateDevice(ctx context.Context, device *model.Device) error
	UpdateDevice(ctx context.Context, device *model.Device) error
	DeleteDevice(ctx context.Context, id string) error
	ListDevices(filter *model.DeviceFilter) ([]model.Device, error)
	SearchDevices(query string) ([]model.Device, error)
	GetDeviceStatusCounts() (map[model.DeviceStatus]int, error)
}

// DatacenterStorage defines datacenter persistence operations
type DatacenterStorage interface {
	ListDatacenters(filter *model.DatacenterFilter) ([]model.Datacenter, error)
	GetDatacenter(id string) (*model.Datacenter, error)
	CreateDatacenter(ctx context.Context, dc *model.Datacenter) error
	UpdateDatacenter(ctx context.Context, dc *model.Datacenter) error
	DeleteDatacenter(ctx context.Context, id string) error
	GetDatacenterDevices(datacenterID string) ([]model.Device, error)
	SearchDatacenters(query string) ([]model.Datacenter, error)
}

// NetworkStorage defines network persistence operations
type NetworkStorage interface {
	ListNetworks(filter *model.NetworkFilter) ([]model.Network, error)
	GetNetwork(id string) (*model.Network, error)
	CreateNetwork(ctx context.Context, network *model.Network) error
	UpdateNetwork(ctx context.Context, network *model.Network) error
	DeleteNetwork(ctx context.Context, id string) error
	GetNetworkDevices(networkID string) ([]model.Device, error)
	GetNetworkUtilization(networkID string) (*model.NetworkUtilization, error)
	SearchNetworks(query string) ([]model.Network, error)
}

// NetworkPoolStorage defines network pool persistence operations
type NetworkPoolStorage interface {
	CreateNetworkPool(ctx context.Context, pool *model.NetworkPool) error
	UpdateNetworkPool(ctx context.Context, pool *model.NetworkPool) error
	DeleteNetworkPool(ctx context.Context, id string) error
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
	AddRelationship(ctx context.Context, parentID, childID, relationshipType, notes string) error
	RemoveRelationship(ctx context.Context, parentID, childID, relationshipType string) error
	GetRelationships(deviceID string) ([]model.DeviceRelationship, error)
	ListAllRelationships() ([]model.DeviceRelationship, error)
	GetRelatedDevices(deviceID, relationshipType string) ([]model.Device, error)
	UpdateRelationshipNotes(ctx context.Context, parentID, childID, relationshipType, notes string) error
}

// DiscoveryStorage defines discovery persistence operations
type DiscoveryStorage interface {
	// Discovered devices
	CreateDiscoveredDevice(ctx context.Context, device *model.DiscoveredDevice) error
	UpdateDiscoveredDevice(ctx context.Context, device *model.DiscoveredDevice) error
	GetDiscoveredDevice(id string) (*model.DiscoveredDevice, error)
	GetDiscoveredDeviceByIP(networkID, ip string) (*model.DiscoveredDevice, error)
	ListDiscoveredDevices(networkID string) ([]model.DiscoveredDevice, error)
	DeleteDiscoveredDevice(ctx context.Context, id string) error
	DeleteDiscoveredDevicesByNetwork(ctx context.Context, networkID string) error
	PromoteDiscoveredDevice(ctx context.Context, discoveredID, deviceID string) error

	// Discovery scans
	CreateDiscoveryScan(ctx context.Context, scan *model.DiscoveryScan) error
	UpdateDiscoveryScan(ctx context.Context, scan *model.DiscoveryScan) error
	GetDiscoveryScan(id string) (*model.DiscoveryScan, error)
	ListDiscoveryScans(networkID string) ([]model.DiscoveryScan, error)
	DeleteDiscoveryScan(ctx context.Context, id string) error

	// Discovery rules
	GetDiscoveryRule(id string) (*model.DiscoveryRule, error)
	GetDiscoveryRuleByNetwork(networkID string) (*model.DiscoveryRule, error)
	SaveDiscoveryRule(ctx context.Context, rule *model.DiscoveryRule) error
	ListDiscoveryRules() ([]model.DiscoveryRule, error)
	DeleteDiscoveryRule(ctx context.Context, id string) error

	// Cleanup
	CleanupOldDiscoveries(olderThanDays int) error
}

// BulkOperations defines bulk operation methods
type BulkOperations interface {
	BulkCreateDevices(ctx context.Context, devices []*model.Device) (*BulkResult, error)
	BulkUpdateDevices(ctx context.Context, devices []*model.Device) (*BulkResult, error)
	BulkDeleteDevices(ctx context.Context, ids []string) (*BulkResult, error)
	BulkAddTags(ctx context.Context, deviceIDs []string, tags []string) (*BulkResult, error)
	BulkRemoveTags(ctx context.Context, deviceIDs []string, tags []string) (*BulkResult, error)
	BulkCreateNetworks(ctx context.Context, networks []*model.Network) (*BulkResult, error)
	BulkDeleteNetworks(ctx context.Context, ids []string) (*BulkResult, error)
}

// AuditStorage defines audit log persistence operations
type AuditStorage interface {
	CreateAuditLog(log *model.AuditLog) error
	ListAuditLogs(filter *model.AuditFilter) ([]model.AuditLog, error)
	GetAuditLog(id string) (*model.AuditLog, error)
	DeleteOldAuditLogs(olderThanDays int) error
}

// OAuthStorage defines OAuth 2.1 persistence operations
type OAuthStorage interface {
	// Clients
	CreateOAuthClient(ctx context.Context, client *model.OAuthClient) error
	GetOAuthClient(clientID string) (*model.OAuthClient, error)
	ListOAuthClients(createdByUserID string) ([]model.OAuthClient, error)
	DeleteOAuthClient(ctx context.Context, clientID string) error

	// Authorization codes
	CreateAuthorizationCode(ctx context.Context, code *model.OAuthAuthorizationCode) error
	GetAuthorizationCode(codeHash string) (*model.OAuthAuthorizationCode, error)
	MarkAuthorizationCodeUsed(codeHash string) error
	CleanupExpiredCodes() error

	// Tokens
	CreateOAuthToken(ctx context.Context, token *model.OAuthToken) error
	GetOAuthTokenByHash(tokenHash string) (*model.OAuthToken, error)
	RevokeOAuthToken(tokenID string) error
	RevokeOAuthTokensByClient(clientID string) error
	RevokeOAuthTokensByUser(userID string) error
	CleanupExpiredTokens() error
}

// ConflictStorage defines conflict persistence operations
type ConflictStorage interface {
	// Conflicts
	CreateConflict(ctx context.Context, conflict *model.Conflict) error
	GetConflict(id string) (*model.Conflict, error)
	ListConflicts(filter *model.ConflictFilter) ([]model.Conflict, error)
	UpdateConflictStatus(ctx context.Context, id string, status model.ConflictStatus, resolvedBy, notes string) error
	DeleteConflict(ctx context.Context, id string) error

	// Detection helpers
	FindDuplicateIPs(ctx context.Context) ([]model.Conflict, error)
	FindOverlappingSubnets(ctx context.Context) ([]model.Conflict, error)
	GetConflictsByDeviceID(deviceID string) ([]model.Conflict, error)
	GetConflictsByIP(ip string) ([]model.Conflict, error)
	MarkConflictsResolvedForDevice(ctx context.Context, deviceID string, resolvedBy string) error
}

// ReservationStorage defines reservation persistence operations
type ReservationStorage interface {
	CreateReservation(ctx context.Context, reservation *model.Reservation) error
	GetReservation(id string) (*model.Reservation, error)
	GetReservationByIP(poolID, ip string) (*model.Reservation, error)
	ListReservations(filter *model.ReservationFilter) ([]model.Reservation, error)
	UpdateReservation(ctx context.Context, reservation *model.Reservation) error
	DeleteReservation(ctx context.Context, id string) error
	GetReservationsByPool(poolID string) ([]model.Reservation, error)
	GetReservationsByUser(userID string) ([]model.Reservation, error)
	ExpireReservations(ctx context.Context) (int64, error)
	IsIPReserved(poolID, ip string) (bool, error)
}

// SnapshotStorage defines utilization snapshot operations
type SnapshotStorage interface {
	// Snapshot operations
	CreateSnapshot(ctx context.Context, snapshot *model.UtilizationSnapshot) error
	ListSnapshots(filter *model.SnapshotFilter) ([]model.UtilizationSnapshot, error)
	GetLatestSnapshots(snapshotType model.SnapshotType) ([]model.UtilizationSnapshot, error)
	DeleteOldSnapshots(olderThanDays int) error

	// Dashboard operations
	GetDashboardStats(staleDays int, recentLimit int) (*model.DashboardStats, error)
	GetUtilizationTrend(resourceType model.SnapshotType, resourceID string, days int) ([]model.UtilizationTrendPoint, error)
}

// WebhookStorage defines webhook persistence operations
type WebhookStorage interface {
	// Webhooks
	CreateWebhook(ctx context.Context, webhook *model.Webhook) error
	GetWebhook(id string) (*model.Webhook, error)
	ListWebhooks(filter *model.WebhookFilter) ([]model.Webhook, error)
	UpdateWebhook(ctx context.Context, webhook *model.Webhook) error
	DeleteWebhook(ctx context.Context, id string) error
	GetWebhooksForEvent(eventType model.EventType) ([]model.Webhook, error)

	// Deliveries
	CreateDelivery(ctx context.Context, delivery *model.WebhookDelivery) error
	GetDelivery(id string) (*model.WebhookDelivery, error)
	ListDeliveries(filter *model.DeliveryFilter) ([]model.WebhookDelivery, error)
	UpdateDelivery(ctx context.Context, delivery *model.WebhookDelivery) error
	DeleteOldDeliveries(olderThanDays int) error
	GetPendingDeliveries(limit int) ([]model.WebhookDelivery, error)
}

// CustomFieldStorage defines custom field persistence operations
type CustomFieldStorage interface {
	// Definitions
	CreateCustomFieldDefinition(ctx context.Context, def *model.CustomFieldDefinition) error
	GetCustomFieldDefinition(id string) (*model.CustomFieldDefinition, error)
	GetCustomFieldDefinitionByKey(key string) (*model.CustomFieldDefinition, error)
	ListCustomFieldDefinitions(filter *model.CustomFieldDefinitionFilter) ([]model.CustomFieldDefinition, error)
	UpdateCustomFieldDefinition(ctx context.Context, def *model.CustomFieldDefinition) error
	DeleteCustomFieldDefinition(ctx context.Context, id string) error

	// Values
	SetCustomFieldValue(ctx context.Context, value *model.CustomFieldValue) error
	GetCustomFieldValues(deviceID string) ([]model.CustomFieldValue, error)
	GetCustomFieldValue(deviceID, fieldID string) (*model.CustomFieldValue, error)
	DeleteCustomFieldValue(ctx context.Context, deviceID, fieldID string) error
	DeleteCustomFieldValuesForDevice(ctx context.Context, deviceID string) error
	DeleteCustomFieldValuesForDefinition(ctx context.Context, fieldID string) error
	GetCustomFieldValuesWithDefinitions(deviceID string) ([]model.CustomFieldWithDefinition, error)
}

// CircuitStorage defines circuit persistence operations
type CircuitStorage interface {
	CreateCircuit(ctx context.Context, circuit *model.Circuit) error
	GetCircuit(id string) (*model.Circuit, error)
	GetCircuitByCircuitID(circuitID string) (*model.Circuit, error)
	ListCircuits(filter *model.CircuitFilter) ([]model.Circuit, error)
	UpdateCircuit(ctx context.Context, circuit *model.Circuit) error
	DeleteCircuit(ctx context.Context, id string) error
	GetCircuitsByDatacenter(datacenterID string) ([]model.Circuit, error)
	GetCircuitsByDevice(deviceID string) ([]model.Circuit, error)
}

// NATStorage defines NAT mapping persistence operations
type NATStorage interface {
	CreateNATMapping(ctx context.Context, mapping *model.NATMapping) error
	GetNATMapping(id string) (*model.NATMapping, error)
	ListNATMappings(filter *model.NATFilter) ([]model.NATMapping, error)
	UpdateNATMapping(ctx context.Context, mapping *model.NATMapping) error
	DeleteNATMapping(ctx context.Context, id string) error
	GetNATMappingsByDevice(deviceID string) ([]model.NATMapping, error)
	GetNATMappingsByDatacenter(datacenterID string) ([]model.NATMapping, error)
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
	APIKeyStorage
	BulkOperations
	AuditStorage
	UserStorage
	RBACStorage
	OAuthStorage
	ConflictStorage
	ReservationStorage
	SnapshotStorage
	WebhookStorage
	CustomFieldStorage
	CircuitStorage
	NATStorage
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
