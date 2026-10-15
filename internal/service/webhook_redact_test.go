package service

import (
	"testing"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// A webhook URL can be the credential itself, so only those who can change
// webhooks see it in full.
func TestWebhookURLsAreHiddenFromReaders(t *testing.T) {
	f := newPrivilegeFixture(t)
	hook := &model.Webhook{Name: "slack", URL: "https://hooks.slack.com/services/T000/B000/secret", Active: true, Events: []model.EventType{model.EventTypeDeviceCreated}}
	if err := f.store.CreateWebhook(f.sys(), hook); err != nil {
		t.Fatal(err)
	}
	svc := NewWebhookService(f.store)
	reader := f.user("reader", f.role("hook-reader", "webhooks:list", "webhooks:read"))
	editor := f.user("editor", f.role("hook-editor", "webhooks:list", "webhooks:read", "webhooks:update"))

	got, err := svc.Get(as(reader), hook.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != "https://hooks.slack.com/…" {
		t.Errorf("reader sees %q", got.URL)
	}
	list, err := svc.List(as(reader), nil)
	if err != nil || len(list) != 1 || list[0].URL != "https://hooks.slack.com/…" {
		t.Errorf("reader's list: %+v, %v", list, err)
	}
	if got, err := svc.Get(as(editor), hook.ID); err != nil || got.URL != hook.URL {
		t.Errorf("editor sees %q, %v", got.URL, err)
	}
}

func TestRedactWebhookURL(t *testing.T) {
	for in, want := range map[string]string{
		"https://example.com":                "https://example.com/",
		"https://example.com/":               "https://example.com/",
		"https://example.com/hook?token=abc": "https://example.com/…",
		"https://user:pass@example.com/":     "https://example.com/…",
		"not a url":                          "(hidden)",
	} {
		if got := RedactWebhookURL(in); got != want {
			t.Errorf("RedactWebhookURL(%q) = %q, want %q", in, got, want)
		}
	}
}
