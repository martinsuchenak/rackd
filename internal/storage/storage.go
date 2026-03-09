package storage

import (
	"context"
	"database/sql"
	"errors"

	"github.com/martinsuchenak/rackd/internal/model"
)

// Predefined errors for storage operations
var (
	ErrDeviceNotFound      = errors.New("device not found")
	ErrInvalidID           = errors.New("invalid ID")
	ErrDatacenterNotFound  = errors.New("datacenter not found")
	ErrNetworkNotFound     = errors.New("network not found")
	ErrPoolNotFound        = errors.New("network pool not found")
	ErrDiscoveryNotFound   = errors.New("discovered device not found")
	ErrScanNotFound        = errors.New("scan not found")
	ErrRuleNotFound        = errors.New("discovery rule not found")
	ErrIPNotAvailable      = errors.New("no IP addresses available")
	ErrIPConflict          = errors.New("IP address already in use")
	ErrAuditLogNotFound    = errors.New("audit log not found")
	ErrUserNotFound        = errors.New("user not found")
	ErrOAuthClientNotFound = errors.New("oauth client not found")
	ErrOAuthCodeNotFound   = errors.New("oauth authorization code not found")
	ErrOAuthCodeExpired    = errors.New("oauth authorization code expired")
	ErrOAuthCodeUsed       = errors.New("oauth authorization code already used")
	ErrOAuthTokenNotFound  = errors.New("oauth token not found")
	ErrOAuthTokenRevoked   = errors.New("oauth token revoked")
	ErrOAuthTokenExpired   = errors.New("oauth token expired")
	ErrReservationNotFound = errors.New("reservation not found")
	ErrReservationExpired  = errors.New("reservation has expired")
	ErrIPAlreadyReserved   = errors.New("IP address is already reserved")
	ErrWebhookNotFound     = errors.New("webhook not found")
	ErrDeliveryNotFound    = errors.New("webhook delivery not found")
	ErrCustomFieldNotFound = errors.New("custom field definition not found")
	ErrDuplicateFieldKey   = errors.New("custom field key already exists")
	ErrCircuitNotFound     = errors.New("circuit not found")
	ErrNATNotFound         = errors.New("NAT mapping not found")
	ErrDNSProviderNotFound = errors.New("DNS provider not found")
	ErrDNSZoneNotFound     = errors.New("DNS zone not found")
	ErrDNSRecordNotFound   = errors.New("DNS record not found")
	ErrAPIKeyNotFound      = errors.New("API key not found")
	ErrRoleNotFound        = errors.New("role not found")
	ErrPermissionNotFound  = errors.New("permission not found")
	ErrConflictNotFound    = errors.New("conflict not found")
)

// DeviceStorage defines device persistence operations
type DeviceStorage interface {
	GetDevice(ctx context.Context, id string) (*model.Device, error)
	CreateDevice(ctx context.Context, device *model.Device) error
	UpdateDevice(ctx context.Context, device *model.Device) error
	DeleteDevice(ctx context.Context, id string) error
	ListDevices(ctx context.Context, filter *model.DeviceFilter) ([]model.Device, error)
	SearchDevices(ctx context.Context, query string) ([]model.Device, error)
	GetDeviceStatusCounts(ctx context.Context) (map[model.DeviceStatus]int, error)
}

// DatacenterStorage defines datacenter persistence operations
type DatacenterStorage interface {
	ListDatacenters(ctx context.Context, filter *model.DatacenterFilter) ([]model.Datacenter, error)
	GetDatacenter(ctx context.Context, id string) (*model.Datacenter, error)
	CreateDatacenter(ctx context.Context, dc *model.Datacenter) error
	UpdateDatacenter(ctx context.Context, dc *model.Datacenter) error
	DeleteDatacenter(ctx context.Context, id string) error
	GetDatacenterDevices(ctx context.Context, datacenterID string) ([]model.Device, error)
	SearchDatacenters(ctx context.Context, query string) ([]model.Datacenter, error)
}

// NetworkStorage defines network persistence operations
type NetworkStorage interface {
	ListNetworks(ctx context.Context, filter *model.NetworkFilter) ([]model.Network, error)
	GetNetwork(ctx context.Context, id string) (*model.Network, error)
	CreateNetwork(ctx context.Context, network *model.Network) error
	UpdateNetwork(ctx context.Context, network *model.Network) error
	DeleteNetwork(ctx context.Context, id string) error
	GetNetworkDevices(ctx context.Context, networkID string) ([]model.Device, error)
	GetNetworkUtilization(ctx context.Context, networkID string) (*model.NetworkUtilization, error)
	SearchNetworks(ctx context.Context, query string) ([]model.Network, error)
}

// NetworkPoolStorage defines network pool persistence operations
type NetworkPoolStorage interface {
	CreateNetworkPool(ctx context.Context, pool *model.NetworkPool) error
	UpdateNetworkPool(ctx context.Context, pool *model.NetworkPool) error
	DeleteNetworkPool(ctx context.Context, id string) error
	GetNetworkPool(ctx context.Context, id string) (*model.NetworkPool, error)
	ListNetworkPools(ctx context.Context, filter *model.NetworkPoolFilter) ([]model.NetworkPool, error)
	GetNextAvailableIP(ctx context.Context, poolID string) (string, error)
	ValidateIPInPool(ctx context.Context, poolID, ip string) (bool, error)
	GetPoolHeatmap(ctx context.Context, poolID string) ([]IPStatus, error)
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
	GetRelationships(ctx context.Context, deviceID string) ([]model.DeviceRelationship, error)
	ListAllRelationships(ctx context.Context) ([]model.DeviceRelationship, error)
	GetRelatedDevices(ctx context.Context, deviceID, relationshipType string) ([]model.Device, error)
	UpdateRelationshipNotes(ctx context.Context, parentID, childID, relationshipType, notes string) error
}

// DiscoveryStorage defines discovery persistence operations
type DiscoveryStorage interface {
	// Discovered devices
	CreateDiscoveredDevice(ctx context.Context, device *model.DiscoveredDevice) error
	UpdateDiscoveredDevice(ctx context.Context, device *model.DiscoveredDevice) error
	GetDiscoveredDevice(ctx context.Context, id string) (*model.DiscoveredDevice, error)
	GetDiscoveredDeviceByIP(ctx context.Context, networkID, ip string) (*model.DiscoveredDevice, error)
	ListDiscoveredDevices(ctx context.Context, networkID string) ([]model.DiscoveredDevice, error)
	DeleteDiscoveredDevice(ctx context.Context, id string) error
	DeleteDiscoveredDevicesByNetwork(ctx context.Context, networkID string) error
	PromoteDiscoveredDevice(ctx context.Context, discoveredID, deviceID string) error

	// Discovery scans
	CreateDiscoveryScan(ctx context.Context, scan *model.DiscoveryScan) error
	UpdateDiscoveryScan(ctx context.Context, scan *model.DiscoveryScan) error
	GetDiscoveryScan(ctx context.Context, id string) (*model.DiscoveryScan, error)
	ListDiscoveryScans(ctx context.Context, networkID string) ([]model.DiscoveryScan, error)
	DeleteDiscoveryScan(ctx context.Context, id string) error

	// Discovery rules
	GetDiscoveryRule(ctx context.Context, id string) (*model.DiscoveryRule, error)
	GetDiscoveryRuleByNetwork(ctx context.Context, networkID string) (*model.DiscoveryRule, error)
	SaveDiscoveryRule(ctx context.Context, rule *model.DiscoveryRule) error
	ListDiscoveryRules(ctx context.Context) ([]model.DiscoveryRule, error)
	DeleteDiscoveryRule(ctx context.Context, id string) error

	// Cleanup
	CleanupOldDiscoveries(ctx context.Context, olderThanDays int) error
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
	CreateAuditLog(ctx context.Context, log *model.AuditLog) error
	ListAuditLogs(ctx context.Context, filter *model.AuditFilter) ([]model.AuditLog, error)
	GetAuditLog(ctx context.Context, id string) (*model.AuditLog, error)
	DeleteOldAuditLogs(ctx context.Context, olderThanDays int) error
}

// OAuthStorage defines OAuth 2.1 persistence operations
type OAuthStorage interface {
	// Clients
	CreateOAuthClient(ctx context.Context, client *model.OAuthClient) error
	GetOAuthClient(ctx context.Context, clientID string) (*model.OAuthClient, error)
	ListOAuthClients(ctx context.Context, createdByUserID string) ([]model.OAuthClient, error)
	DeleteOAuthClient(ctx context.Context, clientID string) error

	// Authorization codes
	CreateAuthorizationCode(ctx context.Context, code *model.OAuthAuthorizationCode) error
	GetAuthorizationCode(ctx context.Context, codeHash string) (*model.OAuthAuthorizationCode, error)
	MarkAuthorizationCodeUsed(ctx context.Context, codeHash string) error
	CleanupExpiredCodes(ctx context.Context) error

	// Tokens
	CreateOAuthToken(ctx context.Context, token *model.OAuthToken) error
	GetOAuthTokenByHash(ctx context.Context, tokenHash string) (*model.OAuthToken, error)
	GetOAuthTokenByHashIncludingRevoked(ctx context.Context, tokenHash string) (*model.OAuthToken, error)
	RevokeOAuthToken(ctx context.Context, tokenID string) error
	RevokeOAuthTokenChain(ctx context.Context, refreshTokenID string) error
	RevokeOAuthTokensByClient(ctx context.Context, clientID string) error
	RevokeOAuthTokensByUser(ctx context.Context, userID string) error
	CleanupExpiredTokens(ctx context.Context) error
}

// ConflictStorage defines conflict persistence operations
type ConflictStorage interface {
	// Conflicts
	CreateConflict(ctx context.Context, conflict *model.Conflict) error
	GetConflict(ctx context.Context, id string) (*model.Conflict, error)
	ListConflicts(ctx context.Context, filter *model.ConflictFilter) ([]model.Conflict, error)
	UpdateConflictStatus(ctx context.Context, id string, status model.ConflictStatus, resolvedBy, notes string) error
	DeleteConflict(ctx context.Context, id string) error

	// Detection helpers
	FindDuplicateIPs(ctx context.Context) ([]model.Conflict, error)
	FindOverlappingSubnets(ctx context.Context) ([]model.Conflict, error)
	GetConflictsByDeviceID(ctx context.Context, deviceID string) ([]model.Conflict, error)
	GetConflictsByIP(ctx context.Context, ip string) ([]model.Conflict, error)
	MarkConflictsResolvedForDevice(ctx context.Context, deviceID string, resolvedBy string) error
}

// ReservationStorage defines reservation persistence operations
type ReservationStorage interface {
	CreateReservation(ctx context.Context, reservation *model.Reservation) error
	GetReservation(ctx context.Context, id string) (*model.Reservation, error)
	GetReservationByIP(ctx context.Context, poolID, ip string) (*model.Reservation, error)
	ListReservations(ctx context.Context, filter *model.ReservationFilter) ([]model.Reservation, error)
	UpdateReservation(ctx context.Context, reservation *model.Reservation) error
	DeleteReservation(ctx context.Context, id string) error
	GetReservationsByPool(ctx context.Context, poolID string) ([]model.Reservation, error)
	GetReservationsByUser(ctx context.Context, userID string) ([]model.Reservation, error)
	ExpireReservations(ctx context.Context) (int64, error)
	IsIPReserved(ctx context.Context, poolID, ip string) (bool, error)
}

// SnapshotStorage defines utilization snapshot operations
type SnapshotStorage interface {
	// Snapshot operations
	CreateSnapshot(ctx context.Context, snapshot *model.UtilizationSnapshot) error
	ListSnapshots(ctx context.Context, filter *model.SnapshotFilter) ([]model.UtilizationSnapshot, error)
	GetLatestSnapshots(ctx context.Context, snapshotType model.SnapshotType) ([]model.UtilizationSnapshot, error)
	DeleteOldSnapshots(ctx context.Context, olderThanDays int) error

	// Dashboard operations
	GetDashboardStats(ctx context.Context, staleDays int, recentLimit int) (*model.DashboardStats, error)
	GetUtilizationTrend(ctx context.Context, resourceType model.SnapshotType, resourceID string, days int) ([]model.UtilizationTrendPoint, error)
}

// WebhookStorage defines webhook persistence operations
type WebhookStorage interface {
	// Webhook operations
	CreateWebhook(ctx context.Context, webhook *model.Webhook) error
	GetWebhook(ctx context.Context, id string) (*model.Webhook, error)
	ListWebhooks(ctx context.Context, filter *model.WebhookFilter) ([]model.Webhook, error)
	GetWebhooksForEvent(ctx context.Context, eventType model.EventType) ([]model.Webhook, error)
	UpdateWebhook(ctx context.Context, webhook *model.Webhook) error
	DeleteWebhook(ctx context.Context, id string) error

	// Delivery operations
	CreateDelivery(ctx context.Context, delivery *model.WebhookDelivery) error
	GetDelivery(ctx context.Context, id string) (*model.WebhookDelivery, error)
	ListDeliveries(ctx context.Context, filter *model.DeliveryFilter) ([]model.WebhookDelivery, error)
	UpdateDelivery(ctx context.Context, delivery *model.WebhookDelivery) error
	DeleteOldDeliveries(ctx context.Context, olderThanDays int) error
	GetPendingDeliveries(ctx context.Context, limit int) ([]model.WebhookDelivery, error)
}

// CustomFieldStorage defines custom field persistence operations
type CustomFieldStorage interface {
	// Definitions
	CreateCustomFieldDefinition(ctx context.Context, def *model.CustomFieldDefinition) error
	GetCustomFieldDefinition(ctx context.Context, id string) (*model.CustomFieldDefinition, error)
	GetCustomFieldDefinitionByKey(ctx context.Context, key string) (*model.CustomFieldDefinition, error)
	ListCustomFieldDefinitions(ctx context.Context, filter *model.CustomFieldDefinitionFilter) ([]model.CustomFieldDefinition, error)
	UpdateCustomFieldDefinition(ctx context.Context, def *model.CustomFieldDefinition) error
	DeleteCustomFieldDefinition(ctx context.Context, id string) error

	// Values
	SetCustomFieldValue(ctx context.Context, value *model.CustomFieldValue) error
	GetCustomFieldValues(ctx context.Context, deviceID string) ([]model.CustomFieldValue, error)
	GetCustomFieldValue(ctx context.Context, deviceID, fieldID string) (*model.CustomFieldValue, error)
	DeleteCustomFieldValue(ctx context.Context, deviceID, fieldID string) error
	DeleteCustomFieldValuesForDevice(ctx context.Context, deviceID string) error
	DeleteCustomFieldValuesForDefinition(ctx context.Context, fieldID string) error
	GetCustomFieldValuesWithDefinitions(ctx context.Context, deviceID string) ([]model.CustomFieldWithDefinition, error)
}

// CircuitStorage defines circuit persistence operations
type CircuitStorage interface {
	CreateCircuit(ctx context.Context, circuit *model.Circuit) error
	GetCircuit(ctx context.Context, id string) (*model.Circuit, error)
	GetCircuitByCircuitID(ctx context.Context, circuitID string) (*model.Circuit, error)
	ListCircuits(ctx context.Context, filter *model.CircuitFilter) ([]model.Circuit, error)
	UpdateCircuit(ctx context.Context, circuit *model.Circuit) error
	DeleteCircuit(ctx context.Context, id string) error
	GetCircuitsByDatacenter(ctx context.Context, datacenterID string) ([]model.Circuit, error)
	GetCircuitsByDevice(ctx context.Context, deviceID string) ([]model.Circuit, error)
}

// NATStorage defines NAT mapping persistence operations
type NATStorage interface {
	CreateNATMapping(ctx context.Context, mapping *model.NATMapping) error
	GetNATMapping(ctx context.Context, id string) (*model.NATMapping, error)
	ListNATMappings(ctx context.Context, filter *model.NATFilter) ([]model.NATMapping, error)
	UpdateNATMapping(ctx context.Context, mapping *model.NATMapping) error
	DeleteNATMapping(ctx context.Context, id string) error
	GetNATMappingsByDevice(ctx context.Context, deviceID string) ([]model.NATMapping, error)
	GetNATMappingsByDatacenter(ctx context.Context, datacenterID string) ([]model.NATMapping, error)
}

// DNSStorage defines DNS persistence operations
type DNSStorage interface {
	// Providers
	CreateDNSProvider(ctx context.Context, provider *model.DNSProviderConfig) error
	GetDNSProvider(ctx context.Context, id string) (*model.DNSProviderConfig, error)
	GetDNSProviderByName(ctx context.Context, name string) (*model.DNSProviderConfig, error)
	ListDNSProviders(ctx context.Context, filter *model.DNSProviderFilter) ([]model.DNSProviderConfig, error)
	UpdateDNSProvider(ctx context.Context, provider *model.DNSProviderConfig) error
	DeleteDNSProvider(ctx context.Context, id string) error

	// Zones
	CreateDNSZone(ctx context.Context, zone *model.DNSZone) error
	GetDNSZone(ctx context.Context, id string) (*model.DNSZone, error)
	GetDNSZoneByName(ctx context.Context, name string) (*model.DNSZone, error)
	ListDNSZones(ctx context.Context, filter *model.DNSZoneFilter) ([]model.DNSZone, error)
	UpdateDNSZone(ctx context.Context, zone *model.DNSZone) error
	DeleteDNSZone(ctx context.Context, id string) error
	GetDNSZonesByNetwork(ctx context.Context, networkID string) ([]model.DNSZone, error)
	GetDNSZonesByProvider(ctx context.Context, providerID string) ([]model.DNSZone, error)

	// Records
	CreateDNSRecord(ctx context.Context, record *model.DNSRecord) error
	GetDNSRecord(ctx context.Context, id string) (*model.DNSRecord, error)
	ListDNSRecords(ctx context.Context, filter *model.DNSRecordFilter) ([]model.DNSRecord, error)
	UpdateDNSRecord(ctx context.Context, record *model.DNSRecord) error
	DeleteDNSRecord(ctx context.Context, id string) error
	DeleteDNSRecordsByZone(ctx context.Context, zoneID string) error
	DeleteDNSRecordsByDevice(ctx context.Context, deviceID string) error
	GetDNSRecordsByDevice(ctx context.Context, deviceID string) ([]model.DNSRecord, error)
	GetDNSRecordByName(ctx context.Context, zoneID, name string, recordType string) (*model.DNSRecord, error)
}

// SSHHostKeyStorage defines SSH host key persistence operations
type SSHHostKeyStorage interface {
	GetSSHHostKey(ctx context.Context, host string) ([]byte, error)
	SaveSSHHostKey(ctx context.Context, host string, key []byte) error
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
	DNSStorage
	SSHHostKeyStorage
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
