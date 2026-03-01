package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/discovery"
	"github.com/martinsuchenak/rackd/internal/log"
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

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	wrapAuth := func(handler http.HandlerFunc) http.HandlerFunc {
		handler = LimitBody(handler)
		if h.sessionManager != nil {
			return AuthMiddlewareWithSessions(h.store, h.sessionManager, handler)
		}
		return AuthMiddleware(h.store, handler)
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
	mux.HandleFunc("GET /api/devices/status-counts", wrapAuth(h.getDeviceStatusCounts))
	mux.HandleFunc("GET /api/devices/{id}", wrapAuth(h.getDevice))
	mux.HandleFunc("PUT /api/devices/{id}", wrapAuth(h.updateDevice))
	mux.HandleFunc("DELETE /api/devices/{id}", wrapAuth(h.deleteDevice))

	// Dashboard routes (RBAC enforced in service layer)
	mux.HandleFunc("GET /api/dashboard", wrapAuth(h.getDashboardStats))
	mux.HandleFunc("GET /api/dashboard/trend", wrapAuth(h.getUtilizationTrend))

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
		mux.HandleFunc("GET /api/credentials", wrapAuth(h.listCredentials))
		mux.HandleFunc("POST /api/credentials", wrapAuth(h.createCredential))
		mux.HandleFunc("GET /api/credentials/{id}", wrapAuth(h.getCredential))
		mux.HandleFunc("PUT /api/credentials/{id}", wrapAuth(h.updateCredential))
		mux.HandleFunc("DELETE /api/credentials/{id}", wrapAuth(h.deleteCredential))
	}

	// Scan Profiles routes (if storage is configured)
	if h.profileStore != nil {
		mux.HandleFunc("GET /api/scan-profiles", wrapAuth(h.listProfiles))
		mux.HandleFunc("POST /api/scan-profiles", wrapAuth(h.createProfile))
		mux.HandleFunc("GET /api/scan-profiles/{id}", wrapAuth(h.getProfile))
		mux.HandleFunc("PUT /api/scan-profiles/{id}", wrapAuth(h.updateProfile))
		mux.HandleFunc("DELETE /api/scan-profiles/{id}", wrapAuth(h.deleteProfile))
	}

	// Scheduled Scans routes (if storage is configured)
	if h.scheduledStore != nil {
		mux.HandleFunc("GET /api/scheduled-scans", wrapAuth(h.listScheduledScans))
		mux.HandleFunc("POST /api/scheduled-scans", wrapAuth(h.createScheduledScan))
		mux.HandleFunc("GET /api/scheduled-scans/{id}", wrapAuth(h.getScheduledScan))
		mux.HandleFunc("PUT /api/scheduled-scans/{id}", wrapAuth(h.updateScheduledScan))
		mux.HandleFunc("DELETE /api/scheduled-scans/{id}", wrapAuth(h.deleteScheduledScan))
	}

	// API Key routes (RBAC enforced in service layer)
	mux.HandleFunc("GET /api/keys", wrapAuth(h.listAPIKeys))
	mux.HandleFunc("POST /api/keys", wrapAuth(h.createAPIKey))
	mux.HandleFunc("GET /api/keys/{id}", wrapAuth(h.getAPIKey))
	mux.HandleFunc("DELETE /api/keys/{id}", wrapAuth(h.deleteAPIKey))

	// Bulk device operations (RBAC enforced in service layer)
	mux.HandleFunc("POST /api/devices/bulk", wrapAuth(h.bulkCreateDevices))
	mux.HandleFunc("PUT /api/devices/bulk", wrapAuth(h.bulkUpdateDevices))
	mux.HandleFunc("DELETE /api/devices/bulk", wrapAuth(h.bulkDeleteDevices))
	mux.HandleFunc("POST /api/devices/bulk/tags", wrapAuth(h.bulkAddTags))
	mux.HandleFunc("DELETE /api/devices/bulk/tags", wrapAuth(h.bulkRemoveTags))

	// Bulk network operations (RBAC enforced in service layer)
	mux.HandleFunc("POST /api/networks/bulk", wrapAuth(h.bulkCreateNetworks))
	mux.HandleFunc("DELETE /api/networks/bulk", wrapAuth(h.bulkDeleteNetworks))

	// Search routes (RBAC enforced in service layer)
	mux.HandleFunc("GET /api/search", wrapAuth(h.search))

	// Audit log routes (RBAC enforced in service layer)
	mux.HandleFunc("GET /api/audit", wrapAuth(h.listAuditLogs))
	mux.HandleFunc("GET /api/audit/export", wrapAuth(h.exportAuditLogs))
	mux.HandleFunc("GET /api/audit/{id}", wrapAuth(h.getAuditLog))

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
	mux.HandleFunc("POST /api/users/{id}/reset-password", wrapAuth(h.resetPassword))

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

	// OAuth 2.1 routes (conditional on OAuth service being configured)
	if h.svc != nil && h.svc.OAuth != nil {
		// Well-known metadata endpoints (no auth required)
		mux.HandleFunc("GET /.well-known/oauth-protected-resource", LimitBody(h.oauthProtectedResource))
		mux.HandleFunc("GET /.well-known/oauth-authorization-server", LimitBody(h.oauthAuthorizationServerMetadata))

		// OAuth flow endpoints (no auth required per OAuth/MCP spec)
		mux.HandleFunc("POST /mcp-oauth/register", LimitBody(h.oauthRegister))
		mux.HandleFunc("GET /mcp-oauth/authorize", LimitBody(h.oauthAuthorize))
		mux.HandleFunc("POST /mcp-oauth/authorize", LimitBody(h.oauthAuthorizeSubmit))
		mux.HandleFunc("POST /mcp-oauth/token", LimitBody(h.oauthToken))
		mux.HandleFunc("POST /mcp-oauth/revoke", LimitBody(h.oauthRevoke))

		// OAuth client management (requires auth)
		mux.HandleFunc("GET /api/oauth/clients", wrapAuth(h.oauthListClients))
		mux.HandleFunc("DELETE /api/oauth/clients/{id}", wrapAuth(h.oauthDeleteClient))
	}

	// Conflict routes (RBAC enforced in service layer)
	mux.HandleFunc("GET /api/conflicts", wrapAuth(h.listConflicts))
	mux.HandleFunc("GET /api/conflicts/summary", wrapAuth(h.getConflictSummary))
	mux.HandleFunc("POST /api/conflicts/detect", wrapAuth(h.detectConflicts))
	mux.HandleFunc("GET /api/conflicts/{id}", wrapAuth(h.getConflict))
	mux.HandleFunc("POST /api/conflicts/{id}/resolve", wrapAuth(h.resolveConflict))
	mux.HandleFunc("DELETE /api/conflicts/{id}", wrapAuth(h.deleteConflict))
	mux.HandleFunc("GET /api/devices/{id}/conflicts", wrapAuth(h.getDeviceConflicts))

	// Reservation routes (RBAC enforced in service layer)
	mux.HandleFunc("GET /api/reservations", wrapAuth(h.listReservations))
	mux.HandleFunc("POST /api/reservations", wrapAuth(h.createReservationWithDefaults))
	mux.HandleFunc("GET /api/reservations/{id}", wrapAuth(h.getReservation))
	mux.HandleFunc("PUT /api/reservations/{id}", wrapAuth(h.updateReservation))
	mux.HandleFunc("DELETE /api/reservations/{id}", wrapAuth(h.deleteReservation))
	mux.HandleFunc("POST /api/reservations/{id}/release", wrapAuth(h.releaseReservation))
	mux.HandleFunc("GET /api/pools/{id}/reservations", wrapAuth(h.listPoolReservations))

	// Webhook routes (RBAC enforced in service layer)
	mux.HandleFunc("GET /api/webhooks", wrapAuth(h.listWebhooks))
	mux.HandleFunc("POST /api/webhooks", wrapAuth(h.createWebhook))
	mux.HandleFunc("GET /api/webhooks/events", wrapAuth(h.getEventTypes))
	mux.HandleFunc("GET /api/webhooks/{id}", wrapAuth(h.getWebhook))
	mux.HandleFunc("PUT /api/webhooks/{id}", wrapAuth(h.updateWebhook))
	mux.HandleFunc("DELETE /api/webhooks/{id}", wrapAuth(h.deleteWebhook))
	mux.HandleFunc("POST /api/webhooks/{id}/ping", wrapAuth(h.pingWebhook))
	mux.HandleFunc("GET /api/webhooks/{id}/deliveries", wrapAuth(h.listWebhookDeliveries))
	mux.HandleFunc("GET /api/webhooks/{id}/deliveries/{deliveryId}", wrapAuth(h.getWebhookDelivery))

	// Custom field routes (RBAC enforced in service layer)
	mux.HandleFunc("GET /api/custom-fields", wrapAuth(h.listCustomFieldDefinitions))
	mux.HandleFunc("POST /api/custom-fields", wrapAuth(h.createCustomFieldDefinition))
	mux.HandleFunc("GET /api/custom-fields/types", wrapAuth(h.getCustomFieldTypes))
	mux.HandleFunc("GET /api/custom-fields/{id}", wrapAuth(h.getCustomFieldDefinition))
	mux.HandleFunc("PUT /api/custom-fields/{id}", wrapAuth(h.updateCustomFieldDefinition))
	mux.HandleFunc("DELETE /api/custom-fields/{id}", wrapAuth(h.deleteCustomFieldDefinition))

	// Circuit routes
	mux.HandleFunc("GET /api/circuits", wrapAuth(h.listCircuits))
	mux.HandleFunc("POST /api/circuits", wrapAuth(h.createCircuit))
	mux.HandleFunc("GET /api/circuits/{id}", wrapAuth(h.getCircuit))
	mux.HandleFunc("PUT /api/circuits/{id}", wrapAuth(h.updateCircuit))
	mux.HandleFunc("DELETE /api/circuits/{id}", wrapAuth(h.deleteCircuit))

	// NAT routes
	mux.HandleFunc("GET /api/nat", wrapAuth(h.listNATMappings))
	mux.HandleFunc("POST /api/nat", wrapAuth(h.createNATMapping))
	mux.HandleFunc("GET /api/nat/{id}", wrapAuth(h.getNATMapping))
	mux.HandleFunc("PUT /api/nat/{id}", wrapAuth(h.updateNATMapping))
	mux.HandleFunc("DELETE /api/nat/{id}", wrapAuth(h.deleteNATMapping))

	// DNS routes (RBAC enforced in service layer)
	if h.svc != nil && h.svc.DNS != nil {
		// Provider routes
		mux.HandleFunc("GET /api/dns/providers", wrapAuth(h.listDNSProviders))
		mux.HandleFunc("POST /api/dns/providers", wrapAuth(h.createDNSProvider))
		mux.HandleFunc("GET /api/dns/providers/{id}", wrapAuth(h.getDNSProvider))
		mux.HandleFunc("PUT /api/dns/providers/{id}", wrapAuth(h.updateDNSProvider))
		mux.HandleFunc("DELETE /api/dns/providers/{id}", wrapAuth(h.deleteDNSProvider))
		mux.HandleFunc("POST /api/dns/providers/{id}/test", wrapAuth(h.testDNSProvider))
		mux.HandleFunc("GET /api/dns/providers/{id}/zones", wrapAuth(h.listDNSProviderZones))
		// Zone routes
		mux.HandleFunc("GET /api/dns/zones", wrapAuth(h.listDNSZones))
		mux.HandleFunc("POST /api/dns/zones", wrapAuth(h.createDNSZone))
		mux.HandleFunc("GET /api/dns/zones/{id}", wrapAuth(h.getDNSZone))
		mux.HandleFunc("PUT /api/dns/zones/{id}", wrapAuth(h.updateDNSZone))
		mux.HandleFunc("DELETE /api/dns/zones/{id}", wrapAuth(h.deleteDNSZone))
		mux.HandleFunc("POST /api/dns/zones/{id}/sync", wrapAuth(h.syncDNSZone))
		mux.HandleFunc("POST /api/dns/zones/{id}/import", wrapAuth(h.importDNSZone))
		mux.HandleFunc("GET /api/dns/zones/{id}/records", wrapAuth(h.listDNSZoneRecords))
		// Record routes
		mux.HandleFunc("GET /api/dns/records/{id}", wrapAuth(h.getDNSRecord))
		mux.HandleFunc("PUT /api/dns/records/{id}", wrapAuth(h.updateDNSRecord))
		mux.HandleFunc("DELETE /api/dns/records/{id}", wrapAuth(h.deleteDNSRecord))
	}

	// Health check routes (no auth required)
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /readyz", h.readyz)

	// Metrics route (requires auth)
	mux.HandleFunc("GET /metrics", wrapAuth(h.metricsHandler))
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
	config.AddNavItem(NavItem{Label: "Users", Path: "/users", Icon: "user", Order: 15, RequiredPermissions: []PermissionCheck{{Resource: "users", Action: "list"}}})
	config.AddNavItem(NavItem{Label: "Roles", Path: "/roles", Icon: "shield", Order: 16, RequiredPermissions: []PermissionCheck{{Resource: "roles", Action: "list"}}})
	h.writeJSON(w, http.StatusOK, config.Build())
}
