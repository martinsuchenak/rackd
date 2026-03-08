package model

import "time"

// CircuitStatus represents the operational status of a circuit
type CircuitStatus string

const (
	CircuitStatusActive      CircuitStatus = "active"
	CircuitStatusMaintenance CircuitStatus = "maintenance"
	CircuitStatusDown        CircuitStatus = "down"
	CircuitStatusDecom       CircuitStatus = "decommissioned"
)

// ValidCircuitStatuses contains all valid circuit statuses
var ValidCircuitStatuses = []CircuitStatus{
	CircuitStatusActive,
	CircuitStatusMaintenance,
	CircuitStatusDown,
	CircuitStatusDecom,
}

// IsValid checks if the status is a valid circuit status
func (s CircuitStatus) IsValid() bool {
	for _, status := range ValidCircuitStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// String returns the string representation of the status
func (s CircuitStatus) String() string {
	return string(s)
}

// Circuit represents a network circuit (WAN link, cross-connect, etc.)
type Circuit struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	CircuitID      string        `json:"circuit_id"`        // Provider's circuit identifier
	Provider       string        `json:"provider"`          // ISP or provider name
	Type           string        `json:"type"`              // e.g., "fiber", "copper", "microwave", "dark_fiber"
	Status         CircuitStatus `json:"status"`
	CapacityMbps   int           `json:"capacity_mbps"`     // Bandwidth capacity
	DatacenterAID  string        `json:"datacenter_a_id"`   // Endpoint A datacenter
	DatacenterBID  string        `json:"datacenter_b_id"`   // Endpoint B datacenter (optional for WAN)
	DeviceAID      string        `json:"device_a_id"`       // Device at endpoint A (optional)
	DeviceBID      string        `json:"device_b_id"`       // Device at endpoint B (optional)
	PortA          string        `json:"port_a"`            // Port/interface at endpoint A
	PortB          string        `json:"port_b"`            // Port/interface at endpoint B
	IPAddressA     string        `json:"ip_address_a"`      // IP address at endpoint A (optional)
	IPAddressB     string        `json:"ip_address_b"`      // IP address at endpoint B (optional)
	VLANID         int           `json:"vlan_id"`           // VLAN ID (optional)
	Description    string        `json:"description"`
	InstallDate    *time.Time    `json:"install_date,omitempty"`
	TerminateDate  *time.Time    `json:"terminate_date,omitempty"`
	MonthlyCost    float64       `json:"monthly_cost"`      // Optional cost tracking
	ContractNumber string        `json:"contract_number"`   // Optional contract reference
	ContactName    string        `json:"contact_name"`      // Provider contact
	ContactPhone   string        `json:"contact_phone"`     // Provider phone
	ContactEmail   string        `json:"contact_email"`     // Provider email
	Tags           []string      `json:"tags"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// CircuitFilter for filtering circuits
type CircuitFilter struct {
	Pagination
	Provider     string
	Status       CircuitStatus
	DatacenterID string
	Type         string
	Tags         []string
}

// CreateCircuitRequest represents the input for creating a circuit
type CreateCircuitRequest struct {
	Name           string        `json:"name"`
	CircuitID      string        `json:"circuit_id"`
	Provider       string        `json:"provider"`
	Type           string        `json:"type"`
	Status         CircuitStatus `json:"status"`
	CapacityMbps   int           `json:"capacity_mbps"`
	DatacenterAID  string        `json:"datacenter_a_id"`
	DatacenterBID  string        `json:"datacenter_b_id"`
	DeviceAID      string        `json:"device_a_id"`
	DeviceBID      string        `json:"device_b_id"`
	PortA          string        `json:"port_a"`
	PortB          string        `json:"port_b"`
	IPAddressA     string        `json:"ip_address_a"`
	IPAddressB     string        `json:"ip_address_b"`
	VLANID         int           `json:"vlan_id"`
	Description    string        `json:"description"`
	InstallDate    *time.Time    `json:"install_date"`
	MonthlyCost    float64       `json:"monthly_cost"`
	ContractNumber string        `json:"contract_number"`
	ContactName    string        `json:"contact_name"`
	ContactPhone   string        `json:"contact_phone"`
	ContactEmail   string        `json:"contact_email"`
	Tags           []string      `json:"tags"`
}

// UpdateCircuitRequest represents the input for updating a circuit
type UpdateCircuitRequest struct {
	Name           *string        `json:"name,omitempty"`
	CircuitID      *string        `json:"circuit_id,omitempty"`
	Provider       *string        `json:"provider,omitempty"`
	Type           *string        `json:"type,omitempty"`
	Status         *CircuitStatus `json:"status,omitempty"`
	CapacityMbps   *int           `json:"capacity_mbps,omitempty"`
	DatacenterAID  *string        `json:"datacenter_a_id,omitempty"`
	DatacenterBID  *string        `json:"datacenter_b_id,omitempty"`
	DeviceAID      *string        `json:"device_a_id,omitempty"`
	DeviceBID      *string        `json:"device_b_id,omitempty"`
	PortA          *string        `json:"port_a,omitempty"`
	PortB          *string        `json:"port_b,omitempty"`
	IPAddressA     *string        `json:"ip_address_a,omitempty"`
	IPAddressB     *string        `json:"ip_address_b,omitempty"`
	VLANID         *int           `json:"vlan_id,omitempty"`
	Description    *string        `json:"description,omitempty"`
	InstallDate    *time.Time     `json:"install_date,omitempty"`
	TerminateDate  *time.Time     `json:"terminate_date,omitempty"`
	MonthlyCost    *float64       `json:"monthly_cost,omitempty"`
	ContractNumber *string        `json:"contract_number,omitempty"`
	ContactName    *string        `json:"contact_name,omitempty"`
	ContactPhone   *string        `json:"contact_phone,omitempty"`
	ContactEmail   *string        `json:"contact_email,omitempty"`
	Tags           *[]string      `json:"tags,omitempty"`
}
