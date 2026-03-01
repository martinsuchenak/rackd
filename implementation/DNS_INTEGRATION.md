# DNS Integration - Detailed Implementation Plan

## Overview

This document describes the detailed implementation plan for DNS integration in rackd. The feature allows automatic management of DNS records for devices through external DNS providers, with Technitium DNS as the first supported provider.

## Design Decisions

1. **Token Storage**: Use existing `internal/credentials` package with AES-256-GCM encryption
2. **PTR Zones**: Auto-generate reverse zone name from network subnet (e.g., 192.168.1.0/24 → 1.168.192.in-addr.arpa)
3. **Bidirectional Sync**: Implement import from DNS to rackd
4. **Trigger Model**: Configurable per zone, manual by default

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Rackd Core                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              DNSService (service layer)              │   │
│  │   - SyncZoneToDNS(zoneID)                            │   │
│  │   - SyncDeviceToDNS(deviceID)                        │   │
│  │   - ImportFromDNS(zoneID)                            │   │
│  │   - RBAC enforcement                                 │   │
│  └──────────────────────┬──────────────────────────────┘   │
│                         │                                   │
│  ┌──────────────────────▼──────────────────────────────┐   │
│  │            DNSProvider Interface                     │   │
│  │   CreateRecord(zone, record) error                   │   │
│  │   UpdateRecord(zone, record) error                   │   │
│  │   DeleteRecord(zone, name, type) error               │   │
│  │   ListRecords(zone) ([]DNSRecord, error)             │   │
│  │   HealthCheck() error                                │   │
│  └──────────────────────┬──────────────────────────────┘   │
└─────────────────────────┼───────────────────────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
   ┌────▼────┐      ┌────▼────┐      ┌────▼────┐
   │Technitium│      │ PowerDNS │      │ Future: │
   │  (API)   │      │  (API)   │      │ Built-in│
   └──────────┘      └──────────┘      └─────────┘
```

---

## Phase 1: Core Models and Storage

### 1.1 DNS Models

**File**: `internal/model/dns.go`

```go
package model

import "time"

// DNSProviderType defines the type of DNS provider
type DNSProviderType string

const (
    DNSProviderTechnitium DNSProviderType = "technitium"
    DNSProviderPowerDNS   DNSProviderType = "powerdns"
    DNSProviderBIND       DNSProviderType = "bind"
)

// DNSProviderConfig stores configuration for a DNS provider
type DNSProviderConfig struct {
    ID          string          `json:"id"`
    Name        string          `json:"name"`
    Type        DNSProviderType `json:"type"`
    Endpoint    string          `json:"endpoint"`      // API URL
    Token       string          `json:"-"`             // API token (encrypted, write-only)
    Description string          `json:"description,omitempty"`
    CreatedAt   time.Time       `json:"created_at"`
    UpdatedAt   time.Time       `json:"updated_at"`
}

// DNSProviderConfigInput is used for API requests
type DNSProviderConfigInput struct {
    Name        string          `json:"name"`
    Type        DNSProviderType `json:"type"`
    Endpoint    string          `json:"endpoint"`
    Token       string          `json:"token"`
    Description string          `json:"description"`
}

// SyncStatus represents the status of a sync operation
type SyncStatus string

const (
    SyncStatusSuccess SyncStatus = "success"
    SyncStatusFailed  SyncStatus = "failed"
    SyncStatusPartial SyncStatus = "partial"
)

// RecordSyncStatus represents the sync status of a DNS record
type RecordSyncStatus string

const (
    RecordSyncStatusSynced  RecordSyncStatus = "synced"
    RecordSyncStatusPending RecordSyncStatus = "pending"
    RecordSyncStatusFailed  RecordSyncStatus = "failed"
)

// DNSZone represents a DNS zone mapped to a rackd network
type DNSZone struct {
    ID             string     `json:"id"`
    Name           string     `json:"name"`              // e.g., "internal.example.com"
    ProviderID     string     `json:"provider_id"`       // Link to provider config
    NetworkID      string     `json:"network_id,omitempty"` // Optional: link to rackd network
    AutoSync       bool       `json:"auto_sync"`         // Auto-sync on device changes
    CreatePTR      bool       `json:"create_ptr"`        // Create reverse DNS records
    PTRZone        string     `json:"ptr_zone,omitempty"` // e.g., "1.168.192.in-addr.arpa"
    TTL            int        `json:"ttl"`               // Default TTL for records
    Description    string     `json:"description,omitempty"`
    LastSyncAt     *time.Time `json:"last_sync_at,omitempty"`
    LastSyncStatus SyncStatus `json:"last_sync_status,omitempty"`
    LastSyncError  string     `json:"last_sync_error,omitempty"`
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
}

// DNSZoneInput is used for API requests
type DNSZoneInput struct {
    Name        string `json:"name"`
    ProviderID  string `json:"provider_id"`
    NetworkID   string `json:"network_id"`
    AutoSync    bool   `json:"auto_sync"`
    CreatePTR   bool   `json:"create_ptr"`
    PTRZone     string `json:"ptr_zone"`     // Optional override
    TTL         int    `json:"ttl"`
    Description string `json:"description"`
}

// DNSRecord represents a DNS record tracked in rackd
type DNSRecord struct {
    ID           string           `json:"id"`
    ZoneID       string           `json:"zone_id"`
    DeviceID     string           `json:"device_id,omitempty"` // Link to device
    Name         string           `json:"name"`                // e.g., "server-01"
    Type         string           `json:"type"`                // A, AAAA, CNAME, PTR
    Value        string           `json:"value"`               // IP address or target
    TTL          int              `json:"ttl"`
    SyncStatus   RecordSyncStatus `json:"sync_status"`
    LastSyncAt   *time.Time       `json:"last_sync_at,omitempty"`
    ErrorMessage string           `json:"error_message,omitempty"`
    CreatedAt    time.Time        `json:"created_at"`
    UpdatedAt    time.Time        `json:"updated_at"`
}

// DNSZoneFilter is used for filtering zones
type DNSZoneFilter struct {
    ProviderID string
    NetworkID  string
}

// SyncResult represents the result of a sync operation
type SyncResult struct {
    ZoneID         string    `json:"zone_id"`
    Status         SyncStatus `json:"status"`
    RecordsSynced  int       `json:"records_synced"`
    RecordsFailed  int       `json:"records_failed"`
    RecordsSkipped int       `json:"records_skipped"`
    Errors         []string  `json:"errors,omitempty"`
    SyncedAt       time.Time `json:"synced_at"`
}

// ImportResult represents the result of an import operation
type ImportResult struct {
    ZoneID          string   `json:"zone_id"`
    RecordsImported int      `json:"records_imported"`
    RecordsSkipped  int      `json:"records_skipped"`
    DevicesCreated  int      `json:"devices_created"`
    Errors          []string `json:"errors,omitempty"`
}

// CreateDNSProviderRequest is used for creating a provider
type CreateDNSProviderRequest struct {
    Name        string          `json:"name"`
    Type        DNSProviderType `json:"type"`
    Endpoint    string          `json:"endpoint"`
    Token       string          `json:"token"`
    Description string          `json:"description"`
}

// UpdateDNSProviderRequest is used for updating a provider
type UpdateDNSProviderRequest struct {
    Name        *string          `json:"name,omitempty"`
    Endpoint    *string          `json:"endpoint,omitempty"`
    Token       *string          `json:"token,omitempty"`
    Description *string          `json:"description,omitempty"`
}

// CreateDNSZoneRequest is used for creating a zone
type CreateDNSZoneRequest struct {
    Name        string `json:"name"`
    ProviderID  string `json:"provider_id"`
    NetworkID   string `json:"network_id"`
    AutoSync    bool   `json:"auto_sync"`
    CreatePTR   bool   `json:"create_ptr"`
    PTRZone     string `json:"ptr_zone"`
    TTL         int    `json:"ttl"`
    Description string `json:"description"`
}

// UpdateDNSZoneRequest is used for updating a zone
type UpdateDNSZoneRequest struct {
    Name        *string `json:"name,omitempty"`
    ProviderID  *string `json:"provider_id,omitempty"`
    NetworkID   *string `json:"network_id,omitempty"`
    AutoSync    *bool   `json:"auto_sync,omitempty"`
    CreatePTR   *bool   `json:"create_ptr,omitempty"`
    PTRZone     *string `json:"ptr_zone,omitempty"`
    TTL         *int    `json:"ttl,omitempty"`
    Description *string `json:"description,omitempty"`
}
```

### 1.2 Storage Interface

**File**: `internal/storage/storage.go` (add to existing file)

Add error definitions:
```go
var (
    // ... existing errors ...
    ErrDNSProviderNotFound = errors.New("DNS provider not found")
    ErrDNSZoneNotFound     = errors.New("DNS zone not found")
    ErrDNSRecordNotFound   = errors.New("DNS record not found")
)
```

Add interface:
```go
// DNSStorage defines DNS provider and zone persistence operations
type DNSStorage interface {
    // Providers
    CreateDNSProvider(ctx context.Context, provider *model.DNSProviderConfig) error
    GetDNSProvider(id string) (*model.DNSProviderConfig, error)
    GetDNSProviderByName(name string) (*model.DNSProviderConfig, error)
    ListDNSProviders() ([]model.DNSProviderConfig, error)
    UpdateDNSProvider(ctx context.Context, provider *model.DNSProviderConfig) error
    DeleteDNSProvider(ctx context.Context, id string) error

    // Zones
    CreateDNSZone(ctx context.Context, zone *model.DNSZone) error
    GetDNSZone(id string) (*model.DNSZone, error)
    GetDNSZoneByName(name string) (*model.DNSZone, error)
    ListDNSZones(filter *model.DNSZoneFilter) ([]model.DNSZone, error)
    UpdateDNSZone(ctx context.Context, zone *model.DNSZone) error
    DeleteDNSZone(ctx context.Context, id string) error
    GetDNSZonesByNetwork(networkID string) ([]model.DNSZone, error)
    GetDNSZonesByProvider(providerID string) ([]model.DNSZone, error)

    // Records
    CreateDNSRecord(ctx context.Context, record *model.DNSRecord) error
    GetDNSRecord(id string) (*model.DNSRecord, error)
    ListDNSRecords(zoneID string) ([]model.DNSRecord, error)
    UpdateDNSRecord(ctx context.Context, record *model.DNSRecord) error
    DeleteDNSRecord(ctx context.Context, id string) error
    DeleteDNSRecordsByZone(ctx context.Context, zoneID string) error
    DeleteDNSRecordsByDevice(ctx context.Context, deviceID string) error
    GetDNSRecordsByDevice(deviceID string) ([]model.DNSRecord, error)
    GetDNSRecordByName(zoneID, name, rtype string) (*model.DNSRecord, error)
}
```

Add to ExtendedStorage:
```go
type ExtendedStorage interface {
    // ... existing interfaces ...
    DNSStorage
    // ...
}
```

### 1.3 Database Migration

**File**: `internal/storage/migrations.go` (add new migration)

```go
{
    ID: "20260301000000",
    Name: "add_dns_tables",
    Up: func(db *sql.DB) error {
        // Create dns_provider_configs table
        _, err := db.Exec(`
            CREATE TABLE IF NOT EXISTS dns_provider_configs (
                id TEXT PRIMARY KEY,
                name TEXT NOT NULL UNIQUE,
                type TEXT NOT NULL,
                endpoint TEXT NOT NULL,
                token TEXT NOT NULL,
                description TEXT,
                created_at DATETIME NOT NULL,
                updated_at DATETIME NOT NULL
            )
        `)
        if err != nil {
            return err
        }

        // Create dns_zones table
        _, err = db.Exec(`
            CREATE TABLE IF NOT EXISTS dns_zones (
                id TEXT PRIMARY KEY,
                name TEXT NOT NULL UNIQUE,
                provider_id TEXT NOT NULL,
                network_id TEXT,
                auto_sync INTEGER NOT NULL DEFAULT 0,
                create_ptr INTEGER NOT NULL DEFAULT 1,
                ptr_zone TEXT,
                ttl INTEGER NOT NULL DEFAULT 3600,
                description TEXT,
                last_sync_at DATETIME,
                last_sync_status TEXT,
                last_sync_error TEXT,
                created_at DATETIME NOT NULL,
                updated_at DATETIME NOT NULL,
                FOREIGN KEY (provider_id) REFERENCES dns_provider_configs(id) ON DELETE RESTRICT,
                FOREIGN KEY (network_id) REFERENCES networks(id) ON DELETE SET NULL
            )
        `)
        if err != nil {
            return err
        }

        // Create dns_records table
        _, err = db.Exec(`
            CREATE TABLE IF NOT EXISTS dns_records (
                id TEXT PRIMARY KEY,
                zone_id TEXT NOT NULL,
                device_id TEXT,
                name TEXT NOT NULL,
                type TEXT NOT NULL,
                value TEXT NOT NULL,
                ttl INTEGER NOT NULL,
                sync_status TEXT NOT NULL DEFAULT 'pending',
                last_sync_at DATETIME,
                error_message TEXT,
                created_at DATETIME NOT NULL,
                updated_at DATETIME NOT NULL,
                FOREIGN KEY (zone_id) REFERENCES dns_zones(id) ON DELETE CASCADE,
                FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE SET NULL,
                UNIQUE(zone_id, name, type)
            )
        `)
        if err != nil {
            return err
        }

        // Create indexes
        indexes := []string{
            "CREATE INDEX IF NOT EXISTS idx_dns_zones_provider ON dns_zones(provider_id)",
            "CREATE INDEX IF NOT EXISTS idx_dns_zones_network ON dns_zones(network_id)",
            "CREATE INDEX IF NOT EXISTS idx_dns_records_zone ON dns_records(zone_id)",
            "CREATE INDEX IF NOT EXISTS idx_dns_records_device ON dns_records(device_id)",
        }
        for _, idx := range indexes {
            if _, err := db.Exec(idx); err != nil {
                return err
            }
        }

        // Add RBAC permissions
        permissions := []struct{ id, name, desc string }{
            {"dns-provider:list", "List DNS providers", "View DNS provider configurations"},
            {"dns-provider:read", "View DNS provider", "View DNS provider details"},
            {"dns-provider:create", "Create DNS provider", "Create new DNS provider"},
            {"dns-provider:update", "Update DNS provider", "Modify DNS provider"},
            {"dns-provider:delete", "Delete DNS provider", "Delete DNS provider"},
            {"dns-zone:list", "List DNS zones", "View DNS zones"},
            {"dns-zone:read", "View DNS zone", "View DNS zone details"},
            {"dns-zone:create", "Create DNS zone", "Create new DNS zone"},
            {"dns-zone:update", "Update DNS zone", "Modify DNS zone"},
            {"dns-zone:delete", "Delete DNS zone", "Delete DNS zone"},
            {"dns:sync", "Sync DNS records", "Trigger DNS synchronization"},
            {"dns:import", "Import DNS records", "Import records from DNS server"},
        }

        for _, p := range permissions {
            _, err := db.Exec(`
                INSERT INTO permissions (id, name, description) VALUES (?, ?, ?)
                ON CONFLICT(id) DO NOTHING
            `, p.id, p.name, p.desc)
            if err != nil {
                return err
            }
        }

        // Add permissions to admin role
        adminPerms := []string{
            "dns-provider:list", "dns-provider:read", "dns-provider:create",
            "dns-provider:update", "dns-provider:delete",
            "dns-zone:list", "dns-zone:read", "dns-zone:create",
            "dns-zone:update", "dns-zone:delete",
            "dns:sync", "dns:import",
        }
        for _, permID := range adminPerms {
            _, err := db.Exec(`
                INSERT INTO role_permissions (role_id, permission_id)
                SELECT r.id, p.id FROM roles r, permissions p
                WHERE r.name = 'admin' AND p.id = ?
                ON CONFLICT DO NOTHING
            `, permID)
            if err != nil {
                return err
            }
        }

        // Add read-only permissions to operator and viewer roles
        readOnlyPerms := []string{"dns-provider:list", "dns-provider:read", "dns-zone:list", "dns-zone:read"}
        for _, role := range []string{"operator", "viewer"} {
            for _, permID := range readOnlyPerms {
                _, err := db.Exec(`
                    INSERT INTO role_permissions (role_id, permission_id)
                    SELECT r.id, p.id FROM roles r, permissions p
                    WHERE r.name = ? AND p.id = ?
                    ON CONFLICT DO NOTHING
                `, role, permID)
                if err != nil {
                    return err
                }
            }
        }

        // Add sync permission to operator
        _, err = db.Exec(`
            INSERT INTO role_permissions (role_id, permission_id)
            SELECT r.id, p.id FROM roles r, permissions p
            WHERE r.name = 'operator' AND p.id = 'dns:sync'
            ON CONFLICT DO NOTHING
        `)
        if err != nil {
            return err
        }

        return nil
    },
    Down: func(db *sql.DB) error {
        _, err := db.Exec(`DROP TABLE IF EXISTS dns_records`)
        if err != nil {
            return err
        }
        _, err = db.Exec(`DROP TABLE IF EXISTS dns_zones`)
        if err != nil {
            return err
        }
        _, err = db.Exec(`DROP TABLE IF EXISTS dns_provider_configs`)
        return err
    },
},
```

### 1.4 SQLite Storage Implementation

**File**: `internal/storage/dns_sqlite.go`

Key implementation notes:
- Use `newUUID()` for ID generation
- Handle nullable fields with `sql.NullString` and `sql.NullTime`
- Encrypt tokens before storage (use credentials package encryptor)
- Decrypt tokens on retrieval
- Auto-generate PTR zone from network subnet when CreatePTR is true and PTRZone is empty

---

## Phase 2: Provider Interface and Technitium Implementation

### 2.1 Provider Interface

**File**: `internal/dns/provider.go`

```go
package dns

import (
    "context"
)

// Provider defines the interface for DNS providers
type Provider interface {
    // Name returns the provider type name
    Name() string

    // CreateRecord creates a new DNS record
    CreateRecord(ctx context.Context, zone string, record *Record) error

    // UpdateRecord updates an existing DNS record
    UpdateRecord(ctx context.Context, zone string, record *Record) error

    // DeleteRecord deletes a DNS record
    DeleteRecord(ctx context.Context, zone string, name string, rtype string) error

    // GetRecord retrieves a specific record
    GetRecord(ctx context.Context, zone string, name string, rtype string) (*Record, error)

    // ListRecords lists all records in a zone
    ListRecords(ctx context.Context, zone string) ([]*Record, error)

    // ListZones lists all available zones
    ListZones(ctx context.Context) ([]string, error)

    // ZoneExists checks if a zone exists
    ZoneExists(ctx context.Context, zone string) (bool, error)

    // HealthCheck verifies connectivity
    HealthCheck(ctx context.Context) error
}

// Record represents a DNS record
type Record struct {
    Name     string // Relative name (e.g., "server-01")
    Type     string // A, AAAA, CNAME, PTR, TXT
    Value    string // IP address or target
    TTL      int
    Priority *int   // For MX records
}
```

### 2.2 Technitium Client

**File**: `internal/dns/technitium.go`

Technitium DNS API Reference: https://github.com/TechnitiumSoftware/DnsServer/blob/master/APIDOCS.md

Key API endpoints:
- `GET /api/zones/list?token=xxx` - List all zones
- `GET /api/zones/records/get?token=xxx&zone=example.com` - Get zone records
- `POST /api/records/add?token=xxx&zone=example.com&name=server01&type=A&value=192.168.1.10` - Add record
- `POST /api/records/delete?token=xxx&zone=example.com&name=server01&type=A` - Delete record
- `GET /api/status?token=xxx` - Check server status

Implementation structure:
```go
package dns

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "time"
)

type TechnitiumClient struct {
    endpoint string
    token    string
    client   *http.Client
}

func NewTechnitiumClient(endpoint, token string) *TechnitiumClient {
    return &TechnitiumClient{
        endpoint: strings.TrimSuffix(endpoint, "/"),
        token:    token,
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

func (c *TechnitiumClient) Name() string {
    return "technitium"
}

func (c *TechnitiumClient) doAPI(ctx context.Context, method, path string, params url.Values, result interface{}) error {
    // Build URL with token
    params.Set("token", c.token)
    fullURL := c.endpoint + path + "?" + params.Encode()

    var req *http.Request
    var err error
    if method == "POST" {
        req, err = http.NewRequestWithContext(ctx, "POST", fullURL, nil)
    } else {
        req, err = http.NewRequestWithContext(ctx, "GET", fullURL, nil)
    }
    if err != nil {
        return err
    }

    resp, err := c.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var response struct {
        Status  string          `json:"status"`
        Message string          `json:"errorMessage,omitempty"`
        Response json.RawMessage `json:"response,omitempty"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return err
    }

    if response.Status != "ok" {
        return fmt.Errorf("API error: %s", response.Message)
    }

    if result != nil && response.Response != nil {
        return json.Unmarshal(response.Response, result)
    }
    return nil
}

// Implement all Provider interface methods...
```

---

## Phase 3: Service Layer

### 3.1 DNS Service

**File**: `internal/service/dns.go`

Key responsibilities:
- Provider CRUD with RBAC enforcement
- Zone CRUD with RBAC enforcement
- Sync operations
- Import operations
- Token encryption/decryption

```go
package service

import (
    "context"
    "fmt"

    "github.com/martinsuchenak/rackd/internal/dns"
    "github.com/martinsuchenak/rackd/internal/model"
    "github.com/martinsuchenak/rackd/internal/storage"
)

type DNSService struct {
    store        storage.ExtendedStorage
    encryptor    *credentials.Encryptor
    providerCache map[string]dns.Provider
}

func NewDNSService(store storage.ExtendedStorage, encryptor *credentials.Encryptor) *DNSService {
    return &DNSService{
        store:        store,
        encryptor:    encryptor,
        providerCache: make(map[string]dns.Provider),
    }
}

// getProvider returns a cached or newly created provider client
func (s *DNSService) getProvider(ctx context.Context, providerID string) (dns.Provider, error) {
    // Check cache first
    if p, ok := s.providerCache[providerID]; ok {
        return p, nil
    }

    // Get provider config from storage
    config, err := s.store.GetDNSProvider(providerID)
    if err != nil {
        return nil, err
    }

    // Decrypt token
    token, err := s.encryptor.Decrypt(config.Token)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt token: %w", err)
    }

    // Create provider client based on type
    var p dns.Provider
    switch config.Type {
    case model.DNSProviderTechnitium:
        p = dns.NewTechnitiumClient(config.Endpoint, token)
    default:
        return nil, fmt.Errorf("unsupported provider type: %s", config.Type)
    }

    // Cache the provider
    s.providerCache[providerID] = p
    return p, nil
}

// SyncZone syncs all records for a zone to the DNS provider
func (s *DNSService) SyncZone(ctx context.Context, zoneID string) (*model.SyncResult, error) {
    // RBAC check
    if err := requirePermission(ctx, s.store, "dns", "sync"); err != nil {
        return nil, err
    }

    // Get zone
    zone, err := s.store.GetDNSZone(zoneID)
    if err != nil {
        return nil, err
    }

    // Get provider
    provider, err := s.getProvider(ctx, zone.ProviderID)
    if err != nil {
        return nil, err
    }

    // Get records to sync
    records, err := s.store.ListDNSRecords(zoneID)
    if err != nil {
        return nil, err
    }

    result := &model.SyncResult{
        ZoneID:   zoneID,
        SyncedAt: time.Now(),
        Errors:   []string{},
    }

    // Sync each record
    for _, record := range records {
        dnsRecord := &dns.Record{
            Name:  record.Name,
            Type:  record.Type,
            Value: record.Value,
            TTL:   record.TTL,
        }

        err := provider.CreateRecord(ctx, zone.Name, dnsRecord)
        if err != nil {
            result.RecordsFailed++
            result.Errors = append(result.Errors, fmt.Sprintf("%s.%s: %v", record.Name, record.Type, err))
            record.SyncStatus = model.RecordSyncStatusFailed
            record.ErrorMessage = err.Error()
        } else {
            result.RecordsSynced++
            record.SyncStatus = model.RecordSyncStatusSynced
            now := time.Now()
            record.LastSyncAt = &now
            record.ErrorMessage = ""
        }

        s.store.UpdateDNSRecord(ctx, &record)
    }

    // Update zone sync status
    if result.RecordsFailed > 0 && result.RecordsSynced > 0 {
        result.Status = model.SyncStatusPartial
    } else if result.RecordsFailed > 0 {
        result.Status = model.SyncStatusFailed
    } else {
        result.Status = model.SyncStatusSuccess
    }

    // Update zone
    zone.LastSyncAt = &result.SyncedAt
    zone.LastSyncStatus = result.Status
    if len(result.Errors) > 0 {
        zone.LastSyncError = strings.Join(result.Errors, "; ")
    }
    s.store.UpdateDNSZone(ctx, zone)

    return result, nil
}

// ImportFromDNS imports records from DNS provider
func (s *DNSService) ImportFromDNS(ctx context.Context, zoneID string) (*model.ImportResult, error) {
    // RBAC check
    if err := requirePermission(ctx, s.store, "dns", "import"); err != nil {
        return nil, err
    }

    // Get zone
    zone, err := s.store.GetDNSZone(zoneID)
    if err != nil {
        return nil, err
    }

    // Get provider
    provider, err := s.getProvider(ctx, zone.ProviderID)
    if err != nil {
        return nil, err
    }

    // List records from DNS
    dnsRecords, err := provider.ListRecords(ctx, zone.Name)
    if err != nil {
        return nil, err
    }

    result := &model.ImportResult{
        ZoneID: zoneID,
        Errors: []string{},
    }

    // Import each record
    for _, dr := range dnsRecords {
        // Skip PTR records if not enabled
        if dr.Type == "PTR" && !zone.CreatePTR {
            result.RecordsSkipped++
            continue
        }

        // Check if record already exists
        existing, _ := s.store.GetDNSRecordByName(zoneID, dr.Name, dr.Type)
        if existing != nil {
            result.RecordsSkipped++
            continue
        }

        // Create record
        record := &model.DNSRecord{
            ZoneID:     zoneID,
            Name:       dr.Name,
            Type:       dr.Type,
            Value:      dr.Value,
            TTL:        dr.TTL,
            SyncStatus: model.RecordSyncStatusSynced,
        }

        // Try to link to device by hostname
        if dr.Type == "A" || dr.Type == "AAAA" {
            devices, _ := s.store.SearchDevices(dr.Name)
            if len(devices) > 0 {
                record.DeviceID = devices[0].ID
            }
        }

        if err := s.store.CreateDNSRecord(ctx, record); err != nil {
            result.Errors = append(result.Errors, fmt.Sprintf("%s.%s: %v", dr.Name, dr.Type, err))
        } else {
            result.RecordsImported++
        }
    }

    return result, nil
}

// SyncDeviceToDNS syncs a single device's DNS records
func (s *DNSService) SyncDeviceToDNS(ctx context.Context, deviceID string) error {
    // Get device
    device, err := s.store.GetDevice(deviceID)
    if err != nil {
        return err
    }

    // Skip if no hostname
    if device.Hostname == "" {
        return nil
    }

    // Get all zones that have auto-sync enabled for this device's network
    zones, err := s.store.ListDNSZones(&model.DNSZoneFilter{
        // Could filter by network if device has addresses
    })
    if err != nil {
        return err
    }

    for _, zone := range zones {
        if !zone.AutoSync {
            continue
        }

        // Check if device has IPs in this zone's network
        if zone.NetworkID != "" {
            hasIP := false
            for _, addr := range device.Addresses {
                if addr.NetworkID == zone.NetworkID {
                    hasIP = true
                    break
                }
            }
            if !hasIP {
                continue
            }
        }

        // Get provider
        provider, err := s.getProvider(ctx, zone.ProviderID)
        if err != nil {
            continue
        }

        // Create/update A records for each IP
        for _, addr := range device.Addresses {
            if addr.IP == "" {
                continue
            }

            // Create or update record in local DB
            existing, _ := s.store.GetDNSRecordByName(zone.ID, device.Hostname, "A")

            record := &dns.Record{
                Name:  device.Hostname,
                Type:  "A",
                Value: addr.IP,
                TTL:   zone.TTL,
            }

            if err := provider.CreateRecord(ctx, zone.Name, record); err != nil {
                // Log error but continue
                continue
            }

            // Update local tracking
            if existing != nil {
                existing.Value = addr.IP
                existing.SyncStatus = model.RecordSyncStatusSynced
                now := time.Now()
                existing.LastSyncAt = &now
                s.store.UpdateDNSRecord(ctx, existing)
            } else {
                now := time.Now()
                s.store.CreateDNSRecord(ctx, &model.DNSRecord{
                    ZoneID:     zone.ID,
                    DeviceID:   deviceID,
                    Name:       device.Hostname,
                    Type:       "A",
                    Value:      addr.IP,
                    TTL:        zone.TTL,
                    SyncStatus: model.RecordSyncStatusSynced,
                    LastSyncAt: &now,
                })
            }

            // Create PTR record if enabled
            if zone.CreatePTR && zone.PTRZone != "" {
                ptrRecord := &dns.Record{
                    Name:  extractPTRName(addr.IP),
                    Type:  "PTR",
                    Value: device.Hostname + "." + zone.Name,
                    TTL:   zone.TTL,
                }
                provider.CreateRecord(ctx, zone.PTRZone, ptrRecord)
            }
        }
    }

    return nil
}
```

---

## Phase 4: API Handlers

**File**: `internal/api/dns_handlers.go`

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/dns/providers | List providers |
| POST | /api/dns/providers | Create provider |
| GET | /api/dns/providers/{id} | Get provider |
| PUT | /api/dns/providers/{id} | Update provider |
| DELETE | /api/dns/providers/{id} | Delete provider |
| POST | /api/dns/providers/{id}/test | Test provider connection |
| GET | /api/dns/providers/{id}/zones | List zones for provider |
| GET | /api/dns/zones | List zones |
| POST | /api/dns/zones | Create zone |
| GET | /api/dns/zones/{id} | Get zone |
| PUT | /api/dns/zones/{id} | Update zone |
| DELETE | /api/dns/zones/{id} | Delete zone |
| POST | /api/dns/zones/{id}/sync | Sync zone to DNS |
| POST | /api/dns/zones/{id}/import | Import from DNS |
| GET | /api/dns/zones/{id}/records | List records in zone |
| DELETE | /api/dns/records/{id} | Delete record |

---

## Phase 5: CLI Commands

**File**: `cmd/dns/dns.go`

### Commands

```
rackd dns provider list
rackd dns provider get <id>
rackd dns provider create --name <n> --type technitium --endpoint <url> --token <t>
rackd dns provider update <id> [flags]
rackd dns provider delete <id>
rackd dns provider test <id>

rackd dns zone list [--provider <id>] [--network <id>]
rackd dns zone get <id>
rackd dns zone create --name <zone> --provider <id> [--network <id>] [flags]
rackd dns zone update <id> [flags]
rackd dns zone delete <id>

rackd dns sync <zone-id>
rackd dns import <zone-id>
rackd dns records <zone-id>
```

---

## Phase 6: Web UI

### Files to Create

1. `webui/src/core/types.ts` - Add TypeScript types for DNS
2. `webui/src/core/api.ts` - Add DNS API methods
3. `webui/src/components/dns.ts` - DNS UI components
4. `webui/src/partials/pages/dns-providers.html` - Provider management page
5. `webui/src/partials/pages/dns-zones.html` - Zone management page
6. `webui/src/partials/pages/dns-records.html` - Records view page

### Navigation

Add to `webui/src/index.html`:
- DNS Providers link under Settings/Integrations
- DNS Zones link in main navigation

---

## Utility Functions

### PTR Zone Generation

```go
// generatePTRZone generates a PTR zone name from a CIDR subnet
func generatePTRZone(subnet string) (string, error) {
    // Parse CIDR notation (e.g., 192.168.1.0/24 -> 1.168.192.in-addr.arpa)
    _, ipNet, err := net.ParseCIDR(subnet)
    if err != nil {
        return "", err
    }

    ip := ipNet.IP
    if ip.To4() == nil {
        return "", fmt.Errorf("IPv6 not supported for PTR zone generation")
    }

    // Get prefix length
    ones, _ := ipNet.Mask.Size()

    // Determine number of octets to use
    octets := ones / 8
    if octets == 0 {
        octets = 1
    }

    // Build reverse octets
    parts := strings.Split(ip.String(), ".")
    reverseParts := make([]string, octets)
    for i := 0; i < octets; i++ {
        reverseParts[octets-1-i] = parts[i]
    }

    return strings.Join(reverseParts, ".") + ".in-addr.arpa", nil
}

// extractPTRName extracts the PTR record name from an IP
func extractPTRName(ip string) string {
    parts := strings.Split(ip, ".")
    // Reverse the parts
    for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
        parts[i], parts[j] = parts[j], parts[i]
    }
    return strings.Join(parts, ".") + ".in-addr.arpa"
}
```

---

## Testing Strategy

### Unit Tests

1. `internal/dns/technitium_test.go` - Test Technitium client with mock server
2. `internal/storage/dns_sqlite_test.go` - Test storage operations

### Integration Tests

1. `internal/api/dns_handlers_test.go` - Test API endpoints

### Manual Testing

1. Create Technitium provider, test connection
2. Create zone linked to network
3. Add devices with hostnames
4. Trigger sync, verify records in Technitium
5. Test PTR record creation
6. Test import from DNS
7. Verify RBAC permissions work

---

## Future Enhancements

- Additional providers: PowerDNS, BIND (nsupdate), Pi-hole
- Built-in DNS server option (CoreDNS embed)
- DHCP integration (similar provider pattern)
- DNSSEC support
- Multiple records per device (round-robin)
- SRV record support
- TSIG authentication for BIND
