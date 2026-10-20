package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/martinsuchenak/rackd/internal/storage/storagetest"
	"github.com/martinsuchenak/rackd/pkg/model"
)

// Two enabled mappings can't forward the same external endpoint: they'd say
// different things about where its traffic goes (found by a Strix run).
func TestNATMappingsCantShareAnExternalEndpoint(t *testing.T) {
	svc := NewNATService(storagetest.New(t))
	ctx := SystemContext(context.Background(), "test")
	create := func(name, extIP string, port int, proto model.NATProtocol, enabled bool) (*model.NATMapping, error) {
		return svc.Create(ctx, &model.CreateNATRequest{Name: name, ExternalIP: extIP, ExternalPort: port,
			InternalIP: "10.0.0.5", InternalPort: 80, Protocol: proto, Enabled: enabled})
	}

	web, err := create("web", "203.0.113.20", 443, model.NATProtocolTCP, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name  string
		ip    string
		port  int
		proto model.NATProtocol
	}{
		{"same endpoint", "203.0.113.20", 443, model.NATProtocolTCP},
		{"any protocol", "203.0.113.20", 443, model.NATProtocolAny},
		{"every port", "203.0.113.20", 0, model.NATProtocolTCP},
	} {
		_, err := create(c.name, c.ip, c.port, c.proto, true)
		if !errors.Is(err, ErrAlreadyExists) || !strings.Contains(err.Error(), `"web"`) {
			t.Errorf("%s: %v, want already exists naming web", c.name, err)
		}
	}
	for _, c := range []struct {
		name    string
		ip      string
		port    int
		proto   model.NATProtocol
		enabled bool
	}{
		{"udp on the same port", "203.0.113.20", 443, model.NATProtocolUDP, true},
		{"another port", "203.0.113.20", 8443, model.NATProtocolTCP, true},
		{"another IP", "203.0.113.21", 443, model.NATProtocolTCP, true},
		{"disabled duplicate", "203.0.113.20", 443, model.NATProtocolTCP, false},
	} {
		if _, err := create(c.name, c.ip, c.port, c.proto, c.enabled); err != nil {
			t.Errorf("%s: %v, want it created", c.name, err)
		}
	}

	// IPv6 written two ways is one address.
	if _, err := create("v6", "2001:DB8::1", 443, model.NATProtocolTCP, true); err != nil {
		t.Fatal(err)
	}
	if _, err := create("v6 again", "2001:db8:0::1", 443, model.NATProtocolTCP, true); !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("same IPv6 written differently: %v, want already exists", err)
	}

	// Moving a mapping onto a used endpoint, or enabling a duplicate, is
	// refused; editing anything else isn't.
	other, err := svc.List(ctx, &model.NATFilter{})
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]model.NATMapping{}
	for _, m := range other {
		byName[m.Name] = m
	}
	port := 443
	if _, err := svc.Update(ctx, byName["another port"].ID, &model.UpdateNATRequest{ExternalPort: &port}); !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("move onto web's endpoint: %v, want already exists", err)
	}
	on := true
	if _, err := svc.Update(ctx, byName["disabled duplicate"].ID, &model.UpdateNATRequest{Enabled: &on}); !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("enable a duplicate: %v, want already exists", err)
	}
	desc := "front door"
	if _, err := svc.Update(ctx, web.ID, &model.UpdateNATRequest{Description: &desc}); err != nil {
		t.Errorf("edit web's description: %v", err)
	}
}

// Mappings that already shared an endpoint before the check existed can still
// be edited, as long as the edit doesn't move or enable them.
func TestExistingDuplicateNATMappingsStayEditable(t *testing.T) {
	store := storagetest.New(t)
	svc := NewNATService(store)
	ctx := SystemContext(context.Background(), "test")
	first, err := svc.Create(ctx, &model.CreateNATRequest{Name: "a", ExternalIP: "203.0.113.30", ExternalPort: 443,
		InternalIP: "10.0.0.5", InternalPort: 80, Protocol: model.NATProtocolTCP, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	// The second is written directly, as an older version would have.
	if _, err := store.DB().ExecContext(ctx, `
		INSERT INTO nat_mappings (id, name, external_ip, external_port, internal_ip, internal_port, protocol, enabled, tags, created_at, updated_at)
		SELECT 'legacy-dup', 'b', external_ip, external_port, '10.0.0.6', 22, protocol, 1, '[]', created_at, updated_at
		FROM nat_mappings WHERE id = ?`, first.ID); err != nil {
		t.Fatal(err)
	}
	name := "b renamed"
	if _, err := svc.Update(ctx, "legacy-dup", &model.UpdateNATRequest{Name: &name}); err != nil {
		t.Errorf("rename a legacy duplicate: %v", err)
	}
	port := 444
	if _, err := svc.Update(ctx, "legacy-dup", &model.UpdateNATRequest{ExternalPort: &port}); err != nil {
		t.Errorf("move the duplicate off the shared endpoint: %v", err)
	}
	port = 443
	if _, err := svc.Update(ctx, "legacy-dup", &model.UpdateNATRequest{ExternalPort: &port}); !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("move it back: %v, want already exists", err)
	}
}

// A device saved with an address as the web UI builds it (no pool, no
// network) claims the reservation for that address.
func TestDeviceWithAnUnboundAddressClaimsTheReservation(t *testing.T) {
	store := storagetest.New(t)
	ctx := SystemContext(context.Background(), "test")
	network := &model.Network{Name: "lan", Subnet: "10.9.0.0/24"}
	if err := store.CreateNetwork(ctx, network); err != nil {
		t.Fatal(err)
	}
	pool := &model.NetworkPool{NetworkID: network.ID, Name: "servers", StartIP: "10.9.0.10", EndIP: "10.9.0.20"}
	if err := store.CreateNetworkPool(ctx, pool); err != nil {
		t.Fatal(err)
	}
	reservation, err := NewReservationService(store).Create(ctx, &model.CreateReservationRequest{PoolID: pool.ID, IPAddress: "10.9.0.12"})
	if err != nil {
		t.Fatal(err)
	}

	device := &model.Device{Name: "srv", Addresses: []model.Address{{IP: "10.9.0.12", Type: "ipv4"}}}
	if err := NewDeviceService(store).Create(ctx, device); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetReservation(ctx, reservation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.ReservationStatusClaimed {
		t.Errorf("reservation is %s, want claimed", got.Status)
	}
}
