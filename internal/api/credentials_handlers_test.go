package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestCredentialHandlers(t *testing.T) {
	env := setupExtendedTestHandler(t, false, true, false, false)
	defer env.close()

	t.Run("CreateGetUpdateDeleteCredential", func(t *testing.T) {
		body := `{"name":"phase2-ssh","type":"ssh_key","ssh_username":"admin","ssh_key_id":"key-1"}`
		req := authReq(httptest.NewRequest("POST", "/api/credentials", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := performRequest(env.mux, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var created model.CredentialResponse
		if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
			t.Fatalf("failed to decode credential: %v", err)
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/credentials/"+created.ID, nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		updateReq := authReq(httptest.NewRequest("PUT", "/api/credentials/"+created.ID, bytes.NewBufferString(`{"name":"phase2-ssh-updated","type":"ssh_key","ssh_username":"root","ssh_key_id":"key-2"}`)))
		updateReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, updateReq)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("DELETE", "/api/credentials/"+created.ID, nil)))
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", w.Code)
		}
	})

	t.Run("CreateCredential_InvalidJSON", func(t *testing.T) {
		w := performRequest(env.mux, authReq(httptest.NewRequest("POST", "/api/credentials", bytes.NewBufferString("{"))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("Credential_NotFound", func(t *testing.T) {
		w := performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/credentials/nonexistent", nil)))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}

		updateReq := authReq(httptest.NewRequest("PUT", "/api/credentials/nonexistent", bytes.NewBufferString(`{"name":"missing","type":"ssh_key","ssh_username":"root","ssh_key_id":"key-2"}`)))
		updateReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, updateReq)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("DELETE", "/api/credentials/nonexistent", nil)))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("Credential_ForbiddenWithoutPermission", func(t *testing.T) {
		_, limitedToken := env.createAPIUser(t, "limited-credential-user")

		req := authReqWithToken(httptest.NewRequest("GET", "/api/credentials", nil), limitedToken)
		w := performRequest(env.mux, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}

		createReq := authReqWithToken(httptest.NewRequest("POST", "/api/credentials", bytes.NewBufferString(`{"name":"limited-ssh","type":"ssh_key","ssh_username":"admin","ssh_key_id":"key-1"}`)), limitedToken)
		createReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, createReq)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}
