package api

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/martinsuchenak/rackd/internal/model"
)

// ValidationError represents a validation failure
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors collects multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// ValidateDevice validates a device for creation or update
func ValidateDevice(device *model.Device) ValidationErrors {
	var errs ValidationErrors

	// Name is required and must be reasonable length
	if strings.TrimSpace(device.Name) == "" {
		errs = append(errs, ValidationError{Field: "name", Message: "name is required"})
	} else if len(device.Name) > 255 {
		errs = append(errs, ValidationError{Field: "name", Message: "name must be 255 characters or less"})
	}

	// Hostname validation (if provided)
	if device.Hostname != "" {
		if len(device.Hostname) > 253 {
			errs = append(errs, ValidationError{Field: "hostname", Message: "hostname must be 253 characters or less"})
		} else if !isValidHostname(device.Hostname) {
			errs = append(errs, ValidationError{Field: "hostname", Message: "hostname contains invalid characters"})
		}
	}

	// Description length check
	if len(device.Description) > 4096 {
		errs = append(errs, ValidationError{Field: "description", Message: "description must be 4096 characters or less"})
	}

	// Validate addresses
	for i, addr := range device.Addresses {
		addrErrs := validateAddress(addr, i)
		errs = append(errs, addrErrs...)
	}

	// Validate tags
	for i, tag := range device.Tags {
		if strings.TrimSpace(tag) == "" {
			errs = append(errs, ValidationError{Field: fmt.Sprintf("tags[%d]", i), Message: "tag cannot be empty"})
		} else if len(tag) > 128 {
			errs = append(errs, ValidationError{Field: fmt.Sprintf("tags[%d]", i), Message: "tag must be 128 characters or less"})
		}
	}

	// Validate domains
	for i, domain := range device.Domains {
		if strings.TrimSpace(domain) == "" {
			errs = append(errs, ValidationError{Field: fmt.Sprintf("domains[%d]", i), Message: "domain cannot be empty"})
		} else if len(domain) > 253 {
			errs = append(errs, ValidationError{Field: fmt.Sprintf("domains[%d]", i), Message: "domain must be 253 characters or less"})
		} else if !isValidDomain(domain) {
			errs = append(errs, ValidationError{Field: fmt.Sprintf("domains[%d]", i), Message: "invalid domain format"})
		}
	}

	return errs
}

func validateAddress(addr model.Address, index int) ValidationErrors {
	var errs ValidationErrors
	fieldPrefix := fmt.Sprintf("addresses[%d]", index)

	// IP is required for addresses
	if strings.TrimSpace(addr.IP) == "" {
		errs = append(errs, ValidationError{Field: fieldPrefix + ".ip", Message: "IP address is required"})
	} else if !isValidIP(addr.IP) {
		errs = append(errs, ValidationError{Field: fieldPrefix + ".ip", Message: "invalid IP address format"})
	}

	// Port validation (if provided)
	if addr.Port != nil {
		if *addr.Port < 1 || *addr.Port > 65535 {
			errs = append(errs, ValidationError{Field: fieldPrefix + ".port", Message: "port must be between 1 and 65535"})
		}
	}

	// Type length validation (type is a freeform label, not strictly validated)
	if len(addr.Type) > 64 {
		errs = append(errs, ValidationError{Field: fieldPrefix + ".type", Message: "type must be 64 characters or less"})
	}

	// Label length check
	if len(addr.Label) > 128 {
		errs = append(errs, ValidationError{Field: fieldPrefix + ".label", Message: "label must be 128 characters or less"})
	}

	return errs
}

// ValidateNetwork validates a network for creation or update
func ValidateNetwork(network *model.Network) ValidationErrors {
	var errs ValidationErrors

	if strings.TrimSpace(network.Name) == "" {
		errs = append(errs, ValidationError{Field: "name", Message: "name is required"})
	} else if len(network.Name) > 255 {
		errs = append(errs, ValidationError{Field: "name", Message: "name must be 255 characters or less"})
	}

	if strings.TrimSpace(network.Subnet) == "" {
		errs = append(errs, ValidationError{Field: "subnet", Message: "subnet is required"})
	} else if !isValidCIDR(network.Subnet) {
		errs = append(errs, ValidationError{Field: "subnet", Message: "invalid CIDR notation"})
	}

	if network.VLANID < 0 || network.VLANID > 4094 {
		errs = append(errs, ValidationError{Field: "vlan_id", Message: "VLAN ID must be between 0 and 4094"})
	}

	if len(network.Description) > 4096 {
		errs = append(errs, ValidationError{Field: "description", Message: "description must be 4096 characters or less"})
	}

	return errs
}

// ValidateNetworkPool validates a network pool for creation or update
func ValidateNetworkPool(pool *model.NetworkPool) ValidationErrors {
	var errs ValidationErrors

	if strings.TrimSpace(pool.Name) == "" {
		errs = append(errs, ValidationError{Field: "name", Message: "name is required"})
	} else if len(pool.Name) > 255 {
		errs = append(errs, ValidationError{Field: "name", Message: "name must be 255 characters or less"})
	}

	if strings.TrimSpace(pool.StartIP) == "" {
		errs = append(errs, ValidationError{Field: "start_ip", Message: "start IP is required"})
	} else if !isValidIP(pool.StartIP) {
		errs = append(errs, ValidationError{Field: "start_ip", Message: "invalid start IP address"})
	}

	if strings.TrimSpace(pool.EndIP) == "" {
		errs = append(errs, ValidationError{Field: "end_ip", Message: "end IP is required"})
	} else if !isValidIP(pool.EndIP) {
		errs = append(errs, ValidationError{Field: "end_ip", Message: "invalid end IP address"})
	}

	// Validate IP range order
	if isValidIP(pool.StartIP) && isValidIP(pool.EndIP) {
		startIP := net.ParseIP(pool.StartIP)
		endIP := net.ParseIP(pool.EndIP)
		if startIP != nil && endIP != nil {
			// Normalize to same format for comparison
			start4 := startIP.To4()
			end4 := endIP.To4()
			if start4 != nil && end4 != nil {
				if ipToUint32(start4) > ipToUint32(end4) {
					errs = append(errs, ValidationError{Field: "end_ip", Message: "end IP must be greater than or equal to start IP"})
				}
			}
		}
	}

	if len(pool.Description) > 4096 {
		errs = append(errs, ValidationError{Field: "description", Message: "description must be 4096 characters or less"})
	}

	return errs
}

// ValidateDatacenter validates a datacenter for creation or update
func ValidateDatacenter(dc *model.Datacenter) ValidationErrors {
	var errs ValidationErrors

	if strings.TrimSpace(dc.Name) == "" {
		errs = append(errs, ValidationError{Field: "name", Message: "name is required"})
	} else if len(dc.Name) > 255 {
		errs = append(errs, ValidationError{Field: "name", Message: "name must be 255 characters or less"})
	}

	if len(dc.Location) > 255 {
		errs = append(errs, ValidationError{Field: "location", Message: "location must be 255 characters or less"})
	}

	if len(dc.Description) > 4096 {
		errs = append(errs, ValidationError{Field: "description", Message: "description must be 4096 characters or less"})
	}

	return errs
}

// Helper functions

func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

func isValidCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
}

var hostnameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)

func isValidHostname(hostname string) bool {
	if len(hostname) > 253 {
		return false
	}
	return hostnameRegex.MatchString(hostname)
}

var domainRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)

func isValidDomain(domain string) bool {
	if len(domain) > 253 {
		return false
	}
	return domainRegex.MatchString(domain)
}

func ipToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}
