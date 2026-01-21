package model

import "time"

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

type Address struct {
	IP         string `json:"ip"`
	Port       int    `json:"port"`
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
}
