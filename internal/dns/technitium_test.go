package dns

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTechnitiumTestServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(handler))
}

func TestTechnitiumClientDoAPIHandlesMalformedAndErrorResponses(t *testing.T) {
	t.Run("http status error", func(t *testing.T) {
		server := newTechnitiumTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "boom", http.StatusBadGateway)
		})
		defer server.Close()

		client := NewTechnitiumClientWithHTTPClient(server.URL, "token", server.Client())
		if err := client.HealthCheck(context.Background()); err == nil {
			t.Fatal("expected HTTP status failure")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		server := newTechnitiumTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{"))
		})
		defer server.Close()

		client := NewTechnitiumClientWithHTTPClient(server.URL, "token", server.Client())
		if err := client.HealthCheck(context.Background()); err == nil {
			t.Fatal("expected JSON decode failure")
		}
	})

	t.Run("api error payload", func(t *testing.T) {
		server := newTechnitiumTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"error","errorMessage":"bad token"}`))
		})
		defer server.Close()

		client := NewTechnitiumClientWithHTTPClient(server.URL, "token", server.Client())
		if err := client.HealthCheck(context.Background()); err == nil {
			t.Fatal("expected API error")
		}
	})
}

func TestTechnitiumClientListZonesHealthAndZoneExists(t *testing.T) {
	server := newTechnitiumTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/zones/list":
			_, _ = w.Write([]byte(`{"status":"ok","response":{"zones":[{"name":"example.test","type":"Primary"},{"name":"other.test","type":"Primary"}]}}`))
		case "/api/status":
			_, _ = w.Write([]byte(`{"status":"ok","response":{"version":"1.0"}}`))
		default:
			http.NotFound(w, r)
		}
	})
	defer server.Close()

	client := NewTechnitiumClientWithHTTPClient(server.URL, "token", server.Client())

	zones, err := client.ListZones(context.Background())
	if err != nil {
		t.Fatalf("ListZones failed: %v", err)
	}
	if len(zones) != 2 || zones[0] != "example.test" {
		t.Fatalf("unexpected zones: %v", zones)
	}

	exists, err := client.ZoneExists(context.Background(), "example.test")
	if err != nil {
		t.Fatalf("ZoneExists failed: %v", err)
	}
	if !exists {
		t.Fatal("expected zone to exist")
	}

	if err := client.HealthCheck(context.Background()); err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
}

func TestTechnitiumClientRecordOperations(t *testing.T) {
	var deleted bool
	server := newTechnitiumTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/records/add":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/records/delete":
			deleted = true
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/zones/records/get":
			if deleted {
				_, _ = w.Write([]byte(`{"status":"ok","response":{"records":[]}}`))
				return
			}
			zone := r.URL.Query().Get("zone")
			domain := r.URL.Query().Get("domain")
			if zone == "" || domain == "" {
				http.Error(w, "missing query", http.StatusBadRequest)
				return
			}
			body := fmt.Sprintf(`{"status":"ok","response":{"records":[{"name":"www","type":"A","ttl":300,"rData":{"ipAddress":"10.0.0.5"}},{"name":"alias","type":"CNAME","ttl":300,"rData":{"cname":"target.example.test"}}]}}`)
			_, _ = w.Write([]byte(body))
		default:
			http.NotFound(w, r)
		}
	})
	defer server.Close()

	client := NewTechnitiumClientWithHTTPClient(server.URL, "token", server.Client())

	record, err := client.GetRecord(context.Background(), "example.test", "www", "A")
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}
	if record.Value != "10.0.0.5" {
		t.Fatalf("unexpected record value: %+v", record)
	}

	list, err := client.ListRecords(context.Background(), "example.test")
	if err != nil {
		t.Fatalf("ListRecords failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 records, got %d", len(list))
	}

	if err := client.CreateRecord(context.Background(), "example.test", &Record{Name: "www", Type: "A", Value: "10.0.0.5", TTL: 300}); err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}
	if err := client.UpdateRecord(context.Background(), "example.test", &Record{Name: "www", Type: "A", Value: "10.0.0.6", TTL: 300}); err != nil {
		t.Fatalf("UpdateRecord failed: %v", err)
	}
	if !deleted {
		t.Fatal("expected update path to delete old record after create")
	}
}

// TestTechnitiumClientLegacyTokenFallback covers servers that reject the
// Authorization header (404/401/403) but accept the legacy token query
// parameter: the client must retry once with the parameter and succeed.
func TestTechnitiumClientLegacyTokenFallback(t *testing.T) {
	var sawQueryParamToken bool
	server := newTechnitiumTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/status" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") == "Bearer secret-token" && r.URL.Query().Get("token") == "" {
			// Header-only request rejected (observed in the field even on
			// v15.x deployments behind certain configurations).
			http.Error(w, `{"status":"error","errorMessage":"token required"}`, http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("token") == "secret-token" {
			sawQueryParamToken = true
			_, _ = w.Write([]byte(`{"status":"ok","response":{"version":"15.2"}}`))
			return
		}
		http.Error(w, `{"status":"error"}`, http.StatusUnauthorized)
	})
	defer server.Close()

	client := NewTechnitiumClientWithHTTPClient(server.URL, "secret-token", server.Client())
	if err := client.HealthCheck(context.Background()); err != nil {
		t.Fatalf("HealthCheck with legacy fallback failed: %v", err)
	}
	if !sawQueryParamToken {
		t.Fatal("expected fallback retry with token query parameter")
	}
}

// TestTechnitiumClientEndpointNormalization verifies trailing slashes and a
// trailing /api are stripped so paths like /api/api/status are never built.
func TestTechnitiumClientEndpointNormalization(t *testing.T) {
	for _, endpoint := range []string{"http://host:5380", "http://host:5380/", "http://host:5380/api", "http://host:5380/api/"} {
		c := NewTechnitiumClientWithHTTPClient(endpoint, "tok", nil)
		if c.endpoint != "http://host:5380" {
			t.Errorf("endpoint %q normalized to %q, want http://host:5380", endpoint, c.endpoint)
		}
	}
}
