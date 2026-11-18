package service

import (
	"context"
	"sort"
	"strings"
	"testing"

	dnspkg "github.com/martinsuchenak/rackd/internal/dns"
	"github.com/martinsuchenak/rackd/internal/storage/storagetest"
	"github.com/martinsuchenak/rackd/pkg/model"
)

// listOnlyProvider serves a fixed record listing; nothing else is called
// by an import.
type listOnlyProvider struct {
	dnspkg.Provider
	records []*dnspkg.Record
}

func (p *listOnlyProvider) ListRecords(context.Context, string) ([]*dnspkg.Record, error) {
	return p.records, nil
}

func TestImportFromDNS_DeleteMissing(t *testing.T) {
	store := storagetest.New(t)
	ctx := SystemContext(context.Background(), "test")

	providerCfg := &model.DNSProviderConfig{Name: "tech", Type: model.DNSProviderTypeTechnitium, Endpoint: "http://dns.example.test"}
	if err := store.CreateDNSProvider(ctx, providerCfg); err != nil {
		t.Fatal(err)
	}
	zone := &model.DNSZone{Name: "example.test", ProviderID: providerCfg.ID, TTL: 300}
	if err := store.CreateDNSZone(ctx, zone); err != nil {
		t.Fatal(err)
	}
	for _, r := range []struct {
		name   string
		status model.RecordSyncStatus
	}{
		{"kept", model.RecordSyncStatusSynced},     // still on the provider
		{"gone", model.RecordSyncStatusSynced},     // removed on the provider
		{"pending", model.RecordSyncStatusPending}, // not pushed yet
		{"failed", model.RecordSyncStatusFailed},   // push failed
	} {
		if err := store.CreateDNSRecord(ctx, &model.DNSRecord{ZoneID: zone.ID, Name: r.name, Type: "A", Value: "10.0.0.1", TTL: 300, SyncStatus: r.status}); err != nil {
			t.Fatal(err)
		}
	}

	svc := NewDNSService(store, nil)
	svc.providerCache[providerCfg.ID] = &listOnlyProvider{records: []*dnspkg.Record{
		{Name: "kept", Type: "A", Value: "10.0.0.1", TTL: 300},
		{Name: "new", Type: "A", Value: "10.0.0.2", TTL: 300},
	}}

	names := func() string {
		recs, err := store.ListDNSRecords(ctx, &model.DNSRecordFilter{ZoneID: zone.ID})
		if err != nil {
			t.Fatal(err)
		}
		var n []string
		for _, r := range recs {
			n = append(n, r.Name)
		}
		sort.Strings(n)
		return strings.Join(n, ",")
	}

	// Without the option nothing is deleted.
	res, err := svc.ImportFromDNS(ctx, zone.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.Deleted != 0 || names() != "failed,gone,kept,new,pending" {
		t.Fatalf("plain import: deleted %d, records %s", res.Deleted, names())
	}

	res, err = svc.ImportFromDNSWithOptions(ctx, zone.ID, model.ImportDNSOptions{DeleteMissing: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Deleted != 1 || !res.Success {
		t.Fatalf("delete_missing: %+v", res)
	}
	if got := names(); got != "failed,kept,new,pending" {
		t.Fatalf("records after delete_missing: %s", got)
	}
}
