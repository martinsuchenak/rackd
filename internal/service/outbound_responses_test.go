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

// A webhook's delivery records hold what its endpoint answered, which can be
// an internal service's response, so only those who can edit the webhook see
// it. Readers still see that a delivery failed.
func TestWebhookDeliveryResponsesOnlyForEditors(t *testing.T) {
	f := newPrivilegeFixture(t)
	hook := &model.Webhook{Name: "hook", URL: "http://192.168.10.53/hook", Events: []model.EventType{"device.created"}, Active: true}
	if err := f.store.CreateWebhook(f.sys(), hook); err != nil {
		t.Fatal(err)
	}
	delivery := &model.WebhookDelivery{WebhookID: hook.ID, EventType: "device.created", Payload: "{}", ResponseCode: 401,
		ResponseBody: "Invalid token or session expired.", Error: "webhook returned status 401", Status: model.DeliveryStatusFailed, AttemptNumber: 1}
	if err := f.store.CreateDelivery(f.sys(), delivery); err != nil {
		t.Fatal(err)
	}
	svc := NewWebhookService(f.store)
	viewerRole, err := f.store.GetRoleByName(f.sys(), "viewer")
	if err != nil {
		t.Fatal(err)
	}
	viewer := f.user("viewer", viewerRole.ID)

	list, err := svc.ListDeliveries(as(viewer), &model.DeliveryFilter{WebhookID: hook.ID})
	if err != nil || len(list) != 1 {
		t.Fatalf("viewer list: %v %v", list, err)
	}
	one, err := svc.GetDelivery(as(viewer), delivery.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []model.WebhookDelivery{list[0], *one} {
		if d.ResponseBody != "" || d.ResponseCode != 0 || strings.Contains(d.Error, "401") || d.Error == "" || d.Status != model.DeliveryStatusFailed {
			t.Errorf("viewer sees %+v, want a failed delivery without the response", d)
		}
	}

	one, err = svc.GetDelivery(as(f.admin), delivery.ID)
	if err != nil || one.ResponseBody == "" || one.ResponseCode != 401 {
		t.Errorf("admin sees %+v (%v), want the response", one, err)
	}
}
