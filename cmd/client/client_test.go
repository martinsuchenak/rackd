package client

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	_ = os.Unsetenv("RACKD_SERVER_URL")
	_ = os.Unsetenv("RACKD_TOKEN")

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
	_ = os.Setenv("RACKD_SERVER_URL", "http://test:9090")
	_ = os.Setenv("RACKD_TOKEN", "test-token")
	defer func() { _ = os.Unsetenv("RACKD_SERVER_URL") }()
	defer func() { _ = os.Unsetenv("RACKD_TOKEN") }()
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
		_, _ = w.Write([]byte(`{"status":"ok"}`))
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

func TestClient_DoRequestDetectsNonV1Responses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/devices": // a pre-v1 server: GET falls through to the web UI
			if r.Method != http.MethodGet {
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte("<!doctype html>"))
		case "/api/v1/down": // a proxy error page
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadGateway)
		case "/api/v1/wrong-method": // a v1 server's own 405
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = w.Write([]byte(`{"code":"METHOD_NOT_ALLOWED","error":"method not allowed"}`))
		default:
			w.Header().Set("Content-Type", "text/html")
		}
	}))
	defer server.Close()
	client := NewClient(&Config{ServerURL: server.URL, Timeout: "5s"})

	for _, method := range []string{"GET", "POST"} {
		if _, err := client.DoRequest(method, "/api/v1/devices", nil); !errors.Is(err, ErrNotV1Response) {
			t.Errorf("%s against a pre-v1 server: err = %v, want ErrNotV1Response", method, err)
		}
	}
	for _, path := range []string{"/api/v1/down", "/api/v1/wrong-method", "/login"} {
		resp, err := client.DoRequest("POST", path, nil)
		if err != nil {
			t.Errorf("%s: %v, want the response passed through", path, err)
			continue
		}
		_ = resp.Body.Close()
	}
}

func TestHandleError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad request","code":"INVALID_INPUT"}`))
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
