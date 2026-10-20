package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/pkg/model"
)

// Testing a provider makes the server call its endpoint, and the failure can
// carry the endpoint's own response (found by a Strix run: a viewer read an
// internal service's error through it). Viewers can't test; testers who
// can't edit the provider get the failure without its cause.
func TestDNSProviderTestRevealsTheCauseOnlyToEditors(t *testing.T) {
	f := newPrivilegeFixture(t)
	enc, err := credentials.NewEncryptor(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	token, err := enc.Encrypt("provider-token")
	if err != nil {
		t.Fatal(err)
	}
	// Stored directly: saving through the service would refuse the address.
	provider := &model.DNSProviderConfig{Name: "dns", Type: model.DNSProviderTypeTechnitium, Endpoint: "http://127.0.0.1:9", Token: token}
	if err := f.store.CreateDNSProvider(f.sys(), provider); err != nil {
		t.Fatal(err)
	}
	svc := NewDNSService(f.store, enc)

	builtin := func(name string) string {
		r, err := f.store.GetRoleByName(f.sys(), name)
		if err != nil {
			t.Fatal(err)
		}
		return r.ID
	}
	viewer := f.user("viewer", builtin("viewer"))
	operator := f.user("operator", builtin("operator"))

	wantForbidden(t, "viewer tests a provider", svc.TestProvider(as(viewer), provider.ID))

	err = svc.TestProvider(as(operator), provider.ID)
	if !errors.Is(err, ErrDNSProviderTestFailed) || strings.Contains(err.Error(), "127.0.0.1") || strings.Contains(err.Error(), "SSRF") {
		t.Errorf("operator: %v, want a test failure without its cause", err)
	}

	err = svc.TestProvider(as(f.admin), provider.ID)
	if !errors.Is(err, ErrDNSProviderTestFailed) || !strings.Contains(err.Error(), "SSRF") {
		t.Errorf("admin: %v, want the cause", err)
	}
}
