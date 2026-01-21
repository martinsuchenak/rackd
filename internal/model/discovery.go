package model

import "time"

type DiscoveredDevice struct {
	ID                 string        `json:"id"`
	IP                 string        `json:"ip"`
	MACAddress         string        `json:"mac_address"`
	Hostname           string        `json:"hostname"`
	NetworkID          string        `json:"network_id"`
	Status             string        `json:"status"`
	Confidence         int           `json:"confidence"`
	OSGuess            string        `json:"os_guess"`
	Vendor             string        `json:"vendor"`
	OpenPorts          []int         `json:"open_ports"`
	Services           []ServiceInfo `json:"services"`
	FirstSeen          time.Time     `json:"first_seen"`
	LastSeen           time.Time     `json:"last_seen"`
	PromotedToDeviceID string        `json:"promoted_to_device_id,omitempty"`
	PromotedAt         *time.Time    `json:"promoted_at,omitempty"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

type ServiceInfo struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Service  string `json:"service"`
	Version  string `json:"version"`
}

type DiscoveryScan struct {
	ID              string     `json:"id"`
	NetworkID       string     `json:"network_id"`
	Status          string     `json:"status"`
	ScanType        string     `json:"scan_type"`
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

type DiscoveryRule struct {
	ID            string    `json:"id"`
	NetworkID     string    `json:"network_id"`
	Enabled       bool      `json:"enabled"`
	ScanType      string    `json:"scan_type"`
	IntervalHours int       `json:"interval_hours"`
	ExcludeIPs    string    `json:"exclude_ips"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

const (
	ScanTypeQuick = "quick"
	ScanTypeFull  = "full"
	ScanTypeDeep  = "deep"
)

const (
	ScanStatusPending   = "pending"
	ScanStatusRunning   = "running"
	ScanStatusCompleted = "completed"
	ScanStatusFailed    = "failed"
)
