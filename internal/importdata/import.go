package importdata

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

// ConflictMode determines how to handle conflicts during import
type ConflictMode string

const (
	ConflictSkip   ConflictMode = "skip"   // Skip conflicting records
	ConflictUpdate ConflictMode = "update" // Update existing records
	ConflictFail   ConflictMode = "fail"   // Fail on conflict
)

// ImportResult contains the results of an import operation
type ImportResult struct {
	Total    int
	Created  int
	Updated  int
	Skipped  int
	Failed   int
	Errors   []string
}

// ImportDevicesJSON imports devices from JSON
func ImportDevicesJSON(r io.Reader) ([]model.Device, error) {
	var devices []model.Device
	if err := json.NewDecoder(r).Decode(&devices); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}
	return devices, nil
}

// ImportDevicesCSV imports devices from CSV
func ImportDevicesCSV(r io.Reader) ([]model.Device, error) {
	reader := csv.NewReader(r)
	
	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	// Map header to indices
	headerMap := make(map[string]int)
	for i, col := range header {
		headerMap[col] = i
	}

	var devices []model.Device
	lineNum := 1

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read line %d: %w", lineNum, err)
		}
		lineNum++

		device := model.Device{
			ID:           getField(record, headerMap, "id"),
			Name:         getField(record, headerMap, "name"),
			Hostname:     getField(record, headerMap, "hostname"),
			Description:  getField(record, headerMap, "description"),
			MakeModel:    getField(record, headerMap, "make_model"),
			OS:           getField(record, headerMap, "os"),
			DatacenterID: getField(record, headerMap, "datacenter_id"),
			Username:     getField(record, headerMap, "username"),
			Location:     getField(record, headerMap, "location"),
		}

		// Parse addresses
		if addrStr := getField(record, headerMap, "addresses"); addrStr != "" {
			device.Addresses = parseAddresses(addrStr)
		}

		// Parse tags
		if tagsStr := getField(record, headerMap, "tags"); tagsStr != "" {
			device.Tags = strings.Split(tagsStr, ";")
		}

		// Parse domains
		if domainsStr := getField(record, headerMap, "domains"); domainsStr != "" {
			device.Domains = strings.Split(domainsStr, ";")
		}

		// Parse timestamps
		if createdStr := getField(record, headerMap, "created_at"); createdStr != "" {
			if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
				device.CreatedAt = t
			}
		}
		if updatedStr := getField(record, headerMap, "updated_at"); updatedStr != "" {
			if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
				device.UpdatedAt = t
			}
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// ImportNetworksJSON imports networks from JSON
func ImportNetworksJSON(r io.Reader) ([]model.Network, error) {
	var networks []model.Network
	if err := json.NewDecoder(r).Decode(&networks); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}
	return networks, nil
}

// ImportNetworksCSV imports networks from CSV
func ImportNetworksCSV(r io.Reader) ([]model.Network, error) {
	reader := csv.NewReader(r)
	
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	headerMap := make(map[string]int)
	for i, col := range header {
		headerMap[col] = i
	}

	var networks []model.Network
	lineNum := 1

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read line %d: %w", lineNum, err)
		}
		lineNum++

		network := model.Network{
			ID:           getField(record, headerMap, "id"),
			Name:         getField(record, headerMap, "name"),
			Subnet:       getField(record, headerMap, "subnet"),
			Description:  getField(record, headerMap, "description"),
			DatacenterID: getField(record, headerMap, "datacenter_id"),
		}

		// Parse VLAN ID
		if vlanStr := getField(record, headerMap, "vlan_id"); vlanStr != "" {
			if vlan, err := strconv.Atoi(vlanStr); err == nil {
				network.VLANID = vlan
			}
		}

		// Parse timestamps
		if createdStr := getField(record, headerMap, "created_at"); createdStr != "" {
			if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
				network.CreatedAt = t
			}
		}
		if updatedStr := getField(record, headerMap, "updated_at"); updatedStr != "" {
			if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
				network.UpdatedAt = t
			}
		}

		networks = append(networks, network)
	}

	return networks, nil
}

// ImportDatacentersJSON imports datacenters from JSON
func ImportDatacentersJSON(r io.Reader) ([]model.Datacenter, error) {
	var datacenters []model.Datacenter
	if err := json.NewDecoder(r).Decode(&datacenters); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}
	return datacenters, nil
}

// ImportDatacentersCSV imports datacenters from CSV
func ImportDatacentersCSV(r io.Reader) ([]model.Datacenter, error) {
	reader := csv.NewReader(r)
	
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	headerMap := make(map[string]int)
	for i, col := range header {
		headerMap[col] = i
	}

	var datacenters []model.Datacenter
	lineNum := 1

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read line %d: %w", lineNum, err)
		}
		lineNum++

		datacenter := model.Datacenter{
			ID:          getField(record, headerMap, "id"),
			Name:        getField(record, headerMap, "name"),
			Location:    getField(record, headerMap, "location"),
			Description: getField(record, headerMap, "description"),
		}

		// Parse timestamps
		if createdStr := getField(record, headerMap, "created_at"); createdStr != "" {
			if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
				datacenter.CreatedAt = t
			}
		}
		if updatedStr := getField(record, headerMap, "updated_at"); updatedStr != "" {
			if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
				datacenter.UpdatedAt = t
			}
		}

		datacenters = append(datacenters, datacenter)
	}

	return datacenters, nil
}

// Helper functions

func getField(record []string, headerMap map[string]int, field string) string {
	if idx, ok := headerMap[field]; ok && idx < len(record) {
		return record[idx]
	}
	return ""
}

func parseAddresses(addrStr string) []model.Address {
	var addresses []model.Address
	parts := strings.Split(addrStr, ";")
	for _, part := range parts {
		if part == "" {
			continue
		}
		// Format: networkID:IP
		if idx := strings.Index(part, ":"); idx > 0 {
			addresses = append(addresses, model.Address{
				NetworkID: part[:idx],
				IP:        part[idx+1:],
			})
		}
	}
	return addresses
}
