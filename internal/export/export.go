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

// sanitizeCell neutralizes CSV formula injection: spreadsheet applications
// (Excel, LibreOffice, Google Sheets) interpret a cell as a formula when it
// starts with =, +, -, @, a tab, or a carriage return. Prefixing a single
// quote forces the value to be treated as text.
func sanitizeCell(value string) string {
	if value == "" {
		return value
	}
	switch value[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + value
	}
	return value
}

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
			sanitizeCell(device.Name),
			sanitizeCell(device.Hostname),
			sanitizeCell(device.Description),
			sanitizeCell(device.MakeModel),
			sanitizeCell(device.OS),
			device.DatacenterID,
			sanitizeCell(device.Username),
			sanitizeCell(device.Location),
			sanitizeCell(joinAddresses(device.Addresses)),
			sanitizeCell(strings.Join(device.Tags, ";")),
			sanitizeCell(strings.Join(device.Domains, ";")),
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
			sanitizeCell(network.Name),
			sanitizeCell(network.Subnet),
			fmt.Sprintf("%d", network.VLANID),
			sanitizeCell(network.Description),
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
			sanitizeCell(dc.Name),
			sanitizeCell(dc.Location),
			sanitizeCell(dc.Description),
			dc.CreatedAt.Format(time.RFC3339),
			dc.UpdatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}
