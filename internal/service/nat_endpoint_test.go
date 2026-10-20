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
