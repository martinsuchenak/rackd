package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestDNSHandlers(t *testing.T) {
	env := setupDNSTestHandler(t)
	defer env.close()

	network := &model.Network{ID: "dns-net", Name: "dns-net", Subnet: "10.60.0.0/24"}
	if err := env.store.CreateNetwork(context.Background(), network); err != nil {
		t.Fatalf("failed to seed network: %v", err)
	}

	t.Run("ProviderAndZoneCRUD", func(t *testing.T) {
		createProviderReq := authReq(httptest.NewRequest("POST", "/api/dns/providers", bytes.NewBufferString(`{"name":"phase2-dns","type":"bind","endpoint":"https://dns.example.test","token":"super-secret","description":"phase 2 provider"}`)))
		createProviderReq.Header.Set("Content-Type", "application/json")
		w := performRequest(env.mux, createProviderReq)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var provider model.DNSProviderConfig
		if err := json.Unmarshal(w.Body.Bytes(), &provider); err != nil {
			t.Fatalf("failed to decode provider: %v", err)
		}
		if provider.ID == "" {
			t.Fatal("expected provider ID")
		}
		if provider.Token != "" {
			t.Fatal("provider token should not be exposed")
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/dns/providers", nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/dns/providers/"+provider.ID, nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		updateProviderReq := authReq(httptest.NewRequest("PUT", "/api/dns/providers/"+provider.ID, bytes.NewBufferString(`{"description":"updated provider"}`)))
		updateProviderReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, updateProviderReq)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		createZoneReq := authReq(httptest.NewRequest("POST", "/api/dns/zones", bytes.NewBufferString(`{"name":"example.test","provider_id":"`+provider.ID+`","network_id":"dns-net","auto_sync":false,"create_ptr":false,"ttl":300}`)))
		createZoneReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, createZoneReq)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var zone model.DNSZone
		if err := json.Unmarshal(w.Body.Bytes(), &zone); err != nil {
			t.Fatalf("failed to decode zone: %v", err)
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/dns/zones", nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/dns/providers/"+provider.ID+"/zones", nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/dns/zones/"+zone.ID, nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		updateZoneReq := authReq(httptest.NewRequest("PUT", "/api/dns/zones/"+zone.ID, bytes.NewBufferString(`{"description":"updated zone","ttl":600}`)))
		updateZoneReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, updateZoneReq)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("DELETE", "/api/dns/zones/"+zone.ID, nil)))
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("DELETE", "/api/dns/providers/"+provider.ID, nil)))
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("DNSRecordGetUpdateDeleteLinkPromote", func(t *testing.T) {
		provider := &model.DNSProviderConfig{
			Name:     "records-provider",
			Type:     model.DNSProviderTypeBIND,
			Endpoint: "https://dns.example.test",
			Token:    "encrypted-placeholder",
		}
		if err := env.store.CreateDNSProvider(context.Background(), provider); err != nil {
			t.Fatalf("failed to seed provider: %v", err)
		}

		networkID := network.ID
		zone := &model.DNSZone{
			Name:           "records.example.test",
			ProviderID:     provider.ID,
			NetworkID:      &networkID,
			TTL:            300,
			LastSyncStatus: model.SyncStatusSuccess,
		}
		if err := env.store.CreateDNSZone(context.Background(), zone); err != nil {
			t.Fatalf("failed to seed zone: %v", err)
		}

		device := &model.Device{
			Name:     "linked-device",
			Hostname: "linked-device",
			Addresses: []model.Address{
				{ID: "addr-1", IP: "10.60.0.20", Type: "ipv4", NetworkID: network.ID},
			},
		}
		if err := env.store.CreateDevice(context.Background(), device); err != nil {
			t.Fatalf("failed to seed device: %v", err)
		}

		record := &model.DNSRecord{
			ZoneID:     zone.ID,
			Name:       "host-a",
			Type:       "A",
			Value:      "10.60.0.20",
			TTL:        300,
			SyncStatus: model.RecordSyncStatusPending,
		}
		if err := env.store.CreateDNSRecord(context.Background(), record); err != nil {
			t.Fatalf("failed to seed record: %v", err)
		}

		promoteRecord := &model.DNSRecord{
			ZoneID:     zone.ID,
			Name:       "promote-me",
			Type:       "A",
			Value:      "10.60.0.30",
			TTL:        300,
			SyncStatus: model.RecordSyncStatusPending,
		}
		if err := env.store.CreateDNSRecord(context.Background(), promoteRecord); err != nil {
			t.Fatalf("failed to seed promote record: %v", err)
		}

		w := performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/dns/zones/"+zone.ID+"/records", nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/dns/records/"+record.ID, nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		updateReq := authReq(httptest.NewRequest("PUT", "/api/dns/records/"+record.ID, bytes.NewBufferString(`{"ttl":600}`)))
		updateReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, updateReq)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		linkReq := authReq(httptest.NewRequest("POST", "/api/dns/records/"+record.ID+"/link", bytes.NewBufferString(`{"device_id":"`+device.ID+`","address_id":"addr-1"}`)))
		linkReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, linkReq)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		promoteReq := authReq(httptest.NewRequest("POST", "/api/dns/records/"+promoteRecord.ID+"/promote", bytes.NewBufferString(`{"name":"promoted-device"}`)))
		promoteReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, promoteReq)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("DELETE", "/api/dns/records/"+record.ID, nil)))
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("DNS_InvalidJSONForbiddenAndNotFound", func(t *testing.T) {
		w := performRequest(env.mux, authReq(httptest.NewRequest("POST", "/api/dns/providers", bytes.NewBufferString("{"))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("POST", "/api/dns/zones", bytes.NewBufferString("{"))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("PUT", "/api/dns/records/missing", bytes.NewBufferString("{"))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/dns/providers/missing", nil)))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/dns/zones/missing", nil)))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/dns/records/missing", nil)))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}

		_, limitedToken := createAPIUserForStore(t, env.store, "limited-dns-user")

		w = performRequest(env.mux, authReqWithToken(httptest.NewRequest("GET", "/api/dns/providers", nil), limitedToken))
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}

		req := authReqWithToken(httptest.NewRequest("POST", "/api/dns/zones", bytes.NewBufferString(`{"name":"limited.example","provider_id":"missing"}`)), limitedToken)
		req.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}
