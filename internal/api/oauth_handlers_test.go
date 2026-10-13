package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/martinsuchenak/rackd/internal/service"
)

func newOAuthTestServer(t *testing.T) *testServer {
	t.Helper()
	ts := newTestServer(t)
	// OAuth routes are only registered when the OAuth service is configured,
	// as server.go does when MCP_OAUTH_ENABLED=true.
	ts.svc.OAuth = service.NewOAuthService(ts.store, ts.sessionManager, "http://localhost")
	mux := http.NewServeMux()
	ts.handler.RegisterRoutes(mux)
	ts.mux = mux
	return ts
}

func TestOAuthRegisterWithoutScopeIsClientError(t *testing.T) {
	ts := newOAuthTestServer(t)

	w := ts.doRequest(t, http.MethodPost, "/mcp-oauth/register", map[string]any{
		"client_name":   "no-scope-client",
		"redirect_uris": []string{"https://client.example.test/callback"},
	}, "")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for registration without scope, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding error body: %v", err)
	}
	if body["error"] != "invalid_client_metadata" {
		t.Errorf("expected error=invalid_client_metadata, got %q", body["error"])
	}
}

func TestOAuthRegisterWithScopeSucceeds(t *testing.T) {
	ts := newOAuthTestServer(t)

	w := ts.doRequest(t, http.MethodPost, "/mcp-oauth/register", map[string]any{
		"client_name":   "scoped-client",
		"redirect_uris": []string{"https://client.example.test/callback"},
		"scope":         "devices:list",
	}, "")

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for registration with scope, got %d: %s", w.Code, w.Body.String())
	}
}
