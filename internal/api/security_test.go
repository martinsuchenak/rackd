//go:build !short

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/service"
)

// ============================================================================
// Security-Focused Tests
// ============================================================================

// --- CSRF Protection ---

func TestCSRFBlocksSessionPOSTWithoutHeader(t *testing.T) {
	ts := newTestServer(t)
	ts.createAdminUser(t, "admin", "securepassword123")
	cookie := ts.loginUser(t, "admin", "securepassword123")

	// POST without X-Requested-With header should be blocked
	body, _ := json.Marshal(map[string]any{"name": "csrf-test"})
	req := httptest.NewRequest(http.MethodPost, "/api/datacenters", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	// Deliberately NOT setting X-Requested-With

	w := httptest.NewRecorder()
	ts.mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("CSRF: POST without X-Requested-With should be 403, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "CSRF") {
		t.Error("CSRF: response should mention CSRF")
	}
}

func TestCSRFBlocksSessionPUTWithoutHeader(t *testing.T) {
	ts := newTestServer(t)
	ts.createAdminUser(t, "admin", "securepassword123")
	cookie := ts.loginUser(t, "admin", "securepassword123")

	req := httptest.NewRequest(http.MethodPut, "/api/datacenters/fake-id", strings.NewReader(`{"name":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)

	w := httptest.NewRecorder()
	ts.mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("CSRF: PUT without X-Requested-With should be 403, got %d", w.Code)
	}
}

func TestCSRFBlocksSessionDELETEWithoutHeader(t *testing.T) {
	ts := newTestServer(t)
	ts.createAdminUser(t, "admin", "securepassword123")
	cookie := ts.loginUser(t, "admin", "securepassword123")

	req := httptest.NewRequest(http.MethodDelete, "/api/datacenters/fake-id", nil)
	req.AddCookie(cookie)

	w := httptest.NewRecorder()
	ts.mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("CSRF: DELETE without X-Requested-With should be 403, got %d", w.Code)
	}
}

func TestCSRFAllowsGETWithoutHeader(t *testing.T) {
	ts := newTestServer(t)
	ts.createAdminUser(t, "admin", "securepassword123")
	cookie := ts.loginUser(t, "admin", "securepassword123")

	// GET should work without X-Requested-With
	req := httptest.NewRequest(http.MethodGet, "/api/datacenters", nil)
	req.AddCookie(cookie)

	w := httptest.NewRecorder()
	ts.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("CSRF: GET without X-Requested-With should succeed, got %d", w.Code)
	}
}

func TestCSRFNotRequiredForAPIKey(t *testing.T) {
	ts := newTestServer(t)
	userID := ts.createAdminUser(t, "admin", "securepassword123")
	token := ts.createAPIKeyForUser(t, userID)

	// API key auth should not require X-Requested-With
	body, _ := json.Marshal(map[string]any{"name": "apikey-dc"})
	req := httptest.NewRequest(http.MethodPost, "/api/datacenters", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	// No X-Requested-With header

	w := httptest.NewRecorder()
	ts.mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("API key POST without X-Requested-With should succeed, got %d: %s", w.Code, w.Body.String())
	}
}

// --- SQL Injection ---

func TestSQLInjectionInDeviceName(t *testing.T) {
	ts := newTestServer(t)
	userID := ts.createAdminUser(t, "admin", "securepassword123")
	token := ts.createAPIKeyForUser(t, userID)

	payloads := []string{
		"'; DROP TABLE devices; --",
		"\" OR 1=1 --",
		"1; DELETE FROM devices WHERE 1=1",
		"' UNION SELECT * FROM users --",
		"Robert'); DROP TABLE devices;--",
	}

	for _, payload := range payloads {
		t.Run("create_"+payload[:min(len(payload), 20)], func(t *testing.T) {
			w := ts.doRequest(t, http.MethodPost, "/api/devices", map[string]any{
				"name": payload,
			}, token)
			// Should either succeed (treating it as a literal string) or fail validation
			// but NOT cause a 500 or database corruption
			if w.Code == http.StatusInternalServerError {
				t.Errorf("SQL injection payload caused 500: %s", w.Body.String())
			}
		})
	}

	// Verify the database is still functional
	w := ts.doRequest(t, http.MethodGet, "/api/devices", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("database corrupted after SQL injection attempts: status %d", w.Code)
	}
}

func TestSQLInjectionInSearchQuery(t *testing.T) {
	ts := newTestServer(t)
	userID := ts.createAdminUser(t, "admin", "securepassword123")
	token := ts.createAPIKeyForUser(t, userID)

	payloads := []string{
		"' OR '1'='1",
		"'; DROP TABLE devices;--",
		"\" UNION SELECT username, password FROM users --",
		"1 AND 1=1",
	}

	for _, payload := range payloads {
		t.Run(payload[:min(len(payload), 20)], func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/search", nil)
			q := req.URL.Query()
			q.Set("q", payload)
			req.URL.RawQuery = q.Encode()
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			ts.mux.ServeHTTP(w, req)
			if w.Code == http.StatusInternalServerError {
				t.Errorf("SQL injection in search caused 500: %s", w.Body.String())
			}
		})
	}
}

func TestSQLInjectionInListFilters(t *testing.T) {
	ts := newTestServer(t)
	userID := ts.createAdminUser(t, "admin", "securepassword123")
	token := ts.createAPIKeyForUser(t, userID)

	// Injection via query params
	tests := []struct {
		path  string
		param string
		value string
	}{
		{"/api/devices", "tags", "' OR 1=1--"},
		{"/api/devices", "datacenter_id", "' UNION SELECT * FROM users--"},
		{"/api/networks", "datacenter_id", "'; DROP TABLE networks;--"},
		{"/api/devices", "limit", "1;DROP TABLE devices"},
	}

	for _, tt := range tests {
		t.Run(tt.param+"="+tt.value[:min(len(tt.value), 20)], func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			q := req.URL.Query()
			q.Set(tt.param, tt.value)
			req.URL.RawQuery = q.Encode()
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			ts.mux.ServeHTTP(w, req)
			if w.Code == http.StatusInternalServerError {
				t.Errorf("SQL injection via filter caused 500: %s", w.Body.String())
			}
		})
	}
}

// --- XSS Payload Handling ---

func TestXSSPayloadInDeviceFields(t *testing.T) {
	ts := newTestServer(t)
	userID := ts.createAdminUser(t, "admin", "securepassword123")
	token := ts.createAPIKeyForUser(t, userID)

	xssPayloads := []string{
		"<script>alert('xss')</script>",
		"<img src=x onerror=alert(1)>",
		"<svg onload=alert(1)>",
		"javascript:alert(1)",
		"<iframe src='javascript:alert(1)'>",
	}

	for _, payload := range xssPayloads {
		t.Run(payload[:min(len(payload), 20)], func(t *testing.T) {
			w := ts.doRequest(t, http.MethodPost, "/api/devices", map[string]any{
				"name":        payload,
				"description": payload,
				"os":          payload,
			}, token)
			// The API stores data as-is (output encoding is the UI's job)
			// but it should NOT cause a 500
			if w.Code == http.StatusInternalServerError {
				t.Errorf("XSS payload caused 500: %s", w.Body.String())
			}
			if w.Code == http.StatusCreated {
				// Verify the response is JSON (not rendered HTML)
				ct := w.Header().Get("Content-Type")
				if !strings.Contains(ct, "application/json") {
					t.Errorf("response Content-Type should be application/json, got %s", ct)
				}
				// Verify the payload is stored literally, not interpreted
				var device model.Device
				parseJSON(t, w, &device)
				if device.Name != payload {
					t.Errorf("XSS payload was modified: expected %q, got %q", payload, device.Name)
				}
			}
		})
	}
}

// --- Authentication Bypass ---

func TestAuthBypassAttempts(t *testing.T) {
	ts := newTestServer(t)

	protectedEndpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/devices"},
		{http.MethodPost, "/api/devices"},
		{http.MethodGet, "/api/networks"},
		{http.MethodGet, "/api/users"},
		{http.MethodGet, "/api/audit"},
		{http.MethodGet, "/api/keys"},
		{http.MethodGet, "/api/search?q=test"},
	}

	for _, ep := range protectedEndpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			// No auth at all
			w := ts.doRequest(t, ep.method, ep.path, nil, "")
			if w.Code != http.StatusUnauthorized {
				t.Errorf("no auth: expected 401, got %d for %s %s", w.Code, ep.method, ep.path)
			}

			// Malformed Bearer token
			w = ts.doRequest(t, ep.method, ep.path, nil, "")
			req := httptest.NewRequest(ep.method, ep.path, nil)
			req.Header.Set("Authorization", "Bearer")
			rec := httptest.NewRecorder()
			ts.mux.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("malformed bearer: expected 401, got %d", rec.Code)
			}

			// Wrong auth scheme
			req = httptest.NewRequest(ep.method, ep.path, nil)
			req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
			rec = httptest.NewRecorder()
			ts.mux.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("wrong auth scheme: expected 401, got %d", rec.Code)
			}
		})
	}
}

func TestExpiredAPIKeyRejected(t *testing.T) {
	ts := newTestServer(t)
	userID := ts.createAdminUser(t, "admin", "securepassword123")

	// Create an expired API key
	rawToken := "expired-token-123"
	hashed := auth.HashToken(rawToken)
	expired := time.Now().Add(-1 * time.Hour)
	key := &model.APIKey{
		Name:      "expired-key",
		Key:       hashed,
		UserID:    userID,
		ExpiresAt: &expired,
	}
	if err := ts.store.CreateAPIKey(context.Background(), key); err != nil {
		t.Fatalf("failed to create expired key: %v", err)
	}

	w := ts.doRequest(t, http.MethodGet, "/api/devices", nil, rawToken)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expired key: expected 401, got %d", w.Code)
	}
}

func TestLegacyAPIKeyWithoutUserRejected(t *testing.T) {
	ts := newTestServer(t)

	// Create a legacy API key (no user association)
	rawToken := "legacy-token-456"
	hashed := auth.HashToken(rawToken)
	key := &model.APIKey{
		Name:   "legacy-key",
		Key:    hashed,
		UserID: "", // No user
	}
	if err := ts.store.CreateAPIKey(context.Background(), key); err != nil {
		t.Fatalf("failed to create legacy key: %v", err)
	}

	w := ts.doRequest(t, http.MethodGet, "/api/devices", nil, rawToken)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("legacy key: expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Authorization Boundary (User A can't access User B's resources) ---

func TestUserCannotDeleteOtherUsersAPIKey(t *testing.T) {
	ts := newTestServer(t)

	// Create two non-admin users
	adminID := ts.createAdminUser(t, "admin", "securepassword123")
	adminToken := ts.createAPIKeyForUser(t, adminID)

	// Create user B via admin
	w := ts.doRequest(t, http.MethodPost, "/api/users", map[string]any{
		"username":  "userB",
		"password":  "securepassword789",
		"email":     "userb@test.local",
		"full_name": "User B",
	}, adminToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create userB: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var userB map[string]any
	parseJSON(t, w, &userB)
	userBID := userB["id"].(string)

	// Create API key for user B (via admin)
	w = ts.doRequest(t, http.MethodPost, "/api/keys", map[string]any{
		"name":    "userB-key",
		"user_id": userBID,
	}, adminToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create key for userB: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var keyResp map[string]any
	parseJSON(t, w, &keyResp)
	userBKeyID := keyResp["id"].(string)

	// Create a non-admin user A with their own token
	ctx := service.SystemContext(context.Background(), "test")
	userA, _ := ts.svc.Users.Create(ctx, &model.CreateUserRequest{
		Username: "userA",
		Password: "securepassword101",
		Email:    "usera@test.local",
		FullName: "User A",
	})
	userAToken := ts.createAPIKeyForUser(t, userA.ID)

	// User A tries to delete User B's API key
	w = ts.doRequest(t, http.MethodDelete, "/api/keys/"+userBKeyID, nil, userAToken)
	if w.Code == http.StatusNoContent {
		t.Error("user A should NOT be able to delete user B's API key")
	}
}

// --- Request Body Limits ---

func TestRequestBodySizeLimit(t *testing.T) {
	ts := newTestServer(t)
	userID := ts.createAdminUser(t, "admin", "securepassword123")
	token := ts.createAPIKeyForUser(t, userID)

	// Create a body larger than 1MB (MaxRequestBodySize)
	largeBody := make([]byte, 2*1024*1024)
	for i := range largeBody {
		largeBody[i] = 'A'
	}

	req := httptest.NewRequest(http.MethodPost, "/api/devices", bytes.NewReader(largeBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	ts.mux.ServeHTTP(w, req)

	// Should be rejected (400 or 413), not cause a crash
	if w.Code == http.StatusInternalServerError {
		t.Error("large body caused 500 instead of being rejected")
	}
	if w.Code == http.StatusCreated {
		t.Error("large body should not be accepted")
	}
}

// --- Invalid Input Handling ---

func TestInvalidJSONBody(t *testing.T) {
	ts := newTestServer(t)
	userID := ts.createAdminUser(t, "admin", "securepassword123")
	token := ts.createAPIKeyForUser(t, userID)

	invalidBodies := []string{
		"not json at all",
		"{invalid json}",
		"{'single': 'quotes'}",
		"",
		"null",
		"[]",
	}

	for _, body := range invalidBodies {
		t.Run(body[:min(len(body), 15)], func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/devices", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			ts.mux.ServeHTTP(w, req)

			if w.Code == http.StatusInternalServerError {
				t.Errorf("invalid JSON caused 500: body=%q, response=%s", body, w.Body.String())
			}
		})
	}
}

func TestInvalidResourceIDs(t *testing.T) {
	ts := newTestServer(t)
	userID := ts.createAdminUser(t, "admin", "securepassword123")
	token := ts.createAPIKeyForUser(t, userID)

	invalidIDs := []string{
		"nonexistent-uuid",
		"..%2F..%2F..%2Fetc%2Fpasswd",
		"%3Cscript%3Ealert(1)%3C%2Fscript%3E",
		strings.Repeat("a", 1000),
	}

	for _, id := range invalidIDs {
		t.Run(id[:min(len(id), 20)], func(t *testing.T) {
			w := ts.doRequest(t, http.MethodGet, "/api/devices/"+id, nil, token)
			// Should be 404 or 400, never 500
			if w.Code == http.StatusInternalServerError {
				t.Errorf("invalid ID caused 500: id=%q", id)
			}
			if w.Code == http.StatusOK {
				t.Errorf("invalid ID returned 200: id=%q", id)
			}
		})
	}
}

// --- Login Security ---

func TestLoginWithInvalidCredentials(t *testing.T) {
	ts := newTestServer(t)
	ts.createAdminUser(t, "admin", "securepassword123")

	tests := []struct {
		name     string
		username string
		password string
	}{
		{"wrong password", "admin", "wrongpassword123"},
		{"wrong username", "nonexistent", "securepassword123"},
		{"empty username", "", "securepassword123"},
		{"empty password", "admin", ""},
		{"both empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]string{
				"username": tt.username,
				"password": tt.password,
			})
			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			ts.mux.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				t.Errorf("login should fail for %s", tt.name)
			}
			// Should not leak whether the user exists
			if w.Code == http.StatusInternalServerError {
				t.Errorf("login error caused 500 for %s", tt.name)
			}
		})
	}
}

func TestLoginResponseDoesNotLeakPasswordHash(t *testing.T) {
	ts := newTestServer(t)
	ts.createAdminUser(t, "admin", "securepassword123")

	// Successful login
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "securepassword123"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.mux.ServeHTTP(w, req)

	responseBody := w.Body.String()
	if strings.Contains(responseBody, "$2a$") || strings.Contains(responseBody, "$2b$") {
		t.Error("login response contains bcrypt hash")
	}
	if strings.Contains(responseBody, "password") && strings.Contains(responseBody, "hash") {
		t.Error("login response may contain password hash field")
	}
}

// --- Session Security ---

func TestSessionCookieAttributes(t *testing.T) {
	ts := newTestServer(t)
	ts.createAdminUser(t, "admin", "securepassword123")

	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "securepassword123"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.mux.ServeHTTP(w, req)

	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "rackd_session" {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("no session cookie set")
	}

	if !sessionCookie.HttpOnly {
		t.Error("session cookie should be HttpOnly")
	}
	if sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("session cookie SameSite should be Strict, got %v", sessionCookie.SameSite)
	}
	if sessionCookie.Path != "/" {
		t.Errorf("session cookie Path should be '/', got '%s'", sessionCookie.Path)
	}
}

func TestInvalidSessionCookieRejected(t *testing.T) {
	ts := newTestServer(t)

	fakeCookie := &http.Cookie{
		Name:  "rackd_session",
		Value: "completely-fake-session-token",
	}

	w := ts.doSessionRequest(t, http.MethodGet, "/api/auth/me", nil, fakeCookie)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("fake session cookie: expected 401, got %d", w.Code)
	}
}

// --- Privilege Escalation Prevention ---

func TestNonAdminCannotCreateAdmin(t *testing.T) {
	ts := newTestServer(t)

	// Create admin to bootstrap, then create a non-admin
	adminID := ts.createAdminUser(t, "admin", "securepassword123")
	adminToken := ts.createAPIKeyForUser(t, adminID)

	// Create non-admin user
	w := ts.doRequest(t, http.MethodPost, "/api/users", map[string]any{
		"username":  "regular",
		"password":  "securepassword456",
		"email":     "regular@test.local",
		"full_name": "Regular User",
	}, adminToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create regular user: %d: %s", w.Code, w.Body.String())
	}
	var regularUser map[string]any
	parseJSON(t, w, &regularUser)
	regularID := regularUser["id"].(string)
	regularToken := ts.createAPIKeyForUser(t, regularID)

	// Non-admin tries to create an admin user
	w = ts.doRequest(t, http.MethodPost, "/api/users", map[string]any{
		"username":  "hacker-admin",
		"password":  "securepassword789",
		"email":     "hacker@test.local",
		"full_name": "Hacker",
		"is_admin":  true,
	}, regularToken)
	// Should be forbidden
	if w.Code == http.StatusCreated {
		t.Error("non-admin should NOT be able to create admin users")
	}
}

func TestNonAdminCannotEscalateOwnRole(t *testing.T) {
	ts := newTestServer(t)
	adminID := ts.createAdminUser(t, "admin", "securepassword123")
	adminToken := ts.createAPIKeyForUser(t, adminID)

	// Get admin role ID
	w := ts.doRequest(t, http.MethodGet, "/api/roles", nil, adminToken)
	var roles []map[string]any
	parseJSON(t, w, &roles)
	var adminRoleID string
	for _, r := range roles {
		if r["name"] == "admin" {
			adminRoleID = r["id"].(string)
			break
		}
	}

	// Create non-admin
	ctx := service.SystemContext(context.Background(), "test")
	regular, _ := ts.svc.Users.Create(ctx, &model.CreateUserRequest{
		Username: "regular",
		Password: "securepassword456",
		Email:    "regular@test.local",
		FullName: "Regular",
	})
	regularToken := ts.createAPIKeyForUser(t, regular.ID)

	// Non-admin tries to grant themselves admin role
	if adminRoleID != "" {
		w = ts.doRequest(t, http.MethodPost, "/api/users/grant-role", map[string]any{
			"user_id": regular.ID,
			"role_id": adminRoleID,
		}, regularToken)
		if w.Code == http.StatusOK || w.Code == http.StatusCreated || w.Code == http.StatusNoContent {
			t.Error("non-admin should NOT be able to grant themselves admin role")
		}
	}
}

// --- Security Headers ---

func TestSecurityHeadersOnAPIResponse(t *testing.T) {
	ts := newTestServer(t)

	// Health endpoint (no auth) to check headers
	w := ts.doRequest(t, http.MethodGet, "/healthz", nil, "")

	// API responses should have JSON content type
	ct := w.Header().Get("Content-Type")
	if ct != "" && !strings.Contains(ct, "application/json") && !strings.Contains(ct, "text/plain") {
		t.Errorf("unexpected Content-Type: %s", ct)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
