package model

import "time"

// SnapshotType defines what type of resource the snapshot covers
type SnapshotType string

const (
	SnapshotTypeNetwork SnapshotType = "network"
	SnapshotTypePool    SnapshotType = "pool"
)

// IsValid returns true if the snapshot type is valid
func (s SnapshotType) IsValid() bool {
	return s == SnapshotTypeNetwork || s == SnapshotTypePool
}

// UtilizationSnapshot captures utilization at a point in time
type UtilizationSnapshot struct {
	ID           string       `json:"id"`
	Type         SnapshotType `json:"type"`
	ResourceID   string       `json:"resource_id"`
	ResourceName string       `json:"resource_name"`
	TotalIPs     int          `json:"total_ips"`
	UsedIPs      int          `json:"used_ips"`
	Utilization  float64      `json:"utilization"` // Percentage 0-100
	Timestamp    time.Time    `json:"timestamp"`
	CreatedAt    time.Time    `json:"created_at"`
}

// SnapshotFilter for querying snapshots
type SnapshotFilter struct {
	Pagination
	Type       SnapshotType
	ResourceID string
	After      *time.Time
	Before     *time.Time
}

// DeviceStatusCounts for dashboard device status breakdown
type DeviceStatusCounts struct {
	Planned       int `json:"planned"`
	Active        int `json:"active"`
	Maintenance   int `json:"maintenance"`
	Decommissioned int `json:"decommissioned"`
}

// RecentDiscovery for activity feed
type RecentDiscovery struct {
	ID        string    `json:"id"`
	IP        string    `json:"ip"`
	Hostname  string    `json:"hostname,omitempty"`
	Vendor    string    `json:"vendor,omitempty"`
	NetworkID string    `json:"network_id,omitempty"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

// StaleDevice for health alerts
type StaleDevice struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Hostname string `json:"hostname,omitempty"`
}

// NetworkUtilizationSummary for dashboard network list
type NetworkUtilizationSummary struct {
	NetworkID   string  `json:"network_id"`
	NetworkName string  `json:"network_name"`
	Subnet      string  `json:"subnet"`
	TotalIPs    int     `json:"total_ips"`
	UsedIPs     int     `json:"used_ips"`
	Utilization float64 `json:"utilization"`
}

// UtilizationTrendPoint for trend charts
type UtilizationTrendPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	Utilization float64   `json:"utilization"`
	UsedIPs     int       `json:"used_ips"`
}

// DashboardStats aggregated statistics for the dashboard
type DashboardStats struct {
	// Top-level counts
	TotalDevices     int `json:"total_devices"`
	TotalNetworks    int `json:"total_networks"`
	TotalPools       int `json:"total_pools"`
	TotalDatacenters int `json:"total_datacenters"`

	// Device status breakdown
	DeviceStatusCounts DeviceStatusCounts `json:"device_status_counts"`

	// Discovery stats
	DiscoveredDevices int               `json:"discovered_devices"`
	RecentDiscoveries []RecentDiscovery `json:"recent_discoveries"`

	// Utilization summary
	OverallUtilization float64                     `json:"overall_utilization"`
	NetworkUtilization []NetworkUtilizationSummary `json:"network_utilization"`

	// Stale devices
	StaleDevices       int          `json:"stale_devices"`
	StaleThresholdDays int          `json:"stale_threshold_days"`
	StaleDeviceList    []StaleDevice `json:"stale_device_list"`
}
