package model

import "time"

// DNSProviderType represents the type of DNS provider
type DNSProviderType string

const (
	DNSProviderTypeTechnitium DNSProviderType = "technitium"
	DNSProviderTypePowerDNS   DNSProviderType = "powerdns"
	DNSProviderTypeBIND       DNSProviderType = "bind"
)

// ValidDNSProviderTypes contains all valid DNS provider types
var ValidDNSProviderTypes = []DNSProviderType{
	DNSProviderTypeTechnitium,
	DNSProviderTypePowerDNS,
	DNSProviderTypeBIND,
}

// IsValid checks if the provider type is valid
func (t DNSProviderType) IsValid() bool {
	for _, pt := range ValidDNSProviderTypes {
		if t == pt {
			return true
		}
	}
	return false
}

// String returns the string representation of the provider type
func (t DNSProviderType) String() string {
	return string(t)
}

// DNSProviderConfig represents a DNS provider configuration
type DNSProviderConfig struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Type        DNSProviderType `json:"type"`
	Endpoint    string          `json:"endpoint"`
	Token       string          `json:"-"` // Write-only, never exposed in JSON
	Description string          `json:"description"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// DNSProvider is an alias for DNSProviderConfig for storage interface compatibility
type DNSProvider = DNSProviderConfig

// DNSProviderFilter for filtering DNS providers
type DNSProviderFilter struct {
	Type DNSProviderType
}

// DNSProviderConfigInput represents input for creating/updating provider config (non-pointer fields)
type DNSProviderConfigInput struct {
	Name        string          `json:"name"`
	Type        DNSProviderType `json:"type"`
	Endpoint    string          `json:"endpoint"`
	Token       string          `json:"token"`
	Description string          `json:"description"`
}

// SyncStatus represents the status of a DNS sync operation
type SyncStatus string

const (
	SyncStatusSuccess SyncStatus = "success"
	SyncStatusFailed  SyncStatus = "failed"
	SyncStatusPartial SyncStatus = "partial"
)

// ValidSyncStatuses contains all valid sync statuses
var ValidSyncStatuses = []SyncStatus{
	SyncStatusSuccess,
	SyncStatusFailed,
	SyncStatusPartial,
}

// IsValid checks if the sync status is valid
func (s SyncStatus) IsValid() bool {
	for _, status := range ValidSyncStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// String returns the string representation of the sync status
func (s SyncStatus) String() string {
	return string(s)
}

// DNSRecordType represents the type of DNS record
type DNSRecordType string

const (
	DNSRecordTypeA     DNSRecordType = "A"
	DNSRecordTypeAAAA  DNSRecordType = "AAAA"
	DNSRecordTypeCNAME DNSRecordType = "CNAME"
	DNSRecordTypeMX    DNSRecordType = "MX"
	DNSRecordTypeTXT   DNSRecordType = "TXT"
	DNSRecordTypeNS    DNSRecordType = "NS"
	DNSRecordTypeSOA   DNSRecordType = "SOA"
	DNSRecordTypePTR   DNSRecordType = "PTR"
	DNSRecordTypeSRV   DNSRecordType = "SRV"
)

// IsValid checks if the DNS record type is valid
func (t DNSRecordType) IsValid() bool {
	validTypes := []DNSRecordType{
		DNSRecordTypeA, DNSRecordTypeAAAA, DNSRecordTypeCNAME,
		DNSRecordTypeMX, DNSRecordTypeTXT, DNSRecordTypeNS,
		DNSRecordTypeSOA, DNSRecordTypePTR, DNSRecordTypeSRV,
	}
	for _, rt := range validTypes {
		if t == rt {
			return true
		}
	}
	return false
}

// String returns the string representation of the DNS record type
func (t DNSRecordType) String() string {
	return string(t)
}

// RecordSyncStatus represents the sync status of an individual DNS record
type RecordSyncStatus string

const (
	RecordSyncStatusSynced  RecordSyncStatus = "synced"
	RecordSyncStatusPending RecordSyncStatus = "pending"
	RecordSyncStatusFailed  RecordSyncStatus = "failed"
)

// ValidRecordSyncStatuses contains all valid record sync statuses
var ValidRecordSyncStatuses = []RecordSyncStatus{
	RecordSyncStatusSynced,
	RecordSyncStatusPending,
	RecordSyncStatusFailed,
}

// IsValid checks if the record sync status is valid
func (s RecordSyncStatus) IsValid() bool {
	for _, status := range ValidRecordSyncStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// String returns the string representation of the record sync status
func (s RecordSyncStatus) String() string {
	return string(s)
}

// DNSZone represents a DNS zone
type DNSZone struct {
	ID             string           `json:"id"`
	Name           string           `json:"name"`
	ProviderID     string           `json:"provider_id"`
	NetworkID      *string          `json:"network_id,omitempty"`
	AutoSync       bool             `json:"auto_sync"`
	CreatePTR      bool             `json:"create_ptr"`
	PTRZone        *string          `json:"ptr_zone,omitempty"`
	TTL            int              `json:"ttl"`
	Description    string           `json:"description"`
	LastSyncAt     *time.Time       `json:"last_sync_at,omitempty"`
	LastSyncStatus SyncStatus       `json:"last_sync_status"`
	LastSyncError  *string          `json:"last_sync_error,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// DNSZoneInput represents input for creating/updating a zone (non-pointer fields)
type DNSZoneInput struct {
	Name        string   `json:"name"`
	ProviderID  string   `json:"provider_id"`
	NetworkID   *string  `json:"network_id,omitempty"`
	AutoSync    bool     `json:"auto_sync"`
	CreatePTR   bool     `json:"create_ptr"`
	PTRZone     *string  `json:"ptr_zone,omitempty"`
	TTL         int      `json:"ttl"`
	Description string   `json:"description"`
}

// DNSRecord represents a DNS record
type DNSRecord struct {
	ID           string           `json:"id"`
	ZoneID       string           `json:"zone_id"`
	DeviceID     *string          `json:"device_id,omitempty"`
	Name         string           `json:"name"`
	Type         string           `json:"type"`
	Value        string           `json:"value"`
	TTL          int              `json:"ttl"`
	SyncStatus   RecordSyncStatus `json:"sync_status"`
	LastSyncAt   *time.Time       `json:"last_sync_at,omitempty"`
	ErrorMessage *string          `json:"error_message,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// DNSZoneFilter for filtering DNS zones
type DNSZoneFilter struct {
	ProviderID string
	NetworkID  *string
	AutoSync   *bool
}

// DNSRecordFilter for filtering DNS records
type DNSRecordFilter struct {
	ZoneID     string
	DeviceID   *string
	Type       string
	SyncStatus *RecordSyncStatus
}

// SyncResult represents the result of a DNS sync operation
type SyncResult struct {
	Success     bool     `json:"success"`
	Total       int      `json:"total"`
	Synced      int      `json:"synced"`
	Failed      int      `json:"failed"`
	Error       string   `json:"error,omitempty"`
	FailedIDs   []string `json:"failed_ids,omitempty"`
}

// ImportResult represents the result of a DNS import operation
type ImportResult struct {
	Success    bool     `json:"success"`
	Total      int      `json:"total"`
	Imported   int      `json:"imported"`
	Skipped    int      `json:"skipped"`
	Failed     int      `json:"failed"`
	Error      string   `json:"error,omitempty"`
	SkippedIDs []string `json:"skipped_ids,omitempty"`
	FailedIDs  []string `json:"failed_ids,omitempty"`
}

// CreateDNSProviderRequest represents the input for creating a DNS provider
type CreateDNSProviderRequest struct {
	Name        string          `json:"name"`
	Type        DNSProviderType `json:"type"`
	Endpoint    string          `json:"endpoint"`
	Token       string          `json:"token"`
	Description string          `json:"description"`
}

// UpdateDNSProviderRequest represents the input for updating a DNS provider
type UpdateDNSProviderRequest struct {
	Name        *string          `json:"name,omitempty"`
	Type        *DNSProviderType `json:"type,omitempty"`
	Endpoint    *string          `json:"endpoint,omitempty"`
	Token       *string          `json:"token,omitempty"`
	Description *string          `json:"description,omitempty"`
}

// CreateDNSZoneRequest represents the input for creating a DNS zone
type CreateDNSZoneRequest struct {
	Name        string  `json:"name"`
	ProviderID  string  `json:"provider_id"`
	NetworkID   *string `json:"network_id,omitempty"`
	AutoSync    bool    `json:"auto_sync"`
	CreatePTR   bool    `json:"create_ptr"`
	PTRZone     *string `json:"ptr_zone,omitempty"`
	TTL         int     `json:"ttl"`
	Description string  `json:"description"`
}

// UpdateDNSZoneRequest represents the input for updating a DNS zone
type UpdateDNSZoneRequest struct {
	Name        *string  `json:"name,omitempty"`
	NetworkID   *string  `json:"network_id,omitempty"`
	AutoSync    *bool    `json:"auto_sync,omitempty"`
	CreatePTR   *bool    `json:"create_ptr,omitempty"`
	PTRZone     *string  `json:"ptr_zone,omitempty"`
	TTL         *int     `json:"ttl,omitempty"`
	Description *string  `json:"description,omitempty"`
}

// CreateDNSRecordRequest represents the input for creating a DNS record
type CreateDNSRecordRequest struct {
	ZoneID   string  `json:"zone_id"`
	DeviceID *string `json:"device_id,omitempty"`
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Value    string  `json:"value"`
	TTL      int     `json:"ttl"`
}

// UpdateDNSRecordRequest represents the input for updating a DNS record
type UpdateDNSRecordRequest struct {
	DeviceID *string `json:"device_id,omitempty"`
	Name     *string `json:"name,omitempty"`
	Type     *string `json:"type,omitempty"`
	Value    *string `json:"value,omitempty"`
	TTL      *int    `json:"ttl,omitempty"`
}
