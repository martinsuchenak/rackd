package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBulkHandlers(t *testing.T) {
	env := setupExtendedTestHandler(t, false, false, false, false)
	defer env.close()

	t.Run("BulkDeviceAndNetworkOperations", func(t *testing.T) {
		createDevicesReq := authReq(httptest.NewRequest("POST", "/api/devices/bulk", bytes.NewBufferString(`[{"name":"bulk-device-1"},{"name":"bulk-device-2"}]`)))
		createDevicesReq.Header.Set("Content-Type", "application/json")
		w := performRequest(env.mux, createDevicesReq)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		updateDevicesReq := authReq(httptest.NewRequest("PUT", "/api/devices/bulk", bytes.NewBufferString(`[{"id":"missing-device","name":"updated"}]`)))
		updateDevicesReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, updateDevicesReq)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		deleteDevicesReq := authReq(httptest.NewRequest("DELETE", "/api/devices/bulk", bytes.NewBufferString(`{"ids":["missing-device"]}`)))
		deleteDevicesReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, deleteDevicesReq)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		createNetworksReq := authReq(httptest.NewRequest("POST", "/api/networks/bulk", bytes.NewBufferString(`[{"name":"bulk-net-1","subnet":"10.40.0.0/24"}]`)))
		createNetworksReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, createNetworksReq)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		deleteNetworksReq := authReq(httptest.NewRequest("DELETE", "/api/networks/bulk", bytes.NewBufferString(`{"ids":["missing-network"]}`)))
		deleteNetworksReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, deleteNetworksReq)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("BulkOperations_InvalidJSONAndLimits", func(t *testing.T) {
		w := performRequest(env.mux, authReq(httptest.NewRequest("POST", "/api/devices/bulk", bytes.NewBufferString("{"))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}

		tooManyDevices := "[" + strings.Repeat(`{"name":"x"},`, 100) + `{"name":"x"}]`
		req := authReq(httptest.NewRequest("POST", "/api/devices/bulk", bytes.NewBufferString(tooManyDevices)))
		req.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}

		tooManyIDs := `{"ids":[` + strings.Repeat(`"x",`, 100) + `"x"]}`
		req = authReq(httptest.NewRequest("DELETE", "/api/networks/bulk", bytes.NewBufferString(tooManyIDs)))
		req.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("BulkOperations_ForbiddenWithoutPermission", func(t *testing.T) {
		_, limitedToken := env.createAPIUser(t, "limited-bulk-user")

		req := authReqWithToken(httptest.NewRequest("POST", "/api/devices/bulk", bytes.NewBufferString(`[{"name":"limited-device"}]`)), limitedToken)
		req.Header.Set("Content-Type", "application/json")
		w := performRequest(env.mux, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}

		req = authReqWithToken(httptest.NewRequest("DELETE", "/api/networks/bulk", bytes.NewBufferString(`{"ids":["net-1"]}`)), limitedToken)
		req.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}
