package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/martinsuchenak/rackd/internal/audit"
	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/discovery"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type Handler struct {
	store          storage.ExtendedStorage
	scanner        discovery.Scanner
	credStore      credentials.Storage
	profileStore   storage.ProfileStorage
	scheduledStore storage.ScheduledScanStorage
	sessionManager   *auth.SessionManager
	loginRateLimiter *RateLimiter
	cookieSecure     bool
	sessionTTL       time.Duration
	trustProxy       bool
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

func (h *Handler) SetSessionManager(sm *auth.SessionManager) {
	h.sessionManager = sm
}

func (h *Handler) SetLoginRateLimiter(rl *RateLimiter) {
	h.loginRateLimiter = rl
}

func (h *Handler) SetCookieConfig(secure bool, sessionTTL time.Duration) {
	h.cookieSecure = secure
	h.sessionTTL = sessionTTL
}

func (h *Handler) SetTrustProxy(trustProxy bool) {
	h.trustProxy = trustProxy
}

func (h *Handler) auditContext(r *http.Request) context.Context {
	auditCtx := &audit.Context{
		Source: "api",
	}

	if apiKey := r.Context().Value("api_key"); apiKey != nil {
		if key, ok := apiKey.(*model.APIKey); ok {
			auditCtx.UserID = key.ID
			auditCtx.Username = key.Name
		}
	}

	if session := r.Context().Value(SessionContextKey); session != nil {
		if sess, ok := session.(*auth.Session); ok {
			auditCtx.UserID = sess.UserID
			auditCtx.Username = sess.Username
		}
	}

	auditCtx.IPAddress = r.RemoteAddr

	return audit.WithContext(r.Context(), auditCtx)
}

type HandlerOption func(*handlerConfig)

type handlerConfig struct {
	requireAuth bool
}

func WithAuth() HandlerOption {
	return func(c *handlerConfig) {
		c.requireAuth = true
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, opts ...HandlerOption) {
	cfg := &handlerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	wrap := func(handler http.HandlerFunc) http.HandlerFunc {
		handler = LimitBody(handler)
		if cfg.requireAuth {
			if h.sessionManager != nil {
				return AuthMiddlewareWithSessions(h.store, h.sessionManager, handler)
			}
			return AuthMiddleware(h.store, handler)
		}
		return handler
	}

	wrapAuth := func(handler http.HandlerFunc) http.HandlerFunc {
		handler = LimitBody(handler)
		if h.sessionManager != nil {
			return AuthMiddlewareWithSessions(h.store, h.sessionManager, handler)
		}
		return AuthMiddleware(h.store, handler)
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
	mux.HandleFunc("GET /api/relationships", wrap(h.listAllRelationships))
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

	// API Key routes (always available)
	mux.HandleFunc("GET /api/keys", wrap(h.listAPIKeys))
	mux.HandleFunc("POST /api/keys", wrap(h.createAPIKey))
	mux.HandleFunc("GET /api/keys/{id}", wrap(h.getAPIKey))
	mux.HandleFunc("DELETE /api/keys/{id}", wrap(h.deleteAPIKey))

	// Bulk device operations
	mux.HandleFunc("POST /api/devices/bulk", wrap(h.bulkCreateDevices))
	mux.HandleFunc("PUT /api/devices/bulk", wrap(h.bulkUpdateDevices))
	mux.HandleFunc("DELETE /api/devices/bulk", wrap(h.bulkDeleteDevices))
	mux.HandleFunc("POST /api/devices/bulk/tags", wrap(h.bulkAddTags))
	mux.HandleFunc("DELETE /api/devices/bulk/tags", wrap(h.bulkRemoveTags))

	// Bulk network operations
	mux.HandleFunc("POST /api/networks/bulk", wrap(h.bulkCreateNetworks))
	mux.HandleFunc("DELETE /api/networks/bulk", wrap(h.bulkDeleteNetworks))

	// Audit log routes
	mux.HandleFunc("GET /api/audit", wrap(h.listAuditLogs))
	mux.HandleFunc("GET /api/audit/export", wrap(h.exportAuditLogs))
	mux.HandleFunc("GET /api/audit/{id}", wrap(h.getAuditLog))

	// Auth routes (no auth required for login)
	loginHandler := LimitBody(h.login)
	if h.loginRateLimiter != nil {
		loginHandler = LoginRateLimitMiddleware(h.loginRateLimiter, h.trustProxy, loginHandler)
	}
	mux.HandleFunc("POST /api/auth/login", loginHandler)
	mux.HandleFunc("POST /api/auth/logout", wrapAuth(h.logout))
	mux.HandleFunc("GET /api/auth/me", wrapAuth(h.getCurrentUser))

	// User routes (require auth)
	mux.HandleFunc("GET /api/users", wrapAuth(h.listUsers))
	mux.HandleFunc("POST /api/users", wrapAuth(h.createUser))
	mux.HandleFunc("GET /api/users/{id}", wrapAuth(h.getUser))
	mux.HandleFunc("PUT /api/users/{id}", wrapAuth(h.updateUser))
	mux.HandleFunc("DELETE /api/users/{id}", wrapAuth(h.deleteUser))
	mux.HandleFunc("POST /api/users/{id}/password", wrapAuth(h.changePassword))

	// Health check routes (no auth required)
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /readyz", h.readyz)

	// Metrics route (requires auth)
	mux.HandleFunc("GET /metrics", wrap(h.metricsHandler))
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
	config := NewUIConfigBuilder()
	config.AddNavItem(NavItem{Label: "Users", Path: "/users", Icon: "user", Order: 15})
	h.writeJSON(w, http.StatusOK, config.Build())
}
