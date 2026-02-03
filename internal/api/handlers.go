package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/discovery"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type Handler struct {
	store          storage.ExtendedStorage
	scanner        discovery.Scanner
	credStore      credentials.Storage
	profileStore   storage.ProfileStorage
	scheduledStore storage.ScheduledScanStorage
}

func NewHandler(s storage.ExtendedStorage, scanner discovery.Scanner) *Handler {
	return &Handler{store: s, scanner: scanner}
}

func (h *Handler) SetCredentialsStorage(cs credentials.Storage) {
	h.credStore = cs
}

func (h *Handler) SetProfileStorage(ps storage.ProfileStorage) {
	h.profileStore = ps
}

func (h *Handler) SetScheduledScanStorage(ss storage.ScheduledScanStorage) {
	h.scheduledStore = ss
}

type HandlerOption func(*handlerConfig)

type handlerConfig struct {
	authToken string
}

func WithAuth(token string) HandlerOption {
	return func(c *handlerConfig) {
		c.authToken = token
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, opts ...HandlerOption) {
	cfg := &handlerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	wrap := func(handler http.HandlerFunc) http.HandlerFunc {
		handler = LimitBody(handler)
		if cfg.authToken != "" {
			return AuthMiddleware(cfg.authToken, handler)
		}
		return handler
	}

	// Datacenter routes
	mux.HandleFunc("GET /api/datacenters", wrap(h.listDatacenters))
	mux.HandleFunc("POST /api/datacenters", wrap(h.createDatacenter))
	mux.HandleFunc("GET /api/datacenters/{id}", wrap(h.getDatacenter))
	mux.HandleFunc("PUT /api/datacenters/{id}", wrap(h.updateDatacenter))
	mux.HandleFunc("DELETE /api/datacenters/{id}", wrap(h.deleteDatacenter))
	mux.HandleFunc("GET /api/datacenters/{id}/devices", wrap(h.getDatacenterDevices))

	// Network routes
	mux.HandleFunc("GET /api/networks", wrap(h.listNetworks))
	mux.HandleFunc("POST /api/networks", wrap(h.createNetwork))
	mux.HandleFunc("GET /api/networks/{id}", wrap(h.getNetwork))
	mux.HandleFunc("PUT /api/networks/{id}", wrap(h.updateNetwork))
	mux.HandleFunc("DELETE /api/networks/{id}", wrap(h.deleteNetwork))
	mux.HandleFunc("GET /api/networks/{id}/devices", wrap(h.getNetworkDevices))
	mux.HandleFunc("GET /api/networks/{id}/utilization", wrap(h.getNetworkUtilization))
	mux.HandleFunc("GET /api/networks/{id}/pools", wrap(h.listNetworkPools))
	mux.HandleFunc("POST /api/networks/{id}/pools", wrap(h.createNetworkPool))

	// Pool routes
	mux.HandleFunc("GET /api/pools/{id}", wrap(h.getNetworkPool))
	mux.HandleFunc("PUT /api/pools/{id}", wrap(h.updateNetworkPool))
	mux.HandleFunc("DELETE /api/pools/{id}", wrap(h.deleteNetworkPool))
	mux.HandleFunc("GET /api/pools/{id}/next-ip", wrap(h.getNextIP))
	mux.HandleFunc("GET /api/pools/{id}/heatmap", wrap(h.getPoolHeatmap))

	// Device routes
	mux.HandleFunc("GET /api/devices", wrap(h.listDevices))
	mux.HandleFunc("POST /api/devices", wrap(h.createDevice))
	mux.HandleFunc("GET /api/devices/{id}", wrap(h.getDevice))
	mux.HandleFunc("PUT /api/devices/{id}", wrap(h.updateDevice))
	mux.HandleFunc("DELETE /api/devices/{id}", wrap(h.deleteDevice))

	// Search routes
	mux.HandleFunc("GET /api/search", wrap(h.search))

	// Relationship routes
	mux.HandleFunc("POST /api/devices/{id}/relationships", wrap(h.addRelationship))
	mux.HandleFunc("GET /api/devices/{id}/relationships", wrap(h.getRelationships))
	mux.HandleFunc("GET /api/devices/{id}/related", wrap(h.getRelatedDevices))
	mux.HandleFunc("PATCH /api/devices/{id}/relationships/{child_id}/{type}", wrap(h.updateRelationshipNotes))
	mux.HandleFunc("DELETE /api/devices/{id}/relationships/{child_id}/{type}", wrap(h.removeRelationship))

	// Discovery routes
	mux.HandleFunc("POST /api/discovery/networks/{id}/scan", wrap(h.startScan))
	mux.HandleFunc("GET /api/discovery/scans", wrap(h.listScans))
	mux.HandleFunc("GET /api/discovery/scans/{id}", wrap(h.getScan))
	mux.HandleFunc("POST /api/discovery/scans/{id}/cancel", wrap(h.cancelScan))
	mux.HandleFunc("DELETE /api/discovery/scans/{id}", wrap(h.deleteDiscoveryScan))
	mux.HandleFunc("GET /api/discovery/devices", wrap(h.listDiscoveredDevices))
	mux.HandleFunc("DELETE /api/discovery/devices", wrap(h.deleteDiscoveredDevicesByNetwork))
	mux.HandleFunc("DELETE /api/discovery/devices/{id}", wrap(h.deleteDiscoveredDevice))
	mux.HandleFunc("POST /api/discovery/devices/{id}/promote", wrap(h.promoteDevice))
	mux.HandleFunc("GET /api/discovery/rules", wrap(h.listDiscoveryRules))
	mux.HandleFunc("POST /api/discovery/rules", wrap(h.createDiscoveryRule))
	mux.HandleFunc("GET /api/discovery/rules/{id}", wrap(h.getDiscoveryRule))
	mux.HandleFunc("PUT /api/discovery/rules/{id}", wrap(h.updateDiscoveryRule))
	mux.HandleFunc("DELETE /api/discovery/rules/{id}", wrap(h.deleteDiscoveryRule))

	// Credentials routes (if storage is configured)
	if h.credStore != nil {
		mux.HandleFunc("GET /api/credentials", wrap(h.listCredentials))
		mux.HandleFunc("POST /api/credentials", wrap(h.createCredential))
		mux.HandleFunc("GET /api/credentials/{id}", wrap(h.getCredential))
		mux.HandleFunc("PUT /api/credentials/{id}", wrap(h.updateCredential))
		mux.HandleFunc("DELETE /api/credentials/{id}", wrap(h.deleteCredential))
	}

	// Scan Profiles routes (if storage is configured)
	if h.profileStore != nil {
		mux.HandleFunc("GET /api/scan-profiles", wrap(h.listProfiles))
		mux.HandleFunc("POST /api/scan-profiles", wrap(h.createProfile))
		mux.HandleFunc("GET /api/scan-profiles/{id}", wrap(h.getProfile))
		mux.HandleFunc("PUT /api/scan-profiles/{id}", wrap(h.updateProfile))
		mux.HandleFunc("DELETE /api/scan-profiles/{id}", wrap(h.deleteProfile))
	}

	// Scheduled Scans routes (if storage is configured)
	if h.scheduledStore != nil {
		mux.HandleFunc("GET /api/scheduled-scans", wrap(h.listScheduledScans))
		mux.HandleFunc("POST /api/scheduled-scans", wrap(h.createScheduledScan))
		mux.HandleFunc("GET /api/scheduled-scans/{id}", wrap(h.getScheduledScan))
		mux.HandleFunc("PUT /api/scheduled-scans/{id}", wrap(h.updateScheduledScan))
		mux.HandleFunc("DELETE /api/scheduled-scans/{id}", wrap(h.deleteScheduledScan))
	}

	// Health check routes (no auth required)
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /readyz", h.readyz)

	// Metrics route (no auth required)
	mux.HandleFunc("GET /metrics", h.metricsHandler)
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
		"code":  code,
	})
}

func (h *Handler) internalError(w http.ResponseWriter, err error) {
	log.Error("Internal server error", "error", err)
	h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal Server Error")
}

func (h *Handler) writeValidationErrors(w http.ResponseWriter, errs ValidationErrors) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]any{
		"error":   errs.Error(),
		"code":    "VALIDATION_ERROR",
		"details": errs,
	})
}

func parseArrayParam(r *http.Request, name string) []string {
	values := r.URL.Query()[name]
	if len(values) == 0 {
		return nil
	}
	return values
}

func parseIntParam(r *http.Request, name string, defaultValue int) int {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultValue
	}
	result, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return result
}

func (h *Handler) getConfig(w http.ResponseWriter, r *http.Request) {
	config := NewUIConfigBuilder().Build()
	h.writeJSON(w, http.StatusOK, config)
}
