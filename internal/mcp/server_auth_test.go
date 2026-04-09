package mcp

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleRequestOptionsBypassesAuth(t *testing.T) {
	srv, store := newTestServerWithAuth(t)
	defer store.Close()

	req := httptest.NewRequest(http.MethodOptions, "/mcp", nil)
	w := httptest.NewRecorder()

	srv.HandleRequest(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for OPTIONS, got %d", w.Code)
	}
}

func TestWriteUnauthorizedHeaders(t *testing.T) {
	srv, store := newTestServerWithAuth(t)
	defer store.Close()

	w := httptest.NewRecorder()
	srv.writeUnauthorized(w)
	if got := w.Header().Get("WWW-Authenticate"); got == "" {
		t.Fatal("expected bearer challenge header")
	}

	srv.oauthEnabled = true
	w = httptest.NewRecorder()
	srv.writeUnauthorized(w)
	if got := w.Header().Get("WWW-Authenticate"); got != `Bearer resource_metadata="/.well-known/oauth-protected-resource"` {
		t.Fatalf("unexpected oauth challenge header: %q", got)
	}
}

func TestHandleRequestWithAuthMalformedBearer(t *testing.T) {
	srv, store := newTestServerWithAuth(t)
	defer store.Close()

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()

	srv.HandleRequest(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for empty bearer token, got %d", w.Code)
	}
}
