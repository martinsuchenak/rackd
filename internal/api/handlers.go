package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/martinsuchenak/rackd/internal/audit"
	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/discovery"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/service"
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
	svc              *service.Services
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

func (h *Handler) SetServices(svc *service.Services) {
	h.svc = svc
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

	// Datacenter routes (RBAC enforced in service layer)
	mux.HandleFunc("GET /api/datacenters", wrapAuth(h.listDatacenters))
	mux.HandleFunc("POST /api/datacenters", wrapAuth(h.createDatacenter))
	mux.HandleFunc("GET /api/datacenters/{id}", wrapAuth(h.getDatacenter))
	mux.HandleFunc("PUT /api/datacenters/{id}", wrapAuth(h.updateDatacenter))
	mux.HandleFunc("DELETE /api/datacenters/{id}", wrapAuth(h.deleteDatacenter))
	mux.HandleFunc("GET /api/datacenters/{id}/devices", wrapAuth(h.getDatacenterDevices))

	// Network routes (RBAC enforced in service layer)
	mux.HandleFunc("GET /api/networks", wrapAuth(h.listNetworks))
	mux.HandleFunc("POST /api/networks", wrapAuth(h.createNetwork))
	mux.HandleFunc("GET /api/networks/{id}", wrapAuth(h.getNetwork))
	mux.HandleFunc("PUT /api/networks/{id}", wrapAuth(h.updateNetwork))
	mux.HandleFunc("DELETE /api/networks/{id}", wrapAuth(h.deleteNetwork))
	mux.HandleFunc("GET /api/networks/{id}/devices", wrapAuth(h.getNetworkDevices))
	mux.HandleFunc("GET /api/networks/{id}/utilization", wrapAuth(h.getNetworkUtilization))
	mux.HandleFunc("GET /api/networks/{id}/pools", wrapAuth(h.listNetworkPools))
	mux.HandleFunc("POST /api/networks/{id}/pools", wrapAuth(h.createNetworkPool))

	// Pool routes (RBAC enforced in service layer)
	mux.HandleFunc("GET /api/pools/{id}", wrapAuth(h.getNetworkPool))
	mux.HandleFunc("PUT /api/pools/{id}", wrapAuth(h.updateNetworkPool))
	mux.HandleFunc("DELETE /api/pools/{id}", wrapAuth(h.deleteNetworkPool))
	mux.HandleFunc("GET /api/pools/{id}/next-ip", wrapAuth(h.getNextIP))
	mux.HandleFunc("GET /api/pools/{id}/heatmap", wrapAuth(h.getPoolHeatmap))

	// Device routes (RBAC enforced in service layer)
	mux.HandleFunc("GET /api/devices", wrapAuth(h.listDevices))
	mux.HandleFunc("POST /api/devices", wrapAuth(h.createDevice))
	mux.HandleFunc("GET /api/devices/{id}", wrapAuth(h.getDevice))
	mux.HandleFunc("PUT /api/devices/{id}", wrapAuth(h.updateDevice))
	mux.HandleFunc("DELETE /api/devices/{id}", wrapAuth(h.deleteDevice))

	// Search routes
	mux.HandleFunc("GET /api/search", wrapPerm(h.search, "search", "read"))

	// Relationship routes (RBAC enforced in service layer)
	mux.HandleFunc("GET /api/relationships", wrapAuth(h.listAllRelationships))
	mux.HandleFunc("POST /api/devices/{id}/relationships", wrapAuth(h.addRelationship))
	mux.HandleFunc("GET /api/devices/{id}/relationships", wrapAuth(h.getRelationships))
	mux.HandleFunc("GET /api/devices/{id}/related", wrapAuth(h.getRelatedDevices))
	mux.HandleFunc("PATCH /api/devices/{id}/relationships/{child_id}/{type}", wrapAuth(h.updateRelationshipNotes))
	mux.HandleFunc("DELETE /api/devices/{id}/relationships/{child_id}/{type}", wrapAuth(h.removeRelationship))

	// Discovery routes (RBAC enforced in service layer)
	mux.HandleFunc("POST /api/discovery/networks/{id}/scan", wrapAuth(h.startScan))
	mux.HandleFunc("GET /api/discovery/scans", wrapAuth(h.listScans))
	mux.HandleFunc("GET /api/discovery/scans/{id}", wrapAuth(h.getScan))
	mux.HandleFunc("POST /api/discovery/scans/{id}/cancel", wrapAuth(h.cancelScan))
	mux.HandleFunc("DELETE /api/discovery/scans/{id}", wrapAuth(h.deleteDiscoveryScan))
	mux.HandleFunc("GET /api/discovery/devices", wrapAuth(h.listDiscoveredDevices))
	mux.HandleFunc("DELETE /api/discovery/devices", wrapAuth(h.deleteDiscoveredDevicesByNetwork))
	mux.HandleFunc("DELETE /api/discovery/devices/{id}", wrapAuth(h.deleteDiscoveredDevice))
	mux.HandleFunc("POST /api/discovery/devices/{id}/promote", wrapAuth(h.promoteDevice))
	mux.HandleFunc("GET /api/discovery/rules", wrapAuth(h.listDiscoveryRules))
	mux.HandleFunc("POST /api/discovery/rules", wrapAuth(h.createDiscoveryRule))
	mux.HandleFunc("GET /api/discovery/rules/{id}", wrapAuth(h.getDiscoveryRule))
	mux.HandleFunc("PUT /api/discovery/rules/{id}", wrapAuth(h.updateDiscoveryRule))
	mux.HandleFunc("DELETE /api/discovery/rules/{id}", wrapAuth(h.deleteDiscoveryRule))

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

	// Bulk device operations (RBAC enforced in service layer)
	mux.HandleFunc("POST /api/devices/bulk", wrapAuth(h.bulkCreateDevices))
	mux.HandleFunc("PUT /api/devices/bulk", wrapAuth(h.bulkUpdateDevices))
	mux.HandleFunc("DELETE /api/devices/bulk", wrapAuth(h.bulkDeleteDevices))
	mux.HandleFunc("POST /api/devices/bulk/tags", wrapAuth(h.bulkAddTags))
	mux.HandleFunc("DELETE /api/devices/bulk/tags", wrapAuth(h.bulkRemoveTags))

	// Bulk network operations (RBAC enforced in service layer)
	mux.HandleFunc("POST /api/networks/bulk", wrapAuth(h.bulkCreateNetworks))
	mux.HandleFunc("DELETE /api/networks/bulk", wrapAuth(h.bulkDeleteNetworks))

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

	// User routes (RBAC enforced in service layer)
	mux.HandleFunc("GET /api/users", wrapAuth(h.listUsers))
	mux.HandleFunc("POST /api/users", wrapAuth(h.createUser))
	mux.HandleFunc("GET /api/users/{id}", wrapAuth(h.getUser))
	mux.HandleFunc("PUT /api/users/{id}", wrapAuth(h.updateUser))
	mux.HandleFunc("DELETE /api/users/{id}", wrapAuth(h.deleteUser))
	mux.HandleFunc("POST /api/users/{id}/password", wrapAuth(h.changePassword))

	// Role routes (RBAC enforced in service layer)
	mux.HandleFunc("GET /api/roles", wrapAuth(h.listRoles))
	mux.HandleFunc("POST /api/roles", wrapAuth(h.createRole))
	mux.HandleFunc("GET /api/roles/{id}", wrapAuth(h.getRole))
	mux.HandleFunc("PUT /api/roles/{id}", wrapAuth(h.updateRole))
	mux.HandleFunc("DELETE /api/roles/{id}", wrapAuth(h.deleteRole))
	mux.HandleFunc("GET /api/roles/{id}/permissions", wrapAuth(h.getRolePermissions))
	mux.HandleFunc("GET /api/permissions", wrapAuth(h.listPermissions))
	mux.HandleFunc("POST /api/users/grant-role", wrapAuth(h.grantRoleToUser))
	mux.HandleFunc("POST /api/users/revoke-role", wrapAuth(h.revokeRoleFromUser))
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

func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	switch {
	case errors.Is(err, service.ErrNotFound):
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrForbidden):
		h.writeError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden")
	case errors.Is(err, service.ErrUnauthenticated):
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	case errors.Is(err, service.ErrAlreadyExists):
		h.writeError(w, http.StatusConflict, "ALREADY_EXISTS", err.Error())
	case errors.Is(err, service.ErrIPNotAvailable):
		h.writeError(w, http.StatusConflict, "IP_NOT_AVAILABLE", "No IP addresses available")
	case errors.Is(err, service.ErrValidation):
		h.writeValidationErrors(w, toValidationErrors(err))
	case errors.Is(err, service.ErrSelfDelete):
		h.writeError(w, http.StatusBadRequest, "CANNOT_DELETE_SELF", err.Error())
	case errors.Is(err, service.ErrSystemRole):
		h.writeError(w, http.StatusBadRequest, "SYSTEM_ROLE", err.Error())
	default:
		h.internalError(w, err)
	}
}

func toValidationErrors(err error) ValidationErrors {
	// Check api.ValidationErrors
	if verrs, ok := err.(ValidationErrors); ok {
		return verrs
	}
	// Check service.ValidationErrors and convert
	var svcErrs service.ValidationErrors
	if errors.As(err, &svcErrs) {
		result := make(ValidationErrors, len(svcErrs))
		for i, e := range svcErrs {
			result[i] = ValidationError{Field: e.Field, Message: e.Message}
		}
		return result
	}
	return ValidationErrors{{Field: "", Message: err.Error()}}
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
