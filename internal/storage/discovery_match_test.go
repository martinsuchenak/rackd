package storage

import (
	"context"
	"testing"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// Scans re-listed hosts the inventory already had as new, offered to promote
// them (creating duplicates), and called devices found by every scan stale,
// because only promotion linked a host to a device.
func TestDiscoveredHostsMatchInventoryDevices(t *testing.T) {
	s := newTestStorage(t)
	defer func() { _ = s.Close() }()
	ctx := context.Background()

	lan := &model.Network{Name: "lan", Subnet: "10.0.0.0/24"}
	lab := &model.Network{Name: "lab", Subnet: "10.0.0.0/24"} // same range, another network
	for _, n := range []*model.Network{lan, lab} {
		if err := s.CreateNetwork(ctx, n); err != nil {
			t.Fatal(err)
		}
	}
	router := &model.Device{Name: "router", Status: model.DeviceStatusActive,
		Addresses: []model.Address{{IP: "10.0.0.5", Type: "ipv4", NetworkID: lan.ID}}}
	printer := &model.Device{Name: "printer", Status: model.DeviceStatusActive,
		Addresses: []model.Address{{IP: "10.0.0.6", Type: "ipv4"}}} // no network set
	unseen := &model.Device{Name: "unseen", Status: model.DeviceStatusActive}
	for _, d := range []*model.Device{router, printer, unseen} {
		if err := s.CreateDevice(ctx, d); err != nil {
			t.Fatal(err)
		}
	}
	host := func(ip, networkID string) *model.DiscoveredDevice {
		d := &model.DiscoveredDevice{IP: ip, NetworkID: networkID, Status: "online"}
		if _, err := s.SaveDiscoveredDevice(ctx, d); err != nil {
			t.Fatal(err)
		}
		return d
	}
	onLAN := host("10.0.0.5", lan.ID)
	noNetwork := host("10.0.0.6", lan.ID)
	otherNetwork := host("10.0.0.5", lab.ID)
	unknown := host("10.0.0.7", lan.ID)

	want := map[string]string{onLAN.ID: router.ID, noNetwork.ID: printer.ID, otherNetwork.ID: "", unknown.ID: ""}
	all, err := s.ListDiscoveredDevices(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range all {
		if d.MatchedDeviceID != want[d.ID] {
			t.Errorf("%s on %s matched %q, want %q", d.IP, d.NetworkID, d.MatchedDeviceID, want[d.ID])
		}
	}
	got, err := s.GetDiscoveredDevice(ctx, onLAN.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.MatchedDeviceID != router.ID || got.MatchedDeviceName != "router" {
		t.Errorf("GetDiscoveredDevice matched %q %q, want the router", got.MatchedDeviceID, got.MatchedDeviceName)
	}

	stats, err := s.GetDashboardStats(ctx, 7, 5)
	if err != nil {
		t.Fatal(err)
	}
	if stats.DiscoveredDevices != 2 {
		t.Errorf("dashboard counts %d discovered hosts not in the inventory, want 2", stats.DiscoveredDevices)
	}
	if stats.StaleDevices != 1 || len(stats.StaleDeviceList) != 1 || stats.StaleDeviceList[0].ID != unseen.ID {
		t.Errorf("stale: %d %+v, want only the device no scan has seen", stats.StaleDevices, stats.StaleDeviceList)
	}
	stale, err := s.ListDevices(ctx, &model.DeviceFilter{StaleDays: 7})
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 1 || stale[0].ID != unseen.ID {
		t.Errorf("stale filter returned %d devices, want only the unseen one", len(stale))
	}
}
