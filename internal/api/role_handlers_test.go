package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestRoleHandlers(t *testing.T) {
	env := setupExtendedTestHandler(t, false, false, false, false)
	defer env.close()

	var permissionID string
	perms, err := env.store.ListPermissions(context.Background(), &model.PermissionFilter{})
	if err != nil {
		t.Fatalf("failed to list permissions: %v", err)
	}
	if len(perms) == 0 {
		t.Fatal("expected seeded permissions")
	}
	permissionID = perms[0].ID

	t.Run("CreateGetUpdateDeleteRole", func(t *testing.T) {
		req := authReq(httptest.NewRequest("POST", "/api/roles", bytes.NewBufferString(`{"name":"phase2-role","description":"phase 2 role","permissions":["`+permissionID+`"]}`)))
		req.Header.Set("Content-Type", "application/json")
		w := performRequest(env.mux, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var created model.RoleResponse
		if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
			t.Fatalf("failed to decode role response: %v", err)
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/roles/"+created.ID, nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		updateReq := authReq(httptest.NewRequest("PUT", "/api/roles/"+created.ID, bytes.NewBufferString(`{"description":"updated role"}`)))
		updateReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, updateReq)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/roles/"+created.ID+"/permissions", nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		grantReq := authReq(httptest.NewRequest("POST", "/api/users/grant-role", bytes.NewBufferString(`{"user_id":"test-user-id","role_id":"`+created.ID+`"}`)))
		grantReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, grantReq)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/users/test-user-id/roles", nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		revokeReq := authReq(httptest.NewRequest("POST", "/api/users/revoke-role", bytes.NewBufferString(`{"user_id":"test-user-id","role_id":"`+created.ID+`"}`)))
		revokeReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, revokeReq)
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("DELETE", "/api/roles/"+created.ID, nil)))
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Role_InvalidJSON", func(t *testing.T) {
		w := performRequest(env.mux, authReq(httptest.NewRequest("POST", "/api/roles", bytes.NewBufferString("{"))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("PUT", "/api/roles/test-role", bytes.NewBufferString("{"))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("Role_NotFound", func(t *testing.T) {
		w := performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/roles/nonexistent", nil)))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}

		updateReq := authReq(httptest.NewRequest("PUT", "/api/roles/nonexistent", bytes.NewBufferString(`{"description":"missing"}`)))
		updateReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, updateReq)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("DELETE", "/api/roles/nonexistent", nil)))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Role_ForbiddenWithoutPermission", func(t *testing.T) {
		_, limitedToken := env.createAPIUser(t, "limited-role-user")

		w := performRequest(env.mux, authReqWithToken(httptest.NewRequest("GET", "/api/roles", nil), limitedToken))
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}

		createReq := authReqWithToken(httptest.NewRequest("POST", "/api/roles", bytes.NewBufferString(`{"name":"limited-role"}`)), limitedToken)
		createReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, createReq)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}
