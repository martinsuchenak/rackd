package client

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	os.Unsetenv("RACKD_SERVER_URL")
	os.Unsetenv("RACKD_TOKEN")

	cfg := LoadConfig()

	if cfg.ServerURL != "http://localhost:8080" {
		t.Errorf("expected default ServerURL, got %s", cfg.ServerURL)
	}
	if cfg.Timeout != "30s" {
		t.Errorf("expected default Timeout, got %s", cfg.Timeout)
	}
	if cfg.Output != "table" {
		t.Errorf("expected default Output, got %s", cfg.Output)
	}
}

func TestLoadConfig_EnvOverride(t *testing.T) {
	os.Setenv("RACKD_SERVER_URL", "http://test:9090")
	os.Setenv("RACKD_TOKEN", "test-token")
	defer os.Unsetenv("RACKD_SERVER_URL")
	defer os.Unsetenv("RACKD_TOKEN")

	cfg := LoadConfig()

	if cfg.ServerURL != "http://test:9090" {
		t.Errorf("expected env ServerURL, got %s", cfg.ServerURL)
	}
	if cfg.Token != "test-token" {
		t.Errorf("expected env Token, got %s", cfg.Token)
	}
}

func TestConfig_GetTimeout(t *testing.T) {
	cfg := &Config{Timeout: "10s"}
	if cfg.GetTimeout().Seconds() != 10 {
		t.Errorf("expected 10s timeout")
	}

	cfg.Timeout = "invalid"
	if cfg.GetTimeout().Seconds() != 30 {
		t.Errorf("expected default 30s for invalid timeout")
	}
}

func TestClient_DoRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("expected auth header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected content-type header")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	cfg := &Config{ServerURL: server.URL, Token: "test-token", Timeout: "5s"}
	client := NewClient(cfg)

	resp, err := client.DoRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHandleError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request","code":"INVALID_INPUT"}`))
	}))
	defer server.Close()

	cfg := &Config{ServerURL: server.URL, Timeout: "5s"}
	client := NewClient(cfg)

	resp, _ := client.DoRequest("GET", "/test", nil)
	err := HandleError(resp)

	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "INVALID_INPUT: bad request" {
		t.Errorf("unexpected error message: %v", err)
	}
}
