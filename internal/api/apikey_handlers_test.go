package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIKeyHandlers(t *testing.T) {
	env := setupExtendedTestHandler(t, false, false, false, false)
	defer env.close()

	t.Run("ListCreateGetDeleteAPIKey", func(t *testing.T) {
		w := performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/keys", nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		createReq := authReq(httptest.NewRequest("POST", "/api/keys", bytes.NewBufferString(`{"name":"phase2-key","description":"phase 2"}`)))
		createReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, createReq)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var created struct {
			ID  string `json:"id"`
			Key string `json:"key"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
			t.Fatalf("failed to decode created key: %v", err)
		}
		if created.Key == "" {
			t.Fatal("expected plaintext API key in create response")
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/keys/"+created.ID, nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("DELETE", "/api/keys/"+created.ID, nil)))
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", w.Code)
		}
	})

	t.Run("CreateAPIKey_InvalidJSON", func(t *testing.T) {
		w := performRequest(env.mux, authReq(httptest.NewRequest("POST", "/api/keys", bytes.NewBufferString("{"))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("APIKey_NotFound", func(t *testing.T) {
		w := performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/keys/nonexistent", nil)))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("DELETE", "/api/keys/nonexistent", nil)))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("APIKey_ForbiddenWithoutPermission", func(t *testing.T) {
		_, limitedToken := env.createAPIUser(t, "limited-apikey-user")

		req := authReqWithToken(httptest.NewRequest("GET", "/api/keys", nil), limitedToken)
		w := performRequest(env.mux, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}

		createReq := authReqWithToken(httptest.NewRequest("POST", "/api/keys", bytes.NewBufferString(`{"name":"limited-key"}`)), limitedToken)
		createReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, createReq)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}
