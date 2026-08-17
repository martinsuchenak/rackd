package export

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestExportDevicesJSON(t *testing.T) {
	devices := []model.Device{
		{
			ID:          "dev-1",
			Name:        "server-1",
			Hostname:    "server1.example.com",
			Description: "Test server",
			Tags:        []string{"prod", "web"},
			Addresses: []model.Address{
				{IP: "10.0.0.1", NetworkID: "net-1"},
			},
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	var buf bytes.Buffer
	if err := ExportDevices(devices, FormatJSON, &buf); err != nil {
		t.Fatalf("ExportDevices failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "server-1") {
		t.Error("Expected output to contain device name")
	}
	if !strings.Contains(output, "10.0.0.1") {
		t.Error("Expected output to contain IP address")
	}
}

func TestExportDevicesCSV(t *testing.T) {
	devices := []model.Device{
		{
			ID:       "dev-1",
			Name:     "server-1",
			Hostname: "server1.example.com",
			Tags:     []string{"prod", "web"},
			Addresses: []model.Address{
				{IP: "10.0.0.1", NetworkID: "net-1"},
			},
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	var buf bytes.Buffer
	if err := ExportDevices(devices, FormatCSV, &buf); err != nil {
		t.Fatalf("ExportDevices failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines (header + 1 device), got %d", len(lines))
	}

	// Check header
	if !strings.Contains(lines[0], "id") || !strings.Contains(lines[0], "name") {
		t.Error("Expected CSV header with id and name")
	}

	// Check data
	if !strings.Contains(lines[1], "server-1") {
		t.Error("Expected CSV data to contain device name")
	}
}

func TestExportDevicesCSVNeutralizesFormulaInjection(t *testing.T) {
	devices := []model.Device{
		{
			ID:          "dev-1",
			Name:        "=HYPERLINK(\"http://127.0.0.1:9494/leak\",\"click-me\")",
			Hostname:    "server1.example.com",
			Description: "=cmd|' /C calc'!A1",
			MakeModel:   "+sum(1+1)",
			OS:          "@SUM(1+1)",
			Username:    "-2+3",
			Location:    "=HYPERLINK(\"http://evil.example\",\"x\")",
			CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	var buf bytes.Buffer
	if err := ExportDevices(devices, FormatCSV, &buf); err != nil {
		t.Fatalf("ExportDevices failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(lines))
	}

	for _, prefix := range []string{"\"=", "\"+", "\"-", "\"@", "\"+sum", "\"-2", "\"@"} {
		if strings.Contains(output, prefix+"HYPERLINK") || strings.HasPrefix(lines[1], prefix) {
			t.Errorf("Expected formula injection to be neutralized, found raw formula in: %s", lines[1])
		}
	}
	if !strings.Contains(lines[1], "'=HYPERLINK") {
		t.Error("Expected sanitized name to start with an apostrophe before =HYPERLINK")
	}
	if !strings.Contains(lines[1], "'=cmd|") {
		t.Error("Expected sanitized description to start with an apostrophe before =cmd|")
	}
	if !strings.Contains(lines[1], "'+sum(1+1)") {
		t.Error("Expected sanitized make_model to start with an apostrophe before +sum")
	}
	if !strings.Contains(lines[1], "'@SUM(1+1)") {
		t.Error("Expected sanitized os to start with an apostrophe before @SUM")
	}
	if !strings.Contains(lines[1], "'-2+3") {
		t.Error("Expected sanitized username to start with an apostrophe before -2+3")
	}
	// The CSV writer quotes cells containing commas; ensure no cell value
	// itself begins with a formula character after the opening quote.
	for _, cell := range strings.Split(lines[1], "\",\"") {
		trimmed := strings.Trim(cell, "\"")
		if trimmed == "" {
			continue
		}
		switch trimmed[0] {
		case '=', '+', '-', '@', '\t', '\r':
			t.Errorf("Cell %q starts with a formula character", cell)
		}
	}
}

func TestExportNetworksJSON(t *testing.T) {
	networks := []model.Network{
		{
			ID:          "net-1",
			Name:        "Production",
			Subnet:      "10.0.0.0/24",
			VLANID:      100,
			Description: "Production network",
			CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	var buf bytes.Buffer
	if err := ExportNetworks(networks, FormatJSON, &buf); err != nil {
		t.Fatalf("ExportNetworks failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Production") {
		t.Error("Expected output to contain network name")
	}
	if !strings.Contains(output, "10.0.0.0/24") {
		t.Error("Expected output to contain subnet")
	}
}

func TestExportNetworksCSV(t *testing.T) {
	networks := []model.Network{
		{
			ID:        "net-1",
			Name:      "Production",
			Subnet:    "10.0.0.0/24",
			VLANID:    100,
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	var buf bytes.Buffer
	if err := ExportNetworks(networks, FormatCSV, &buf); err != nil {
		t.Fatalf("ExportNetworks failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}

	if !strings.Contains(lines[1], "Production") {
		t.Error("Expected CSV data to contain network name")
	}
}

func TestExportDatacentersJSON(t *testing.T) {
	datacenters := []model.Datacenter{
		{
			ID:          "dc-1",
			Name:        "NYC",
			Location:    "New York",
			Description: "NYC datacenter",
			CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	var buf bytes.Buffer
	if err := ExportDatacenters(datacenters, FormatJSON, &buf); err != nil {
		t.Fatalf("ExportDatacenters failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "NYC") {
		t.Error("Expected output to contain datacenter name")
	}
}

func TestExportDatacentersCSV(t *testing.T) {
	datacenters := []model.Datacenter{
		{
			ID:        "dc-1",
			Name:      "NYC",
			Location:  "New York",
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	var buf bytes.Buffer
	if err := ExportDatacenters(datacenters, FormatCSV, &buf); err != nil {
		t.Fatalf("ExportDatacenters failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}

	if !strings.Contains(lines[1], "NYC") {
		t.Error("Expected CSV data to contain datacenter name")
	}
}
