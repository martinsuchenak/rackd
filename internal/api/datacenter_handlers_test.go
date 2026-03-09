package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/internal/storage"
)

const testAPIKeyValue = "test-api-key-secret"

func init() {
	log.Init("console", "error", io.Discard)
}

func setupTestHandler(t *testing.T) (*Handler, storage.ExtendedStorage) {
	t.Helper()
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Create a test user with admin role for RBAC
	passwordHash, _ := auth.HashPassword("test-password")
	testUser := &model.User{
		ID:           "test-user-id",
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: passwordHash,
		IsActive:     true,
		IsAdmin:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := store.CreateUser(context.Background(), testUser); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Get the admin role (created by bootstrap)
	roles, err := store.ListRoles(context.Background(), nil)
	var adminRoleID string
	if err != nil || len(roles) == 0 {
		// Create admin role if it doesn't exist
		adminRole := &model.Role{
			ID:          "admin-role-id",
			Name:        "admin",
			Description: "Administrator role",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := store.CreateRole(context.Background(), adminRole); err != nil {
			t.Fatalf("failed to create admin role: %v", err)
		}
		adminRoleID = adminRole.ID
	} else {
		adminRoleID = roles[0].ID
	}

	// Get all existing permissions (created by migrations) and assign to admin role
	allPerms, err := store.ListPermissions(context.Background(), &model.PermissionFilter{
		Pagination: model.Pagination{Limit: model.MaxPageSize},
	})
	if err != nil {
		t.Fatalf("failed to list permissions: %v", err)
	}
	var permissionIDs []string
	for _, p := range allPerms {
		permissionIDs = append(permissionIDs, p.ID)
	}

	// Assign all permissions to admin role
	if err := store.SetRolePermissions(context.Background(), adminRoleID, permissionIDs); err != nil {
		t.Fatalf("failed to set role permissions: %v", err)
	}

	// Assign the admin role to the test user
	if err := store.AssignRoleToUser(context.Background(), testUser.ID, adminRoleID); err != nil {
		t.Fatalf("failed to assign admin role: %v", err)
	}

	// Create an API key associated with the test user
	apiKey := &model.APIKey{
		ID:     "test-key-id",
		Name:   "test-key",
		Key:    auth.HashToken(testAPIKeyValue),
		UserID: testUser.ID,
	}
	if err := store.CreateAPIKey(context.Background(), apiKey); err != nil {
		t.Fatalf("failed to create test API key: %v", err)
	}

	h := NewHandler(store, nil,
		WithServices(service.NewServices(store, nil, nil)),
	)
	return h, store
}

// authReq adds the test API key Bearer token to a request
func authReq(req *http.Request) *http.Request {
	req.Header.Set("Authorization", "Bearer "+testAPIKeyValue)
	return req
}

func TestDatacenterHandlers(t *testing.T) {
	h, store := setupTestHandler(t)
	defer store.Close()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	t.Run("CreateDatacenter", func(t *testing.T) {
		body := `{"name":"DC1","location":"NYC","description":"Test DC"}`
		req := authReq(httptest.NewRequest("POST", "/api/datacenters", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}
	})

	t.Run("CreateDatacenter_MissingName", func(t *testing.T) {
		body := `{"location":"NYC"}`
		req := authReq(httptest.NewRequest("POST", "/api/datacenters", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateDatacenter_InvalidJSON", func(t *testing.T) {
		req := authReq(httptest.NewRequest("POST", "/api/datacenters", bytes.NewBufferString("invalid")))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateDatacenter_Unauthenticated", func(t *testing.T) {
		body := `{"name":"DC-noauth"}`
		req := httptest.NewRequest("POST", "/api/datacenters", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("ListDatacenters", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/datacenters", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("GetDatacenter_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/datacenters/nonexistent", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	// Create a datacenter for subsequent tests
	var dcID string
	t.Run("CreateAndGet", func(t *testing.T) {
		body := `{"name":"DC2","location":"LA"}`
		req := authReq(httptest.NewRequest("POST", "/api/datacenters", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		dcID = resp["id"].(string)

		req = authReq(httptest.NewRequest("GET", "/api/datacenters/"+dcID, nil))
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("UpdateDatacenter", func(t *testing.T) {
		body := `{"name":"DC2-Updated"}`
		req := authReq(httptest.NewRequest("PUT", "/api/datacenters/"+dcID, bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("UpdateDatacenter_NotFound", func(t *testing.T) {
		body := `{"name":"Updated"}`
		req := authReq(httptest.NewRequest("PUT", "/api/datacenters/nonexistent", bytes.NewBufferString(body)))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("UpdateDatacenter_InvalidJSON", func(t *testing.T) {
		req := authReq(httptest.NewRequest("PUT", "/api/datacenters/"+dcID, bytes.NewBufferString("invalid")))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("GetDatacenterDevices", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/datacenters/"+dcID+"/devices", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("GetDatacenterDevices_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/datacenters/nonexistent/devices", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("DeleteDatacenter", func(t *testing.T) {
		req := authReq(httptest.NewRequest("DELETE", "/api/datacenters/"+dcID, nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected %d, got %d", http.StatusNoContent, w.Code)
		}
	})

	t.Run("DeleteDatacenter_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("DELETE", "/api/datacenters/nonexistent", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}
