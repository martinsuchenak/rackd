package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestNATHandlers(t *testing.T) {
	h, store := setupTestHandler(t)
	defer store.Close()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	t.Run("ListNATMappings_Empty", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/nat", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var mappings []model.NATMapping
		if err := json.Unmarshal(w.Body.Bytes(), &mappings); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(mappings) != 0 {
			t.Errorf("expected empty list, got %d mappings", len(mappings))
		}
	})

	t.Run("CreateNATMapping", func(t *testing.T) {
		body := `{
			"name": "Web Server NAT",
			"external_ip": "203.0.113.10",
			"external_port": 443,
			"internal_ip": "192.168.1.10",
			"internal_port": 443,
			"protocol": "tcp",
			"description": "HTTPS to internal web server",
			"enabled": true
		}`
		req := authReq(httptest.NewRequest("POST", "/api/nat", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
				t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var mapping model.NATMapping
		if err := json.Unmarshal(w.Body.Bytes(), &mapping); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if mapping.Name != "Web Server NAT" {
			t.Errorf("expected name 'Web Server NAT', got '%s'", mapping.Name)
		}
		if mapping.ExternalIP != "203.0.113.10" {
			t.Errorf("expected external_ip '203.0.113.10', got '%s'", mapping.ExternalIP)
		}
		if mapping.ExternalPort != 443 {
			t.Errorf("expected external_port 443, got %d", mapping.ExternalPort)
		}
		if mapping.InternalIP != "192.168.1.10" {
			t.Errorf("expected internal_ip '192.168.1.10', got '%s'", mapping.InternalIP)
		}
		if mapping.InternalPort != 443 {
			t.Errorf("expected internal_port 443, got %d", mapping.InternalPort)
		}
		if mapping.Protocol != model.NATProtocolTCP {
			t.Errorf("expected protocol 'tcp', got '%s'", mapping.Protocol)
		}
		if mapping.Enabled != true {
			t.Errorf("expected enabled true, got %v", mapping.Enabled)
		}
	})

	t.Run("CreateNATMapping_MissingName", func(t *testing.T) {
		body := `{"external_ip": "203.0.113.10", "external_port": 443, "internal_ip": "192.168.1.10", "internal_port": 443}`
		req := authReq(httptest.NewRequest("POST", "/api/nat", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateNATMapping_MissingExternalIP", func(t *testing.T) {
		body := `{"name": "test", "external_port": 443, "internal_ip": "192.168.1.10", "internal_port": 443}`
		req := authReq(httptest.NewRequest("POST", "/api/nat", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateNATMapping_MissingInternalIP", func(t *testing.T) {
		body := `{"name": "test", "external_ip": "203.0.113.10", "external_port": 443, "internal_port": 443}`
		req := authReq(httptest.NewRequest("POST", "/api/nat", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateNATMapping_InvalidPort", func(t *testing.T) {
		body := `{"name": "test", "external_ip": "203.0.113.10", "external_port": 99999, "internal_ip": "192.168.1.10", "internal_port": 443}`
		req := authReq(httptest.NewRequest("POST", "/api/nat", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateNATMapping_InvalidProtocol", func(t *testing.T) {
		body := `{"name": "test", "external_ip": "203.0.113.10", "external_port": 443, "internal_ip": "192.168.1.10", "internal_port": 443, "protocol": "invalid"} `
		req := authReq(httptest.NewRequest("POST", "/api/nat", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateNATMapping_DefaultProtocol", func(t *testing.T) {
		body := `{"name": "test", "external_ip": "203.0.113.10", "external_port": 443, "internal_ip": "192.168.1.10", "internal_port": 443}`
		req := authReq(httptest.NewRequest("POST", "/api/nat", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var mapping model.NATMapping
		if err := json.Unmarshal(w.Body.Bytes(), &mapping); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if mapping.Protocol != model.NATProtocolTCP {
			t.Errorf("expected default protocol 'tcp', got '%s'", mapping.Protocol)
		}
	})

	t.Run("GetNATMapping", func(t *testing.T) {
		// First create a mapping
		body := `{"name": "Get Test", "external_ip": "203.0.113.20", "external_port": 80, "internal_ip": "192.168.1.20", "internal_port": 80, "protocol": "tcp"}`
		req := authReq(httptest.NewRequest("POST", "/api/nat", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("failed to create mapping: %d", w.Code)
		}

		var created model.NATMapping
		if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
			t.Fatalf("failed to unmarshal created response: %v", err)
		}

		// Now get it
		req = authReq(httptest.NewRequest("GET", "/api/nat/"+created.ID, nil))
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}

		var retrieved model.NATMapping
		if err := json.Unmarshal(w.Body.Bytes(), &retrieved); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if retrieved.ID != created.ID {
			t.Errorf("expected ID %s, got %s", created.ID, retrieved.ID)
		}
		if retrieved.Name != "Get Test" {
			t.Errorf("expected name 'Get Test', got '%s'", retrieved.Name)
		}
	})

	t.Run("GetNATMapping_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/nat/non-existent-id", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("UpdateNATMapping", func(t *testing.T) {
		// First create a mapping
		body := `{"name": "Update Test", "external_ip": "203.0.113.30", "external_port": 8080, "internal_ip": "192.168.1.30", "internal_port": 80, "protocol": "tcp"}`
		req := authReq(httptest.NewRequest("POST", "/api/nat", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("failed to create mapping: %d", w.Code)
		}

		var created model.NATMapping
		if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
			t.Fatalf("failed to unmarshal created response: %v", err)
		}

		// Update it
		updateBody := `{"name": "Updated Name", "external_port": 8443, "enabled": false}`
		req = authReq(httptest.NewRequest("PUT", "/api/nat/"+created.ID, bytes.NewBufferString(updateBody)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var updated model.NATMapping
		if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
			t.Fatalf("failed to unmarshal updated response: %v", err)
		}

		if updated.Name != "Updated Name" {
			t.Errorf("expected name 'Updated Name' got '%s'", updated.Name)
		}
		if updated.ExternalPort != 8443 {
			t.Errorf("expected external_port 8443, got %d", updated.ExternalPort)
		}
		if updated.Enabled != false {
			t.Errorf("expected enabled false, got %v", updated.Enabled)
		}
		// Original values should remain
		if updated.ExternalIP != "203.0.113.30" {
			t.Errorf("external_ip should remain unchanged")
		}
		if updated.InternalIP != "192.168.1.30" {
			t.Errorf("internal_ip should remain unchanged")
		}
	})

	t.Run("UpdateNATMapping_NotFound", func(t *testing.T) {
		body := `{"name": "test"}`
		req := authReq(httptest.NewRequest("PUT", "/api/nat/non-existent-id", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("DeleteNATMapping", func(t *testing.T) {
		// First create a mapping
		body := `{"name": "Delete Test", "external_ip": "203.0.113.40", "external_port": 9000, "internal_ip": "192.168.1.40", "internal_port": 9000, "protocol": "tcp"}`
		req := authReq(httptest.NewRequest("POST", "/api/nat", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("failed to create mapping: %d", w.Code)
		}

		var created model.NATMapping
		if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
			t.Fatalf("failed to unmarshal created response: %v", err)
		}

		// Delete it
		req = authReq(httptest.NewRequest("DELETE", "/api/nat/"+created.ID, nil))
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected %d, got %d", http.StatusNoContent, w.Code)
		}

		// Verify deleted
		req = authReq(httptest.NewRequest("GET", "/api/nat/"+created.ID, nil))
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d after delete, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("DeleteNATMapping_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("DELETE", "/api/nat/non-existent-id", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("ListNATMappings_WithFilters", func(t *testing.T) {
		// Use unique IP addresses to avoid conflicts with other tests
		uniqueIPs := []string{"223.0.113.10", "223.0.113.20", "223.0.113.30"}

		// Create multiple mappings with unique IPs
		mappings := []string{
			`{"name": "Filter TCP Enabled", "external_ip": "` + uniqueIPs[0] + `", "external_port": 443, "internal_ip": "192.168.1.10", "internal_port": 443, "protocol": "tcp", "enabled": true}`,
			`{"name": "Filter UDP Disabled", "external_ip": "` + uniqueIPs[1] + `", "external_port": 53, "internal_ip": "192.168.1.20", "internal_port": 53, "protocol": "udp", "enabled": false}`,
			`{"name": "Filter TCP Disabled", "external_ip": "` + uniqueIPs[2] + `", "external_port": 80, "internal_ip": "192.168.1.30", "internal_port": 80, "protocol": "tcp", "enabled": false}`,
		}

		for _, body := range mappings {
			createReq := authReq(httptest.NewRequest("POST", "/api/nat", bytes.NewBufferString(body)))
			createReq.Header.Set("Content-Type", "application/json")
			createW := httptest.NewRecorder()
			mux.ServeHTTP(createW, createReq)
			if createW.Code != http.StatusCreated {
				t.Fatalf("failed to create mapping: %d", createW.Code)
			}
		}

		// Filter by protocol - should get 2 TCP mappings (our unique ones)
		req := authReq(httptest.NewRequest("GET", "/api/nat?protocol=tcp", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}

		var tcpMappings []model.NATMapping
		if err := json.Unmarshal(w.Body.Bytes(), &tcpMappings); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// Verify our TCP mappings are present
		tcpNames := make(map[string]bool)
		for _, m := range tcpMappings {
			if m.Name == "Filter TCP Enabled" || m.Name == "Filter TCP Disabled" {
				tcpNames[m.Name] = true
			}
		}
		if !tcpNames["Filter TCP Enabled"] || !tcpNames["Filter TCP Disabled"] {
			t.Errorf("expected both TCP filter mappings, got: %v", tcpMappings)
		}

		// Filter by external IP - should get exactly 1
		req = authReq(httptest.NewRequest("GET", "/api/nat?external_ip="+uniqueIPs[0], nil))
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}

		var filteredMappings []model.NATMapping
		if err := json.Unmarshal(w.Body.Bytes(), &filteredMappings); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if len(filteredMappings) != 1 {
			t.Errorf("expected 1 mapping with external_ip filter, got %d", len(filteredMappings))
		} else if filteredMappings[0].Name != "Filter TCP Enabled" {
			t.Errorf("expected 'Filter TCP Enabled', got '%s'", filteredMappings[0].Name)
		}
	})

	t.Run("CreateNATMapping_WithTags", func(t *testing.T) {
		body := `{"name": "Tagged NAT", "external_ip": "203.0.113.50", "external_port": 443, "internal_ip": "192.168.1.50", "internal_port": 443, "protocol": "tcp", "tags": ["production", "web"]}`
		req := authReq(httptest.NewRequest("POST", "/api/nat", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var mapping model.NATMapping
		if err := json.Unmarshal(w.Body.Bytes(), &mapping); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(mapping.Tags) != 2 {
			t.Errorf("expected 1 tag, got %d", len(mapping.Tags))
		}
	})

	t.Run("CreateNATMapping_AllProtocols", func(t *testing.T) {
		protocols := []model.NATProtocol{model.NATProtocolTCP, model.NATProtocolUDP, model.NATProtocolAny}

		for i, protocol := range protocols {
				body := `{"name": "Protocol Test ` + string(rune('A'+i)) + `", "external_ip": "203.0.113.1` + string(rune('0'+i+1)) + `", "external_port": 443, "internal_ip": "192.168.1.1` + string(rune('0'+i+1)) + `", "internal_port": 443, "protocol": "` + string(protocol) + `"}`
			req := authReq(httptest.NewRequest("POST", "/api/nat", bytes.NewBufferString(body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				t.Errorf("expected %d for protocol %s, got %d: %s", http.StatusCreated, protocol, w.Code, w.Body.String())
			}

			var mapping model.NATMapping
			if err := json.Unmarshal(w.Body.Bytes(), &mapping); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if mapping.Protocol != protocol {
				t.Errorf("expected protocol %s, got %s", protocol, mapping.Protocol)
			}
		}
	})
}
