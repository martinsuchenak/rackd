package model

import "time"

// DeviceStatus represents the lifecycle status of a device
type DeviceStatus string

const (
	DeviceStatusPlanned       DeviceStatus = "planned"
	DeviceStatusActive        DeviceStatus = "active"
	DeviceStatusMaintenance   DeviceStatus = "maintenance"
	DeviceStatusDecommissioned DeviceStatus = "decommissioned"
)

// ValidDeviceStatuses contains all valid device statuses
var ValidDeviceStatuses = []DeviceStatus{
	DeviceStatusPlanned,
	DeviceStatusActive,
	DeviceStatusMaintenance,
	DeviceStatusDecommissioned,
}

// IsValid checks if the status is a valid device status
func (s DeviceStatus) IsValid() bool {
	for _, status := range ValidDeviceStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// String returns the string representation of the status
func (s DeviceStatus) String() string {
	return string(s)
}

type Device struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	Hostname         string       `json:"hostname,omitempty"`
	Description      string       `json:"description"`
	MakeModel        string       `json:"make_model"`
	OS               string       `json:"os"`
	DatacenterID     string       `json:"datacenter_id,omitempty"`
	Username         string       `json:"username,omitempty"`
	Location         string       `json:"location,omitempty"`
	Status           DeviceStatus `json:"status"`
	DecommissionDate *time.Time   `json:"decommission_date,omitempty"`
	StatusChangedAt  *time.Time   `json:"status_changed_at,omitempty"`
	StatusChangedBy  string       `json:"status_changed_by,omitempty"`
	Tags             []string     `json:"tags"`
	Addresses        []Address    `json:"addresses"`
	Domains          []string     `json:"domains"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

type Address struct {
	IP         string `json:"ip"`
	Port       *int   `json:"port,omitempty"`
	Type       string `json:"type"`
	Label      string `json:"label"`
	NetworkID  string `json:"network_id,omitempty"`
	SwitchPort string `json:"switch_port,omitempty"`
	PoolID     string `json:"pool_id,omitempty"`
}

type DeviceFilter struct {
	Tags         []string
	DatacenterID string
	NetworkID    string
	PoolID       string
	Status       DeviceStatus
}
