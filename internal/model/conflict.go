package model

import "time"

// ConflictType represents the type of IP conflict
type ConflictType string

const (
	ConflictTypeDuplicateIP    ConflictType = "duplicate_ip"
	ConflictTypeOverlappingSubnet ConflictType = "overlapping_subnet"
)

// ConflictStatus represents the resolution status of a conflict
type ConflictStatus string

const (
	ConflictStatusActive   ConflictStatus = "active"
	ConflictStatusResolved ConflictStatus = "resolved"
	ConflictStatusIgnored ConflictStatus = "ignored"
)

// Conflict represents an IP address or subnet conflict
type Conflict struct {
	ID          string        `json:"id"`
	Type        ConflictType  `json:"type"`
	Status      ConflictStatus `json:"status"`
	Description string        `json:"description"`

	// For duplicate IP conflicts
	IPAddress    string   `json:"ip_address,omitempty"`
	DeviceIDs    []string `json:"device_ids,omitempty"`
	DeviceNames  []string `json:"device_names,omitempty"`

	// For overlapping subnet conflicts
	NetworkIDs   []string `json:"network_ids,omitempty"`
	NetworkNames []string `json:"network_names,omitempty"`
	Subnets      []string `json:"subnets,omitempty"`

	// Metadata
	DetectedAt  time.Time  `json:"detected_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy   string     `json:"resolved_by,omitempty"`
	Notes        string     `json:"notes,omitempty"`
}

// ConflictFilter defines filter criteria for listing conflicts
type ConflictFilter struct {
	Pagination
	Type   ConflictType
	Status ConflictStatus
}

// ConflictResolution represents a request to resolve a conflict
type ConflictResolution struct {
	ConflictID string `json:"conflict_id"`
	// For duplicate IP: which device keeps the IP
	KeepDeviceID string `json:"keep_device_id"`
	// For overlapping subnet: which network is correct
	KeepNetworkID string `json:"keep_network_id"`
	Notes         string `json:"notes"`
}
