package api

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

// The approval accepted any scope, capped only by the user's permissions, so a
// consent screen reading devices:list could yield a write token (found by a
// Strix run). An approval must now present the consent screen's token and
// grant exactly what it showed.
func TestOAuthApprovalGrantsOnlyWhatConsentShowed(t *testing.T) {
	ts := newOAuthTestServer(t)
	ts.createAdminUser(t, "admin", "securepassword123")
	cookie := ts.loginUser(t, "admin", "securepassword123")

	const redirect = "https://client.example.test/callback"
	w := ts.doRequest(t, http.MethodPost, "/mcp-oauth/register", map[string]any{
		"client_name": "consent-client",
		// A second registered URI, so swapping it in gets past the redirect
		// check and has to be caught by the consent token.
		"redirect_uris":              []string{redirect, "https://client.example.test/other"},
		"scope":                      "devices:list devices:read devices:create devices:update",
		"token_endpoint_auth_method": "none",
	}, "")
	if w.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", w.Code, w.Body.String())
	}
	var client map[string]any
	parseJSON(t, w, &client)
	clientID := client["client_id"].(string)

	verifier := "consent-test-verifier-0123456789abcdefghijklmnop"
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	consentFor := func(scope string, cookie *http.Cookie) *httptest.ResponseRecorder {
		q := url.Values{"response_type": {"code"}, "client_id": {clientID}, "redirect_uri": {redirect},
			"scope": {scope}, "state": {"s1"}, "code_challenge": {challenge}, "code_challenge_method": {"S256"}}
		req := httptest.NewRequest(http.MethodGet, "/mcp-oauth/authorize?"+q.Encode(), nil)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		req.AddCookie(cookie)
		w := httptest.NewRecorder()
		ts.mux.ServeHTTP(w, req)
		return w
	}

	// A viewer holds none of devices:create, so there is nothing to consent
	// to. An empty approval would fall back to the client's registered scope
	// (devices:list among it), which the screen never showed.
	ts.createReadOnlyUser(t, "viewer", "securepassword123")
	if w := consentFor("devices:create", ts.loginUser(t, "viewer", "securepassword123")); w.Code != http.StatusBadRequest ||
		!strings.Contains(w.Body.String(), "invalid_scope") {
		t.Fatalf("consent with nothing grantable: %d %s, want 400 invalid_scope", w.Code, w.Body.String())
	}

	w = consentFor("devices:list", cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("consent data: %d %s", w.Code, w.Body.String())
	}
	var consent map[string]any
	parseJSON(t, w, &consent)
	if consent["scope"] != "devices:list" || consent["consent_token"] == "" {
		t.Fatalf("consent data %v, want devices:list and a token", consent)
	}

	approve := func(changes map[string]any) *httptest.ResponseRecorder {
		body := map[string]any{"client_id": clientID, "redirect_uri": redirect, "scope": "devices:list", "state": "s1",
			"code_challenge": challenge, "code_challenge_method": "S256", "consent_token": consent["consent_token"], "approved": true}
		for k, v := range changes {
			body[k] = v
		}
		return ts.doSessionRequest(t, http.MethodPost, "/mcp-oauth/authorize", body, cookie)
	}

	for name, changes := range map[string]map[string]any{
		"inflated scope":   {"scope": "devices:list devices:create devices:update"},
		"no consent token": {"consent_token": ""},
		"another redirect": {"redirect_uri": "https://client.example.test/other"},
		"forged token":     {"consent_token": "9999999999.forged"},
		"another verifier": {"code_challenge": base64.RawURLEncoding.EncodeToString(make([]byte, 32))},
	} {
		if w := approve(changes); w.Code != http.StatusBadRequest {
			t.Errorf("approve with %s: %d %s, want 400", name, w.Code, w.Body.String())
		}
	}

	w = approve(nil)
	if w.Code != http.StatusOK {
		t.Fatalf("approve as shown: %d %s", w.Code, w.Body.String())
	}
	var redirected map[string]string
	parseJSON(t, w, &redirected)
	u, _ := url.Parse(redirected["redirect_uri"])
	code := u.Query().Get("code")
	if code == "" {
		t.Fatalf("no code in %s", redirected["redirect_uri"])
	}
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {redirect},
		"client_id": {clientID}, "code_verifier": {verifier}}
	req := httptest.NewRequest(http.MethodPost, "/mcp-oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	ts.mux.ServeHTTP(w, req)
	var tok map[string]any
	parseJSON(t, w, &tok)
	if tok["scope"] != "devices:list" {
		t.Errorf("token scope %v, want only devices:list", tok["scope"])
	}
}
