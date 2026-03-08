package model

import "time"

type Network struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Subnet       string    `json:"subnet"`
	VLANID       int       `json:"vlan_id"`
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
	Pagination
	Name         string
	DatacenterID string
	VLANID       int
}

type NetworkPoolFilter struct {
	Pagination
	NetworkID string
	Tags      []string
}

type NetworkUtilization struct {
	NetworkID    string  `json:"network_id"`
	TotalIPs     int     `json:"total_ips"`
	UsedIPs      int     `json:"used_ips"`
	AvailableIPs int     `json:"available_ips"`
	Utilization  float64 `json:"utilization"`
}
