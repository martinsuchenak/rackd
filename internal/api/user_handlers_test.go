package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestUserHandlers(t *testing.T) {
	env := setupExtendedTestHandler(t, true, false, false, false)
	defer env.close()

	t.Run("ListAndGetUsers", func(t *testing.T) {
		w := performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/users", nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/users/test-user-id", nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("CreateAndUpdateUser", func(t *testing.T) {
		body := `{"username":"phase2-user","email":"phase2@example.com","password":"testpass123","full_name":"Phase Two"}`
		req := authReq(httptest.NewRequest("POST", "/api/users", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := performRequest(env.mux, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var created model.UserResponse
		if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
			t.Fatalf("failed to decode user response: %v", err)
		}

		updateReq := authReq(httptest.NewRequest("PUT", "/api/users/"+created.ID, bytes.NewBufferString(`{"full_name":"Updated User"}`)))
		updateReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, updateReq)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("UpdateUser_InvalidJSON", func(t *testing.T) {
		req := authReq(httptest.NewRequest("PUT", "/api/users/test-user-id", bytes.NewBufferString("{")))
		w := performRequest(env.mux, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("CreateUser_DuplicateUsername", func(t *testing.T) {
		req := authReq(httptest.NewRequest("POST", "/api/users", bytes.NewBufferString(`{"username":"testuser","email":"dupe@example.com","password":"testpass123"}`)))
		req.Header.Set("Content-Type", "application/json")
		w := performRequest(env.mux, req)
		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("GetUser_NotFound", func(t *testing.T) {
		w := performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/users/nonexistent", nil)))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("UpdateUser_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("PUT", "/api/users/nonexistent", bytes.NewBufferString(`{"full_name":"Missing"}`)))
		req.Header.Set("Content-Type", "application/json")
		w := performRequest(env.mux, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("ChangeAndResetPassword", func(t *testing.T) {
		changeReq := authReq(httptest.NewRequest("POST", "/api/users/test-user-id/password", bytes.NewBufferString(`{"old_password":"test-password","new_password":"new-password-123"}`)))
		changeReq.Header.Set("Content-Type", "application/json")
		w := performRequest(env.mux, changeReq)
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
		}

		createReq := authReq(httptest.NewRequest("POST", "/api/users", bytes.NewBufferString(`{"username":"reset-target","email":"reset@example.com","password":"testpass123"}`)))
		createReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, createReq)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", w.Code)
		}
		var created model.UserResponse
		if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
			t.Fatalf("failed to decode reset target: %v", err)
		}

		resetReq := authReq(httptest.NewRequest("POST", "/api/users/"+created.ID+"/reset-password", bytes.NewBufferString(`{"new_password":"reset-password-123"}`)))
		resetReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, resetReq)
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("ResetPassword_ForbiddenWithoutPermission", func(t *testing.T) {
		targetReq := authReq(httptest.NewRequest("POST", "/api/users", bytes.NewBufferString(`{"username":"limited-reset-target","email":"limited-reset@example.com","password":"testpass123"}`)))
		targetReq.Header.Set("Content-Type", "application/json")
		w := performRequest(env.mux, targetReq)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var target model.UserResponse
		if err := json.Unmarshal(w.Body.Bytes(), &target); err != nil {
			t.Fatalf("failed to decode target user: %v", err)
		}

		_, limitedToken := env.createAPIUser(t, "limited-user-phase2")
		resetReq := authReqWithToken(httptest.NewRequest("POST", "/api/users/"+target.ID+"/reset-password", bytes.NewBufferString(`{"new_password":"reset-password-123"}`)), limitedToken)
		resetReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, resetReq)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}
