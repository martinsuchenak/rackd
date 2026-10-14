package mcp

import (
	"strings"
	"testing"

	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/service"
)

var dnsTools = []string{"dns_provider_list", "dns_zone_list", "dns_record_list"}

func registered(s *Server, names []string) []string {
	var out []string
	for _, name := range names {
		if _, ok := s.Inner().ToolSource(name); ok {
			out = append(out, name)
		}
	}
	return out
}

func TestDNSToolsNeedTheDNSService(t *testing.T) {
	without, store := newTestServer(t)
	defer func() { _ = store.Close() }()
	if names := registered(without, dnsTools); len(names) != 0 {
		t.Errorf("DNS tools registered without the DNS service: %v", names)
	}
	if strings.Contains(without.base, "DNS") {
		t.Errorf("instructions mention DNS without the DNS service: %q", without.base)
	}

	svc := service.NewServices(store, nil, nil)
	encryptor, err := credentials.NewEncryptor(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	svc.SetDNSService(store, encryptor)
	with := NewServer(svc, store, false)
	if names := registered(with, dnsTools); len(names) != len(dnsTools) {
		t.Errorf("DNS tools with the DNS service: %v", names)
	}
	if !strings.Contains(with.base, "DNS management") {
		t.Errorf("instructions don't mention DNS: %q", with.base)
	}
}
