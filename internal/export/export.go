package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

// Format represents the export format
type Format string

const (
	FormatJSON Format = "json"
	FormatCSV  Format = "csv"
)

// ExportDevices exports devices to the specified format
func ExportDevices(devices []model.Device, format Format, w io.Writer) error {
	switch format {
	case FormatJSON:
		return exportDevicesJSON(devices, w)
	case FormatCSV:
		return exportDevicesCSV(devices, w)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func exportDevicesJSON(devices []model.Device, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(devices)
}

func exportDevicesCSV(devices []model.Device, w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	header := []string{"id", "name", "hostname", "description", "make_model", "os", "datacenter_id", "username", "location", "addresses", "tags", "domains", "created_at", "updated_at"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write rows
	for _, device := range devices {
		row := []string{
			device.ID,
			device.Name,
			device.Hostname,
			device.Description,
			device.MakeModel,
			device.OS,
			device.DatacenterID,
			device.Username,
			device.Location,
			joinAddresses(device.Addresses),
			strings.Join(device.Tags, ";"),
			strings.Join(device.Domains, ";"),
			device.CreatedAt.Format(time.RFC3339),
			device.UpdatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func joinAddresses(addresses []model.Address) string {
	var parts []string
	for _, addr := range addresses {
		parts = append(parts, fmt.Sprintf("%s:%s", addr.NetworkID, addr.IP))
	}
	return strings.Join(parts, ";")
}

// ExportNetworks exports networks to the specified format
func ExportNetworks(networks []model.Network, format Format, w io.Writer) error {
	switch format {
	case FormatJSON:
		return exportNetworksJSON(networks, w)
	case FormatCSV:
		return exportNetworksCSV(networks, w)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func exportNetworksJSON(networks []model.Network, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(networks)
}

func exportNetworksCSV(networks []model.Network, w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	header := []string{"id", "name", "subnet", "vlan_id", "description", "datacenter_id", "created_at", "updated_at"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write rows
	for _, network := range networks {
		row := []string{
			network.ID,
			network.Name,
			network.Subnet,
			fmt.Sprintf("%d", network.VLANID),
			network.Description,
			network.DatacenterID,
			network.CreatedAt.Format(time.RFC3339),
			network.UpdatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// ExportDatacenters exports datacenters to the specified format
func ExportDatacenters(datacenters []model.Datacenter, format Format, w io.Writer) error {
	switch format {
	case FormatJSON:
		return exportDatacentersJSON(datacenters, w)
	case FormatCSV:
		return exportDatacentersCSV(datacenters, w)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func exportDatacentersJSON(datacenters []model.Datacenter, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(datacenters)
}

func exportDatacentersCSV(datacenters []model.Datacenter, w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	header := []string{"id", "name", "location", "description", "created_at", "updated_at"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write rows
	for _, dc := range datacenters {
		row := []string{
			dc.ID,
			dc.Name,
			dc.Location,
			dc.Description,
			dc.CreatedAt.Format(time.RFC3339),
			dc.UpdatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}
