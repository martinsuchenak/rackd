package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestWebhookHandlers(t *testing.T) {
	h, store := setupTestHandler(t)
	defer store.Close()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	t.Run("ListWebhooks_Empty", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/webhooks", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var webhooks []model.Webhook
		if err := json.Unmarshal(w.Body.Bytes(), &webhooks); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if len(webhooks) != 0 {
			t.Errorf("expected empty list, got %d webhooks", len(webhooks))
		}
	})

	t.Run("CreateWebhook", func(t *testing.T) {
		body := `{
			"name": "test-webhook",
			"url": "https://example.com/webhook",
			"secret": "test-secret",
			"events": ["device.created", "device.updated"],
			"active": true,
			"description": "Test webhook"
		}`
		req := authReq(httptest.NewRequest("POST", "/api/webhooks", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var webhook model.Webhook
		if err := json.Unmarshal(w.Body.Bytes(), &webhook); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if webhook.Name != "test-webhook" {
			t.Errorf("expected name 'test-webhook', got '%s'", webhook.Name)
		}
		if webhook.URL != "https://example.com/webhook" {
			t.Errorf("expected URL 'https://example.com/webhook', got '%s'", webhook.URL)
		}
		if len(webhook.Events) != 2 {
			t.Errorf("expected 2 events, got %d", len(webhook.Events))
		}
	})

	t.Run("CreateWebhook_MissingName", func(t *testing.T) {
		body := `{"url": "https://example.com/webhook", "events": ["device.created"]}`
		req := authReq(httptest.NewRequest("POST", "/api/webhooks", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateWebhook_MissingURL", func(t *testing.T) {
		body := `{"name": "test", "events": ["device.created"]}`
		req := authReq(httptest.NewRequest("POST", "/api/webhooks", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateWebhook_MissingEvents", func(t *testing.T) {
		body := `{"name": "test", "url": "https://example.com/webhook"}`
		req := authReq(httptest.NewRequest("POST", "/api/webhooks", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateWebhook_InvalidJSON", func(t *testing.T) {
		req := authReq(httptest.NewRequest("POST", "/api/webhooks", bytes.NewBufferString("invalid")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateWebhook_Unauthenticated", func(t *testing.T) {
		body := `{"name": "test", "url": "https://example.com/webhook", "events": ["device.created"]}`
		req := httptest.NewRequest("POST", "/api/webhooks", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	// Create a webhook for subsequent tests
	var webhookID string
	t.Run("CreateAndGet", func(t *testing.T) {
		body := `{"name": "test-webhook-2", "url": "https://example.com/webhook2", "events": ["device.created"]}`
		req := authReq(httptest.NewRequest("POST", "/api/webhooks", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		webhookID = resp["id"].(string)

		req = authReq(httptest.NewRequest("GET", "/api/webhooks/"+webhookID, nil))
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}

		var webhook model.Webhook
		json.Unmarshal(w.Body.Bytes(), &webhook)
		if webhook.Name != "test-webhook-2" {
			t.Errorf("expected name 'test-webhook-2', got '%s'", webhook.Name)
		}
	})

	t.Run("GetWebhook_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/webhooks/nonexistent", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("UpdateWebhook", func(t *testing.T) {
		body := `{"name": "updated-webhook", "active": false}`
		req := authReq(httptest.NewRequest("PUT", "/api/webhooks/"+webhookID, bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var webhook model.Webhook
		json.Unmarshal(w.Body.Bytes(), &webhook)
		if webhook.Name != "updated-webhook" {
			t.Errorf("expected name 'updated-webhook', got '%s'", webhook.Name)
		}
		if webhook.Active != false {
			t.Errorf("expected active false, got %v", webhook.Active)
		}
	})

	t.Run("UpdateWebhook_NotFound", func(t *testing.T) {
		body := `{"name": "Updated"}`
		req := authReq(httptest.NewRequest("PUT", "/api/webhooks/nonexistent", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("ListWebhooks_WithActiveFilter", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/webhooks?active=false", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}

		var webhooks []model.Webhook
		json.Unmarshal(w.Body.Bytes(), &webhooks)
		for _, w := range webhooks {
			if w.Active != false {
				t.Errorf("expected only inactive webhooks, got active webhook")
			}
		}
	})

	t.Run("PingWebhook", func(t *testing.T) {
		req := authReq(httptest.NewRequest("POST", "/api/webhooks/"+webhookID+"/ping", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// Ping may fail since the webhook URL doesn't exist, but it should attempt
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("expected %d or %d, got %d", http.StatusOK, http.StatusInternalServerError, w.Code)
		}
	})

	t.Run("PingWebhook_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("POST", "/api/webhooks/nonexistent/ping", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("ListWebhookDeliveries", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/webhooks/"+webhookID+"/deliveries", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("ListWebhookDeliveries_WithFilters", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/webhooks/"+webhookID+"/deliveries?status=success&event_type=device.created", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("GetWebhookDelivery_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/webhooks/"+webhookID+"/deliveries/nonexistent", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("GetEventTypes", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/webhooks/events", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var eventTypes []map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &eventTypes); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if len(eventTypes) != len(model.AllEventTypes) {
			t.Errorf("expected %d event types, got %d", len(model.AllEventTypes), len(eventTypes))
		}
	})

	t.Run("DeleteWebhook", func(t *testing.T) {
		req := authReq(httptest.NewRequest("DELETE", "/api/webhooks/"+webhookID, nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected %d, got %d", http.StatusNoContent, w.Code)
		}

		// Verify deletion
		req = authReq(httptest.NewRequest("GET", "/api/webhooks/"+webhookID, nil))
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d after deletion, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("DeleteWebhook_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("DELETE", "/api/webhooks/nonexistent", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("Webhook_ForbiddenWithoutPermission", func(t *testing.T) {
		_, limitedToken := createAPIUserForStore(t, store, "limited-webhook-user")

		req := authReqWithToken(httptest.NewRequest("GET", "/api/webhooks", nil), limitedToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}

		req = authReqWithToken(httptest.NewRequest("POST", "/api/webhooks", bytes.NewBufferString(`{"name":"limited","url":"https://example.com/hook","events":["device.created"]}`)), limitedToken)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestGetEventLabel(t *testing.T) {
	tests := []struct {
		eventType model.EventType
		expected  string
	}{
		{model.EventTypeDeviceCreated, "Device Created"},
		{model.EventTypeDeviceUpdated, "Device Updated"},
		{model.EventTypeDeviceDeleted, "Device Deleted"},
		{model.EventTypeDevicePromoted, "Device Promoted"},
		{model.EventTypeNetworkCreated, "Network Created"},
		{model.EventTypeNetworkUpdated, "Network Updated"},
		{model.EventTypeNetworkDeleted, "Network Deleted"},
		{model.EventTypeDiscoveryStarted, "Discovery Started"},
		{model.EventTypeDiscoveryCompleted, "Discovery Completed"},
		{model.EventTypeDeviceDiscovered, "Device Discovered"},
		{model.EventTypeConflictDetected, "Conflict Detected"},
		{model.EventTypeConflictResolved, "Conflict Resolved"},
		{model.EventTypePoolUtilization, "Pool Utilization High"},
		{model.EventType("unknown.event"), "unknown.event"},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			label := getEventLabel(tt.eventType)
			if label != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, label)
			}
		})
	}
}
