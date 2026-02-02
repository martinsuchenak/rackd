package model

import (
	"fmt"
	"time"
)

var ValidScanTypes = map[string]bool{
	"quick":  true,
	"full":   true,
	"deep":   true,
	"custom": true,
}

const (
	MaxWorkers = 100
	MinPort    = 1
	MaxPort    = 65535
)

type ScanProfile struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ScanType    string    `json:"scan_type"`
	Ports       []int     `json:"ports,omitempty"`
	EnableSNMP  bool      `json:"enable_snmp"`
	EnableSSH   bool      `json:"enable_ssh"`
	TimeoutSec  int       `json:"timeout_sec"`
	MaxWorkers  int       `json:"max_workers"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (s *ScanProfile) Validate() error {
	if !ValidScanTypes[s.ScanType] {
		return fmt.Errorf("invalid scan type: %s (must be one of: quick, full, deep, custom)", s.ScanType)
	}

	if s.MaxWorkers <= 0 || s.MaxWorkers > MaxWorkers {
		return fmt.Errorf("max_workers must be between 1 and %d, got %d", MaxWorkers, s.MaxWorkers)
	}

	if s.TimeoutSec <= 0 {
		return fmt.Errorf("timeout_sec must be positive, got %d", s.TimeoutSec)
	}

	for _, port := range s.Ports {
		if port < MinPort || port > MaxPort {
			return fmt.Errorf("port %d is out of valid range (must be %d-%d)", port, MinPort, MaxPort)
		}
	}

	return nil
}
