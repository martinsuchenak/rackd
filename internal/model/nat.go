package model

import "time"

// NATProtocol represents the protocol type for a NAT mapping
type NATProtocol string

const (
	NATProtocolTCP NATProtocol = "tcp"
	NATProtocolUDP NATProtocol = "udp"
	NATProtocolAny NATProtocol = "any"
)

// ValidNATProtocols contains all valid NAT protocols
var ValidNATProtocols = []NATProtocol{
	NATProtocolTCP,
	NATProtocolUDP,
	NATProtocolAny,
}

// IsValid checks if the protocol is valid
func (p NATProtocol) IsValid() bool {
	for _, proto := range ValidNATProtocols {
		if p == proto {
			return true
		}
	}
	return false
}

// String returns the string representation of the protocol
func (p NATProtocol) String() string {
	return string(p)
}

// NATMapping represents a NAT (Network Address Translation) mapping
type NATMapping struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	ExternalIP   string      `json:"external_ip"`
	ExternalPort int         `json:"external_port"`
	InternalIP   string      `json:"internal_ip"`
	InternalPort int         `json:"internal_port"`
	Protocol     NATProtocol `json:"protocol"`
	DeviceID     string      `json:"device_id,omitempty"`
	Description  string      `json:"description"`
	Enabled      bool        `json:"enabled"`
	DatacenterID string      `json:"datacenter_id,omitempty"`
	NetworkID    string      `json:"network_id,omitempty"`
	Tags         []string    `json:"tags"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// NATFilter for filtering NAT mappings
type NATFilter struct {
	ExternalIP   string
	InternalIP   string
	Protocol     NATProtocol
	DeviceID     string
	DatacenterID string
	NetworkID    string
	Enabled      *bool
	Tags         []string
}

// CreateNATRequest represents the input for creating a NAT mapping
type CreateNATRequest struct {
	Name         string      `json:"name"`
	ExternalIP   string      `json:"external_ip"`
	ExternalPort int         `json:"external_port"`
	InternalIP   string      `json:"internal_ip"`
	InternalPort int         `json:"internal_port"`
	Protocol     NATProtocol `json:"protocol"`
	DeviceID     string      `json:"device_id"`
	Description  string      `json:"description"`
	Enabled      bool        `json:"enabled"`
	DatacenterID string      `json:"datacenter_id"`
	NetworkID    string      `json:"network_id"`
	Tags         []string    `json:"tags"`
}

// UpdateNATRequest represents the input for updating a NAT mapping
type UpdateNATRequest struct {
	Name         *string      `json:"name,omitempty"`
	ExternalIP   *string      `json:"external_ip,omitempty"`
	ExternalPort *int         `json:"external_port,omitempty"`
	InternalIP   *string      `json:"internal_ip,omitempty"`
	InternalPort *int         `json:"internal_port,omitempty"`
	Protocol     *NATProtocol `json:"protocol,omitempty"`
	DeviceID     *string      `json:"device_id,omitempty"`
	Description  *string      `json:"description,omitempty"`
	Enabled      *bool        `json:"enabled,omitempty"`
	DatacenterID *string      `json:"datacenter_id,omitempty"`
	NetworkID    *string      `json:"network_id,omitempty"`
	Tags         *[]string    `json:"tags,omitempty"`
}
