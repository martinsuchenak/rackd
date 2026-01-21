# Data Models

This document defines all data structures used in Rackd.

## Device Model

**File**: `internal/model/device.go`

```go
package model

import "time"

// Device represents a tracked device with all its properties
type Device struct {
    ID           string    `json:"id"`
    Name         string    `json:"name"`
    Description  string    `json:"description"`
    MakeModel    string    `json:"make_model"`
    OS           string    `json:"os"`
    DatacenterID string    `json:"datacenter_id,omitempty"`
    Username     string    `json:"username,omitempty"`
    Location     string    `json:"location,omitempty"`
    Tags         []string  `json:"tags"`
    Addresses    []Address `json:"addresses"`
    Domains      []string  `json:"domains"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

// Address represents a network address for a device
type Address struct {
    IP         string `json:"ip"`
    Port       int    `json:"port"`
    Type       string `json:"type"`                  // "ipv4", "ipv6"
    Label      string `json:"label"`                 // e.g., "management", "data"
    NetworkID  string `json:"network_id,omitempty"`  // Network this IP belongs to
    SwitchPort string `json:"switch_port,omitempty"` // Switch port (e.g., "Gi1/0/1")
    PoolID     string `json:"pool_id,omitempty"`     // Pool this IP belongs to
}

// DeviceFilter holds filter criteria for listing devices
type DeviceFilter struct {
    Tags         []string // Filter by tags (OR logic)
    DatacenterID string   // Filter by datacenter
    NetworkID    string   // Filter by network
}
```

## Datacenter Model

**File**: `internal/model/datacenter.go`

```go
package model

import "time"

type Datacenter struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Location    string    `json:"location"`
    Description string    `json:"description"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type DatacenterFilter struct {
    Name string
}
```

## Network Model

**File**: `internal/model/network.go`

```go
package model

import "time"

type Network struct {
    ID           string    `json:"id"`
    Name         string    `json:"name"`
    Subnet       string    `json:"subnet"`       // CIDR notation (e.g., "192.168.1.0/24")
    VLANID       int       `json:"vlan_id"`      // VLAN identifier
    DatacenterID string    `json:"datacenter_id"`
    Description  string    `json:"description"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

type NetworkPool struct {
    ID          string    `json:"id"`
    NetworkID   string    `json:"network_id"`
    Name        string    `json:"name"`
    StartIP     string    `json:"start_ip"`
    EndIP       string    `json:"end_ip"`
    Description string    `json:"description"`
    Tags        []string  `json:"tags"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type NetworkFilter struct {
    Name         string
    DatacenterID string
    VLANID       int
}

type NetworkPoolFilter struct {
    NetworkID string
    Tags      []string
}

// NetworkUtilization represents IP usage statistics for a network
type NetworkUtilization struct {
    NetworkID    string  `json:"network_id"`
    TotalIPs     int     `json:"total_ips"`
    UsedIPs      int     `json:"used_ips"`
    AvailableIPs int     `json:"available_ips"`
    Utilization  float64 `json:"utilization"` // Percentage 0-100
}
```

## Relationship Model

**File**: `internal/model/relationship.go`

```go
package model

import "time"

// DeviceRelationship represents a parent-child relationship between devices
type DeviceRelationship struct {
    ParentID  string    `json:"parent_id"`
    ChildID   string    `json:"child_id"`
    Type      string    `json:"type"` // "contains", "connected_to", "depends_on"
    CreatedAt time.Time `json:"created_at"`
}

// Relationship types
const (
    RelationshipContains    = "contains"     // Physical containment (chassis -> blade)
    RelationshipConnectedTo = "connected_to" // Network connection
    RelationshipDependsOn   = "depends_on"   // Service dependency
)
```

## Discovery Model

**File**: `internal/model/discovery.go`

```go
package model

import "time"

// DiscoveredDevice represents a device found during network discovery
type DiscoveredDevice struct {
    ID                 string       `json:"id"`
    IP                 string       `json:"ip"`
    MACAddress         string       `json:"mac_address"`
    Hostname           string       `json:"hostname"`
    NetworkID          string       `json:"network_id"`
    Status             string       `json:"status"`     // "online", "offline", "unknown"
    Confidence         int          `json:"confidence"` // 0-100
    OSGuess            string       `json:"os_guess"`
    Vendor             string       `json:"vendor"` // MAC vendor lookup
    OpenPorts          []int        `json:"open_ports"`
    Services           []ServiceInfo `json:"services"`
    FirstSeen          time.Time    `json:"first_seen"`
    LastSeen           time.Time    `json:"last_seen"`
    PromotedToDeviceID string       `json:"promoted_to_device_id,omitempty"`
    PromotedAt         *time.Time   `json:"promoted_at,omitempty"`
    CreatedAt          time.Time    `json:"created_at"`
    UpdatedAt          time.Time    `json:"updated_at"`
}

// ServiceInfo represents a detected service on a port
type ServiceInfo struct {
    Port     int    `json:"port"`
    Protocol string `json:"protocol"` // "tcp", "udp"
    Service  string `json:"service"`  // "ssh", "http", etc.
    Version  string `json:"version"`
}

// DiscoveryScan represents a discovery scan job
type DiscoveryScan struct {
    ID              string     `json:"id"`
    NetworkID       string     `json:"network_id"`
    Status          string     `json:"status"` // "pending", "running", "completed", "failed"
    ScanType        string     `json:"scan_type"` // "quick", "full", "deep"
    TotalHosts      int        `json:"total_hosts"`
    ScannedHosts    int        `json:"scanned_hosts"`
    FoundHosts      int        `json:"found_hosts"`
    ProgressPercent float64    `json:"progress_percent"`
    ErrorMessage    string     `json:"error_message,omitempty"`
    StartedAt       *time.Time `json:"started_at,omitempty"`
    CompletedAt     *time.Time `json:"completed_at,omitempty"`
    CreatedAt       time.Time  `json:"created_at"`
    UpdatedAt       time.Time  `json:"updated_at"`
}

// DiscoveryRule defines how discovery should work for a network
type DiscoveryRule struct {
    ID            string `json:"id"`
    NetworkID     string `json:"network_id"`
    Enabled       bool   `json:"enabled"`
    ScanType      string `json:"scan_type"`      // "quick", "full", "deep"
    IntervalHours int    `json:"interval_hours"` // How often to scan
    ExcludeIPs    string `json:"exclude_ips"`    // Comma-separated IPs to skip
    CreatedAt     time.Time `json:"created_at"`
    UpdatedAt     time.Time `json:"updated_at"`
}

// Scan types
const (
    ScanTypeQuick = "quick" // Ping only
    ScanTypeFull  = "full"  // Ping + common ports
    ScanTypeDeep  = "deep"  // Full port scan + service detection
)

// Scan statuses
const (
    ScanStatusPending   = "pending"
    ScanStatusRunning   = "running"
    ScanStatusCompleted = "completed"
    ScanStatusFailed    = "failed"
)
```
