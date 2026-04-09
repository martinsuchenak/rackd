package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestCustomFieldHandlers(t *testing.T) {
	h, store := setupTestHandler(t)
	defer store.Close()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	t.Run("ListCustomFields_Empty", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/custom-fields", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var definitions []model.CustomFieldDefinition
		if err := json.Unmarshal(w.Body.Bytes(), &definitions); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if len(definitions) != 0 {
			t.Errorf("expected empty list, got %d definitions", len(definitions))
		}
	})

	t.Run("CreateCustomField_TextType", func(t *testing.T) {
		body := `{
			"name": "Asset Tag",
			"key": "asset_tag",
			"type": "text",
			"required": false,
			"description": "Asset tag identifier"
		}`
		req := authReq(httptest.NewRequest("POST", "/api/custom-fields", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var def model.CustomFieldDefinition
		if err := json.Unmarshal(w.Body.Bytes(), &def); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if def.Name != "Asset Tag" {
			t.Errorf("expected name 'Asset Tag', got '%s'", def.Name)
		}
		if def.Key != "asset_tag" {
			t.Errorf("expected key 'asset_tag', got '%s'", def.Key)
		}
		if def.Type != model.CustomFieldTypeText {
			t.Errorf("expected type 'text', got '%s'", def.Type)
		}
	})

	t.Run("CreateCustomField_NumberType", func(t *testing.T) {
		body := `{
			"name": "Port Count",
			"key": "port_count",
			"type": "number",
			"required": true
		}`
		req := authReq(httptest.NewRequest("POST", "/api/custom-fields", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var def model.CustomFieldDefinition
		if err := json.Unmarshal(w.Body.Bytes(), &def); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if def.Type != model.CustomFieldTypeNumber {
			t.Errorf("expected type 'number', got '%s'", def.Type)
		}
		if !def.Required {
			t.Error("expected required to be true")
		}
	})

	t.Run("CreateCustomField_SelectType", func(t *testing.T) {
		body := `{
			"name": "Environment",
			"key": "environment",
			"type": "select",
			"options": ["production", "staging", "development"],
			"required": false
		}`
		req := authReq(httptest.NewRequest("POST", "/api/custom-fields", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var def model.CustomFieldDefinition
		if err := json.Unmarshal(w.Body.Bytes(), &def); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if def.Type != model.CustomFieldTypeSelect {
			t.Errorf("expected type 'select', got '%s'", def.Type)
		}
		if len(def.Options) != 3 {
			t.Errorf("expected 3 options, got %d", len(def.Options))
		}
	})

	t.Run("CreateCustomField_BoolType", func(t *testing.T) {
		body := `{
			"name": "Monitored",
			"key": "monitored",
			"type": "boolean",
			"required": false
		}`
		req := authReq(httptest.NewRequest("POST", "/api/custom-fields", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var def model.CustomFieldDefinition
		if err := json.Unmarshal(w.Body.Bytes(), &def); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if def.Type != model.CustomFieldTypeBool {
			t.Errorf("expected type 'boolean', got '%s'", def.Type)
		}
	})

	t.Run("CreateCustomField_MissingName", func(t *testing.T) {
		body := `{"key": "test", "type": "text"}`
		req := authReq(httptest.NewRequest("POST", "/api/custom-fields", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("CreateCustomField_MissingKey", func(t *testing.T) {
		body := `{"name": "Test", "type": "text"}`
		req := authReq(httptest.NewRequest("POST", "/api/custom-fields", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("CreateCustomField_InvalidType", func(t *testing.T) {
		body := `{"name": "Test", "key": "test", "type": "invalid"}`
		req := authReq(httptest.NewRequest("POST", "/api/custom-fields", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("CreateCustomField_SelectWithoutOptions", func(t *testing.T) {
		body := `{"name": "Test Select", "key": "test_select", "type": "select"}`
		req := authReq(httptest.NewRequest("POST", "/api/custom-fields", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("CreateCustomField_InvalidKey", func(t *testing.T) {
		body := `{"name": "Test", "key": "invalid-key!", "type": "text"}`
		req := authReq(httptest.NewRequest("POST", "/api/custom-fields", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("CreateCustomField_Unauthenticated", func(t *testing.T) {
		body := `{"name": "Test", "key": "test", "type": "text"}`
		req := httptest.NewRequest("POST", "/api/custom-fields", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	// Create a custom field for subsequent tests
	var fieldID string
	t.Run("CreateAndGet", func(t *testing.T) {
		body := `{"name": "Get Test", "key": "get_test", "type": "text"}`
		req := authReq(httptest.NewRequest("POST", "/api/custom-fields", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		fieldID = resp["id"].(string)

		req = authReq(httptest.NewRequest("GET", "/api/custom-fields/"+fieldID, nil))
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}

		var def model.CustomFieldDefinition
		if err := json.Unmarshal(w.Body.Bytes(), &def); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if def.Name != "Get Test" {
			t.Errorf("expected name 'Get Test', got '%s'", def.Name)
		}
	})

	t.Run("GetCustomField_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/custom-fields/non-existent-id", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("UpdateCustomField", func(t *testing.T) {
		body := `{"name": "Updated Name", "description": "Updated description"}`
		req := authReq(httptest.NewRequest("PUT", "/api/custom-fields/"+fieldID, bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var def model.CustomFieldDefinition
		if err := json.Unmarshal(w.Body.Bytes(), &def); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if def.Name != "Updated Name" {
			t.Errorf("expected name 'Updated Name', got '%s'", def.Name)
		}
		if def.Description != "Updated description" {
			t.Errorf("expected description 'Updated description', got '%s'", def.Description)
		}
	})

	t.Run("UpdateCustomField_NotFound", func(t *testing.T) {
		body := `{"name": "Test"}`
		req := authReq(httptest.NewRequest("PUT", "/api/custom-fields/non-existent-id", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("ListCustomFields", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/custom-fields", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}

		var definitions []model.CustomFieldDefinition
		if err := json.Unmarshal(w.Body.Bytes(), &definitions); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		// Should have at least the fields we created
		if len(definitions) < 1 {
			t.Errorf("expected at least 1 definition, got %d", len(definitions))
		}
	})

	t.Run("ListCustomFields_FilterByType", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/custom-fields?type=text", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}

		var definitions []model.CustomFieldDefinition
		if err := json.Unmarshal(w.Body.Bytes(), &definitions); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		// All returned definitions should be text type
		for _, def := range definitions {
			if def.Type != model.CustomFieldTypeText {
				t.Errorf("expected type 'text', got '%s'", def.Type)
			}
		}
	})

	t.Run("GetCustomFieldTypes", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/custom-fields/types", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}

		var types []map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &types); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if len(types) != 4 {
			t.Errorf("expected 4 types, got %d", len(types))
		}
	})

	t.Run("DeleteCustomField", func(t *testing.T) {
		// Create a field to delete
		body := `{"name": "To Delete", "key": "to_delete", "type": "text"}`
		req := authReq(httptest.NewRequest("POST", "/api/custom-fields", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		deleteID := resp["id"].(string)

		// Delete it
		req = authReq(httptest.NewRequest("DELETE", "/api/custom-fields/"+deleteID, nil))
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected %d, got %d", http.StatusNoContent, w.Code)
		}

		// Verify it's gone
		req = authReq(httptest.NewRequest("GET", "/api/custom-fields/"+deleteID, nil))
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("DeleteCustomField_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("DELETE", "/api/custom-fields/non-existent-id", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("CustomField_ForbiddenWithoutPermission", func(t *testing.T) {
		_, limitedToken := createAPIUserForStore(t, store, "limited-custom-field-user")

		req := authReqWithToken(httptest.NewRequest("GET", "/api/custom-fields", nil), limitedToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}

		req = authReqWithToken(httptest.NewRequest("POST", "/api/custom-fields", bytes.NewBufferString(`{"name":"Limited","key":"limited","type":"text"}`)), limitedToken)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}
