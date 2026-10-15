package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

// The stored token is write-only, so moving a provider to a new endpoint must
// not send it there.
func TestChangingADNSEndpointNeedsTheToken(t *testing.T) {
	store, err := storage.NewSQLiteStorageWithPath(filepath.Join(t.TempDir(), "rackd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	enc, err := credentials.NewEncryptor(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	svc := NewDNSService(store, enc)
	ctx := SystemContext(context.Background(), "test")

	provider, err := svc.CreateProvider(ctx, &model.CreateDNSProviderRequest{
		Name: "dns", Type: model.DNSProviderTypeTechnitium, Endpoint: "https://203.0.113.10", Token: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}

	str := func(s string) *string { return &s }
	_, err = svc.UpdateProvider(ctx, provider.ID, &model.UpdateDNSProviderRequest{Endpoint: str("https://203.0.113.99")})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("new endpoint without the token: %v, want a validation error", err)
	}
	// Resending the same endpoint, as an edit form does, is fine.
	if _, err := svc.UpdateProvider(ctx, provider.ID, &model.UpdateDNSProviderRequest{Endpoint: str("https://203.0.113.10"), Description: str("d")}); err != nil {
		t.Errorf("unchanged endpoint: %v", err)
	}
	if _, err := svc.UpdateProvider(ctx, provider.ID, &model.UpdateDNSProviderRequest{Endpoint: str("https://203.0.113.99"), Token: str("secret")}); err != nil {
		t.Errorf("new endpoint with the token: %v", err)
	}
}
