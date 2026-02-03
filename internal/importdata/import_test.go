package importdata

import (
	"bytes"
	"testing"
)

func TestImportDevicesJSON(t *testing.T) {
	jsonData := `[
		{
			"id": "dev-1",
			"name": "server-1",
			"hostname": "server1.example.com",
			"tags": ["prod", "web"],
			"addresses": [{"ip": "10.0.0.1", "network_id": "net-1"}]
		}
	]`

	devices, err := ImportDevicesJSON(bytes.NewBufferString(jsonData))
	if err != nil {
		t.Fatalf("ImportDevicesJSON failed: %v", err)
	}

	if len(devices) != 1 {
		t.Fatalf("Expected 1 device, got %d", len(devices))
	}

	if devices[0].Name != "server-1" {
		t.Errorf("Expected name 'server-1', got '%s'", devices[0].Name)
	}
}

func TestImportDevicesCSV(t *testing.T) {
	csvData := `id,name,hostname,tags,addresses
dev-1,server-1,server1.example.com,prod;web,net-1:10.0.0.1`

	devices, err := ImportDevicesCSV(bytes.NewBufferString(csvData))
	if err != nil {
		t.Fatalf("ImportDevicesCSV failed: %v", err)
	}

	if len(devices) != 1 {
		t.Fatalf("Expected 1 device, got %d", len(devices))
	}

	device := devices[0]
	if device.Name != "server-1" {
		t.Errorf("Expected name 'server-1', got '%s'", device.Name)
	}

	if len(device.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(device.Tags))
	}

	if len(device.Addresses) != 1 {
		t.Errorf("Expected 1 address, got %d", len(device.Addresses))
	} else if device.Addresses[0].IP != "10.0.0.1" {
		t.Errorf("Expected IP '10.0.0.1', got '%s'", device.Addresses[0].IP)
	}
}

func TestImportNetworksJSON(t *testing.T) {
	jsonData := `[
		{
			"id": "net-1",
			"name": "Production",
			"subnet": "10.0.0.0/24",
			"vlan_id": 100
		}
	]`

	networks, err := ImportNetworksJSON(bytes.NewBufferString(jsonData))
	if err != nil {
		t.Fatalf("ImportNetworksJSON failed: %v", err)
	}

	if len(networks) != 1 {
		t.Fatalf("Expected 1 network, got %d", len(networks))
	}

	if networks[0].Name != "Production" {
		t.Errorf("Expected name 'Production', got '%s'", networks[0].Name)
	}
}

func TestImportNetworksCSV(t *testing.T) {
	csvData := `id,name,subnet,vlan_id
net-1,Production,10.0.0.0/24,100`

	networks, err := ImportNetworksCSV(bytes.NewBufferString(csvData))
	if err != nil {
		t.Fatalf("ImportNetworksCSV failed: %v", err)
	}

	if len(networks) != 1 {
		t.Fatalf("Expected 1 network, got %d", len(networks))
	}

	network := networks[0]
	if network.Name != "Production" {
		t.Errorf("Expected name 'Production', got '%s'", network.Name)
	}

	if network.VLANID != 100 {
		t.Errorf("Expected VLAN ID 100, got %d", network.VLANID)
	}
}

func TestImportDatacentersJSON(t *testing.T) {
	jsonData := `[
		{
			"id": "dc-1",
			"name": "NYC",
			"location": "New York"
		}
	]`

	datacenters, err := ImportDatacentersJSON(bytes.NewBufferString(jsonData))
	if err != nil {
		t.Fatalf("ImportDatacentersJSON failed: %v", err)
	}

	if len(datacenters) != 1 {
		t.Fatalf("Expected 1 datacenter, got %d", len(datacenters))
	}

	if datacenters[0].Name != "NYC" {
		t.Errorf("Expected name 'NYC', got '%s'", datacenters[0].Name)
	}
}

func TestImportDatacentersCSV(t *testing.T) {
	csvData := `id,name,location
dc-1,NYC,New York`

	datacenters, err := ImportDatacentersCSV(bytes.NewBufferString(csvData))
	if err != nil {
		t.Fatalf("ImportDatacentersCSV failed: %v", err)
	}

	if len(datacenters) != 1 {
		t.Fatalf("Expected 1 datacenter, got %d", len(datacenters))
	}

	datacenter := datacenters[0]
	if datacenter.Name != "NYC" {
		t.Errorf("Expected name 'NYC', got '%s'", datacenter.Name)
	}

	if datacenter.Location != "New York" {
		t.Errorf("Expected location 'New York', got '%s'", datacenter.Location)
	}
}

func TestParseAddresses(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"net-1:10.0.0.1", 1},
		{"net-1:10.0.0.1;net-2:10.0.1.1", 2},
		{"", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		addrs := parseAddresses(tt.input)
		if len(addrs) != tt.expected {
			t.Errorf("parseAddresses(%q) = %d addresses, want %d", tt.input, len(addrs), tt.expected)
		}
	}
}
