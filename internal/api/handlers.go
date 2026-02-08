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
	store            storage.ExtendedStorage
	scanner          discovery.Scanner
	credStore        credentials.Storage
	profileStore     storage.ProfileStorage
	scheduledStore   storage.ScheduledScanStorage
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

	// wrapPerm requires authentication AND checks RBAC permission.
	// When auth is not configured (cfg.requireAuth == false and no session manager),
	// it falls back to wrap behavior for backward compatibility.
	wrapPerm := func(handler http.HandlerFunc, resource, action string) http.HandlerFunc {
		handler = LimitBody(handler)
		if cfg.requireAuth || h.sessionManager != nil {
			handler = RequirePermission(h.store, resource, action)(handler)
			if h.sessionManager != nil {
				return AuthMiddlewareWithSessions(h.store, h.sessionManager, handler)
			}
			return AuthMiddleware(h.store, handler)
		}
		return handler
	}

	// Datacenter routes
	mux.HandleFunc("GET /api/datacenters", wrapPerm(h.listDatacenters, "datacenters", "list"))
	mux.HandleFunc("POST /api/datacenters", wrapPerm(h.createDatacenter, "datacenters", "create"))
	mux.HandleFunc("GET /api/datacenters/{id}", wrapPerm(h.getDatacenter, "datacenters", "read"))
	mux.HandleFunc("PUT /api/datacenters/{id}", wrapPerm(h.updateDatacenter, "datacenters", "update"))
	mux.HandleFunc("DELETE /api/datacenters/{id}", wrapPerm(h.deleteDatacenter, "datacenters", "delete"))
	mux.HandleFunc("GET /api/datacenters/{id}/devices", wrapPerm(h.getDatacenterDevices, "datacenters", "read"))

	// Network routes
	mux.HandleFunc("GET /api/networks", wrapPerm(h.listNetworks, "networks", "list"))
	mux.HandleFunc("POST /api/networks", wrapPerm(h.createNetwork, "networks", "create"))
	mux.HandleFunc("GET /api/networks/{id}", wrapPerm(h.getNetwork, "networks", "read"))
	mux.HandleFunc("PUT /api/networks/{id}", wrapPerm(h.updateNetwork, "networks", "update"))
	mux.HandleFunc("DELETE /api/networks/{id}", wrapPerm(h.deleteNetwork, "networks", "delete"))
	mux.HandleFunc("GET /api/networks/{id}/devices", wrapPerm(h.getNetworkDevices, "networks", "read"))
	mux.HandleFunc("GET /api/networks/{id}/utilization", wrapPerm(h.getNetworkUtilization, "networks", "read"))
	mux.HandleFunc("GET /api/networks/{id}/pools", wrapPerm(h.listNetworkPools, "pools", "list"))
	mux.HandleFunc("POST /api/networks/{id}/pools", wrapPerm(h.createNetworkPool, "pools", "create"))

	// Pool routes
	mux.HandleFunc("GET /api/pools/{id}", wrapPerm(h.getNetworkPool, "pools", "read"))
	mux.HandleFunc("PUT /api/pools/{id}", wrapPerm(h.updateNetworkPool, "pools", "update"))
	mux.HandleFunc("DELETE /api/pools/{id}", wrapPerm(h.deleteNetworkPool, "pools", "delete"))
	mux.HandleFunc("GET /api/pools/{id}/next-ip", wrapPerm(h.getNextIP, "pools", "read"))
	mux.HandleFunc("GET /api/pools/{id}/heatmap", wrapPerm(h.getPoolHeatmap, "pools", "read"))

	// Device routes
	mux.HandleFunc("GET /api/devices", wrapPerm(h.listDevices, "devices", "list"))
	mux.HandleFunc("POST /api/devices", wrapPerm(h.createDevice, "devices", "create"))
	mux.HandleFunc("GET /api/devices/{id}", wrapPerm(h.getDevice, "devices", "read"))
	mux.HandleFunc("PUT /api/devices/{id}", wrapPerm(h.updateDevice, "devices", "update"))
	mux.HandleFunc("DELETE /api/devices/{id}", wrapPerm(h.deleteDevice, "devices", "delete"))

	// Search routes
	mux.HandleFunc("GET /api/search", wrapPerm(h.search, "search", "read"))

	// Relationship routes
	mux.HandleFunc("GET /api/relationships", wrapPerm(h.listAllRelationships, "relationships", "list"))
	mux.HandleFunc("POST /api/devices/{id}/relationships", wrapPerm(h.addRelationship, "relationships", "create"))
	mux.HandleFunc("GET /api/devices/{id}/relationships", wrapPerm(h.getRelationships, "relationships", "read"))
	mux.HandleFunc("GET /api/devices/{id}/related", wrapPerm(h.getRelatedDevices, "relationships", "read"))
	mux.HandleFunc("PATCH /api/devices/{id}/relationships/{child_id}/{type}", wrapPerm(h.updateRelationshipNotes, "relationships", "update"))
	mux.HandleFunc("DELETE /api/devices/{id}/relationships/{child_id}/{type}", wrapPerm(h.removeRelationship, "relationships", "delete"))

	// Discovery routes
	mux.HandleFunc("POST /api/discovery/networks/{id}/scan", wrapPerm(h.startScan, "discovery", "create"))
	mux.HandleFunc("GET /api/discovery/scans", wrapPerm(h.listScans, "discovery", "list"))
	mux.HandleFunc("GET /api/discovery/scans/{id}", wrapPerm(h.getScan, "discovery", "read"))
	mux.HandleFunc("POST /api/discovery/scans/{id}/cancel", wrapPerm(h.cancelScan, "discovery", "delete"))
	mux.HandleFunc("DELETE /api/discovery/scans/{id}", wrapPerm(h.deleteDiscoveryScan, "discovery", "delete"))
	mux.HandleFunc("GET /api/discovery/devices", wrapPerm(h.listDiscoveredDevices, "discovery", "list"))
	mux.HandleFunc("DELETE /api/discovery/devices", wrapPerm(h.deleteDiscoveredDevicesByNetwork, "discovery", "delete"))
	mux.HandleFunc("DELETE /api/discovery/devices/{id}", wrapPerm(h.deleteDiscoveredDevice, "discovery", "delete"))
	mux.HandleFunc("POST /api/discovery/devices/{id}/promote", wrapPerm(h.promoteDevice, "discovery", "create"))
	mux.HandleFunc("GET /api/discovery/rules", wrapPerm(h.listDiscoveryRules, "discovery", "list"))
	mux.HandleFunc("POST /api/discovery/rules", wrapPerm(h.createDiscoveryRule, "discovery", "create"))
	mux.HandleFunc("GET /api/discovery/rules/{id}", wrapPerm(h.getDiscoveryRule, "discovery", "read"))
	mux.HandleFunc("PUT /api/discovery/rules/{id}", wrapPerm(h.updateDiscoveryRule, "discovery", "update"))
	mux.HandleFunc("DELETE /api/discovery/rules/{id}", wrapPerm(h.deleteDiscoveryRule, "discovery", "delete"))

	// Credentials routes (if storage is configured)
	if h.credStore != nil {
		mux.HandleFunc("GET /api/credentials", wrapPerm(h.listCredentials, "credentials", "list"))
		mux.HandleFunc("POST /api/credentials", wrapPerm(h.createCredential, "credentials", "create"))
		mux.HandleFunc("GET /api/credentials/{id}", wrapPerm(h.getCredential, "credentials", "read"))
		mux.HandleFunc("PUT /api/credentials/{id}", wrapPerm(h.updateCredential, "credentials", "update"))
		mux.HandleFunc("DELETE /api/credentials/{id}", wrapPerm(h.deleteCredential, "credentials", "delete"))
	}

	// Scan Profiles routes (if storage is configured)
	if h.profileStore != nil {
		mux.HandleFunc("GET /api/scan-profiles", wrapPerm(h.listProfiles, "scan-profiles", "list"))
		mux.HandleFunc("POST /api/scan-profiles", wrapPerm(h.createProfile, "scan-profiles", "create"))
		mux.HandleFunc("GET /api/scan-profiles/{id}", wrapPerm(h.getProfile, "scan-profiles", "read"))
		mux.HandleFunc("PUT /api/scan-profiles/{id}", wrapPerm(h.updateProfile, "scan-profiles", "update"))
		mux.HandleFunc("DELETE /api/scan-profiles/{id}", wrapPerm(h.deleteProfile, "scan-profiles", "delete"))
	}

	// Scheduled Scans routes (if storage is configured)
	if h.scheduledStore != nil {
		mux.HandleFunc("GET /api/scheduled-scans", wrapPerm(h.listScheduledScans, "scheduled-scans", "list"))
		mux.HandleFunc("POST /api/scheduled-scans", wrapPerm(h.createScheduledScan, "scheduled-scans", "create"))
		mux.HandleFunc("GET /api/scheduled-scans/{id}", wrapPerm(h.getScheduledScan, "scheduled-scans", "read"))
		mux.HandleFunc("PUT /api/scheduled-scans/{id}", wrapPerm(h.updateScheduledScan, "scheduled-scans", "update"))
		mux.HandleFunc("DELETE /api/scheduled-scans/{id}", wrapPerm(h.deleteScheduledScan, "scheduled-scans", "delete"))
	}

	// API Key routes
	mux.HandleFunc("GET /api/keys", wrapPerm(h.listAPIKeys, "apikeys", "list"))
	mux.HandleFunc("POST /api/keys", wrapPerm(h.createAPIKey, "apikeys", "create"))
	mux.HandleFunc("GET /api/keys/{id}", wrapPerm(h.getAPIKey, "apikeys", "read"))
	mux.HandleFunc("DELETE /api/keys/{id}", wrapPerm(h.deleteAPIKey, "apikeys", "delete"))

	// Bulk device operations
	mux.HandleFunc("POST /api/devices/bulk", wrapPerm(h.bulkCreateDevices, "devices", "create"))
	mux.HandleFunc("PUT /api/devices/bulk", wrapPerm(h.bulkUpdateDevices, "devices", "update"))
	mux.HandleFunc("DELETE /api/devices/bulk", wrapPerm(h.bulkDeleteDevices, "devices", "delete"))
	mux.HandleFunc("POST /api/devices/bulk/tags", wrapPerm(h.bulkAddTags, "devices", "update"))
	mux.HandleFunc("DELETE /api/devices/bulk/tags", wrapPerm(h.bulkRemoveTags, "devices", "update"))

	// Bulk network operations
	mux.HandleFunc("POST /api/networks/bulk", wrapPerm(h.bulkCreateNetworks, "networks", "create"))
	mux.HandleFunc("DELETE /api/networks/bulk", wrapPerm(h.bulkDeleteNetworks, "networks", "delete"))

	// Audit log routes
	mux.HandleFunc("GET /api/audit", wrapPerm(h.listAuditLogs, "audit", "list"))
	mux.HandleFunc("GET /api/audit/export", wrapPerm(h.exportAuditLogs, "audit", "list"))
	mux.HandleFunc("GET /api/audit/{id}", wrapPerm(h.getAuditLog, "audit", "list"))

	// Auth routes (no auth required for login)
	loginHandler := LimitBody(h.login)
	if h.loginRateLimiter != nil {
		loginHandler = LoginRateLimitMiddleware(h.loginRateLimiter, h.trustProxy, loginHandler)
	}
	mux.HandleFunc("POST /api/auth/login", loginHandler)
	mux.HandleFunc("POST /api/auth/logout", wrapAuth(h.logout))
	mux.HandleFunc("GET /api/auth/me", wrapAuth(h.getCurrentUser))

	// User routes (require auth + RBAC)
	mux.HandleFunc("GET /api/users", wrapPerm(h.listUsers, "users", "list"))
	mux.HandleFunc("POST /api/users", wrapPerm(h.createUser, "users", "create"))
	mux.HandleFunc("GET /api/users/{id}", wrapPerm(h.getUser, "users", "read"))
	mux.HandleFunc("PUT /api/users/{id}", wrapPerm(h.updateUser, "users", "update"))
	mux.HandleFunc("DELETE /api/users/{id}", wrapPerm(h.deleteUser, "users", "delete"))
	mux.HandleFunc("POST /api/users/{id}/password", wrapAuth(h.changePassword))

	// Role routes (require auth + RBAC)
	mux.HandleFunc("GET /api/roles", wrapPerm(h.listRoles, "roles", "list"))
	mux.HandleFunc("POST /api/roles", wrapPerm(h.createRole, "roles", "create"))
	mux.HandleFunc("GET /api/roles/{id}", wrapPerm(h.getRole, "roles", "read"))
	mux.HandleFunc("PUT /api/roles/{id}", wrapPerm(h.updateRole, "roles", "update"))
	mux.HandleFunc("DELETE /api/roles/{id}", wrapPerm(h.deleteRole, "roles", "delete"))
	mux.HandleFunc("GET /api/roles/{id}/permissions", wrapPerm(h.getRolePermissions, "roles", "read"))
	mux.HandleFunc("GET /api/permissions", wrapPerm(h.listPermissions, "roles", "list"))
	mux.HandleFunc("POST /api/users/grant-role", wrapPerm(h.grantRoleToUser, "roles", "update"))
	mux.HandleFunc("POST /api/users/revoke-role", wrapPerm(h.revokeRoleFromUser, "roles", "update"))
	mux.HandleFunc("GET /api/users/{id}/roles", wrapAuth(h.getUserRoles))
	mux.HandleFunc("GET /api/users/{id}/permissions", wrapAuth(h.getUserPermissions))

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
	config.AddNavItem(NavItem{Label: "Roles", Path: "/roles", Icon: "shield", Order: 16})
	h.writeJSON(w, http.StatusOK, config.Build())
}
