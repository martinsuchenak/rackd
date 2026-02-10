package discovery

import (
	"context"
	"testing"
	"time"
)

func TestNewLLDPScanner(t *testing.T) {
	scanner := NewLLDPScanner(5 * time.Second)
	if scanner == nil {
		t.Fatal("NewLLDPScanner returned nil")
	}
	if scanner.timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", scanner.timeout)
	}
}

func TestLLDPScanner_Discover(t *testing.T) {
	scanner := NewLLDPScanner(1 * time.Second)

	// Test discover
	result, err := scanner.Discover(context.Background())
	if err != nil {
		t.Logf("Discover returned error (may be expected): %v", err)
	}
	_ = result // Use result
}

func TestLLDPScanner_ParseLLDP(t *testing.T) {
	scanner := NewLLDPScanner(5 * time.Second)

	// Test with empty data
	result := scanner.parseLLDP([]byte{}, nil)
	if result != nil {
		t.Errorf("Expected nil for empty data, got %+v", result)
	}

	// Test with too short data - should not crash
	result = scanner.parseLLDP([]byte{1, 2, 3}, nil)
	if result != nil {
		t.Errorf("Expected nil for short data, got %+v", result)
	}
}

func TestLLDPScanner_ParseChassisID(t *testing.T) {
	scanner := NewLLDPScanner(5 * time.Second)
	result := &LLDPResult{}

	// Test MAC chassis ID (type 4)
	data := []byte{4, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	if len(data) >= 2 {
		scanner.parseChassisID(data, result)

		if result.ChassisType != "MAC" {
			t.Errorf("Expected chassis type MAC, got %s", result.ChassisType)
		}
		if result.ChassisID != "00:11:22:33:44:55" {
			t.Errorf("Expected chassis ID 00:11:22:33:44:55, got %s", result.ChassisID)
		}
	}
}

func TestLLDPScanner_ParsePortID(t *testing.T) {
	scanner := NewLLDPScanner(5 * time.Second)
	result := &LLDPResult{}

	// Test MAC port ID (type 3)
	data := []byte{3, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	scanner.parsePortID(data, result)

	if result.PortID != "00:11:22:33:44:55" {
		t.Errorf("Expected port ID 00:11:22:33:44:55, got %s", result.PortID)
	}
}

func TestLLDPScanner_ParsePortDesc(t *testing.T) {
	scanner := NewLLDPScanner(5 * time.Second)
	result := &LLDPResult{}

	data := []byte("GigabitEthernet0/1")
	scanner.parsePortDesc(data, result)

	if result.PortDesc != "GigabitEthernet0/1" {
		t.Errorf("Expected port desc GigabitEthernet0/1, got %s", result.PortDesc)
	}
}

func TestLLDPScanner_ParseSystemName(t *testing.T) {
	scanner := NewLLDPScanner(5 * time.Second)
	result := &LLDPResult{}

	data := []byte("Router1")
	scanner.parseSystemName(data, result)

	if result.SystemName != "Router1" {
		t.Errorf("Expected system name Router1, got %s", result.SystemName)
	}
}

func TestLLDPScanner_ParseSystemDesc(t *testing.T) {
	scanner := NewLLDPScanner(5 * time.Second)
	result := &LLDPResult{}

	data := []byte("Cisco IOS 15.2")
	scanner.parseSystemDesc(data, result)

	if result.SystemDesc != "Cisco IOS 15.2" {
		t.Errorf("Expected system desc Cisco IOS 15.2, got %s", result.SystemDesc)
	}
}

func TestLLDPScanner_ParseMgmtIP(t *testing.T) {
	scanner := NewLLDPScanner(5 * time.Second)
	result := &LLDPResult{}

	// Test IPv4: addrLen=5, subtype=1(IPv4), then 4 IP bytes
	data := []byte{5, 1, 192, 168, 1, 1}
	scanner.parseMgmtIP(data, result)

	if result.MgmtIP != "192.168.1.1" {
		t.Errorf("Expected IP 192.168.1.1, got %s", result.MgmtIP)
	}
}

func TestLLDPResult_Struct(t *testing.T) {
	result := LLDPResult{
		ChassisID:   "00:11:22:33:44:55",
		ChassisType: "MAC",
		PortID:      "GigabitEthernet0/1",
		PortDesc:    "Uplink",
		SystemName:  "Router1",
		SystemDesc:  "Cisco IOS 15.2",
		MgmtIP:      "192.168.1.1",
	}

	if result.ChassisID != "00:11:22:33:44:55" {
		t.Errorf("Expected chassis ID 00:11:22:33:44:55, got %s", result.ChassisID)
	}
	if result.SystemName != "Router1" {
		t.Errorf("Expected system name Router1, got %s", result.SystemName)
	}
}
