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
	"github.com/martinsuchenak/rackd/internal/ui"
	"github.com/martinsuchenak/rackd/pkg/model"
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
	proxyResolver    *TrustedProxyResolver
	svc              *service.Services
	indexHTML        []byte
}

// HandlerOption configures a Handler during construction.
type HandlerOption func(*Handler)

// WithSessionManager sets the session manager for authentication.
func WithSessionManager(sm *auth.SessionManager) HandlerOption {
	return func(h *Handler) { h.sessionManager = sm }
}

// WithCredentialsStorage sets the credentials storage backend.
func WithCredentialsStorage(cs credentials.Storage) HandlerOption {
	return func(h *Handler) { h.credStore = cs }
}

// WithProfileStorage sets the scan profile storage backend.
func WithProfileStorage(ps storage.ProfileStorage) HandlerOption {
	return func(h *Handler) { h.profileStore = ps }
}

// WithScheduledScanStorage sets the scheduled scan storage backend.
func WithScheduledScanStorage(ss storage.ScheduledScanStorage) HandlerOption {
	return func(h *Handler) { h.scheduledStore = ss }
}

// WithLoginRateLimiter sets the rate limiter for login attempts.
func WithLoginRateLimiter(rl *RateLimiter) HandlerOption {
	return func(h *Handler) { h.loginRateLimiter = rl }
}

// WithCookieConfig sets cookie security and session TTL.
func WithCookieConfig(secure bool, sessionTTL time.Duration) HandlerOption {
	return func(h *Handler) {
		h.cookieSecure = secure
		h.sessionTTL = sessionTTL
	}
}

// WithProxyResolver sets the trusted-proxy resolver used for client-IP
// detection behind reverse proxies.
func WithProxyResolver(resolver *TrustedProxyResolver) HandlerOption {
	return func(h *Handler) { h.proxyResolver = resolver }
}

// WithServices sets the service registry.
func WithServices(svc *service.Services) HandlerOption {
	return func(h *Handler) { h.svc = svc }
}

// WithIndexHTML sets the web UI page served for browser visits to API routes
// that the UI handles, such as the OAuth consent page. It defaults to the
// built-in UI's.
func WithIndexHTML(html []byte) HandlerOption {
	return func(h *Handler) { h.indexHTML = html }
}

// NewHandler creates a new API handler with the given storage, scanner, and options.
func NewHandler(s storage.ExtendedStorage, scanner discovery.Scanner, opts ...HandlerOption) *Handler {
	h := &Handler{store: s, scanner: scanner}
	for _, opt := range opts {
		opt(h)
	}
	if h.indexHTML == nil {
		h.indexHTML = ui.IndexHTML()
	}
	return h
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	route := func(pattern string, handler http.HandlerFunc) { HandleAPI(mux, pattern, handler) }

	wrapAuth := h.authenticated

	// Each sensitive route gets its own buckets with the login limit's
	// settings, so failed logins from an address can't also block that
	// address's token refreshes or password changes, and the other way round.
	strictLimit := func(handler http.HandlerFunc) http.HandlerFunc {
		if h.loginRateLimiter == nil {
			return handler
		}
		limiter := NewRateLimiter(h.loginRateLimiter.requests, h.loginRateLimiter.window)
		return LoginRateLimitMiddleware(limiter, h.proxyResolver, handler)
	}

	wrapSensitiveAuth := func(handler http.HandlerFunc) http.HandlerFunc {
		return strictLimit(wrapAuth(handler))
	}

	wrapSensitiveNoAuth := func(handler http.HandlerFunc) http.HandlerFunc {
		return strictLimit(LimitBody(handler))
	}

	// Datacenter routes (RBAC enforced in service layer)
	route("GET /datacenters", wrapAuth(h.listDatacenters))
	route("POST /datacenters", wrapAuth(h.createDatacenter))
	route("GET /datacenters/{id}", wrapAuth(h.getDatacenter))
	route("PUT /datacenters/{id}", wrapAuth(h.updateDatacenter))
	route("DELETE /datacenters/{id}", wrapAuth(h.deleteDatacenter))
	route("GET /datacenters/{id}/devices", wrapAuth(h.getDatacenterDevices))

	// Network routes (RBAC enforced in service layer)
	route("GET /networks", wrapAuth(h.listNetworks))
	route("POST /networks", wrapAuth(h.createNetwork))
	route("GET /networks/{id}", wrapAuth(h.getNetwork))
	route("PUT /networks/{id}", wrapAuth(h.updateNetwork))
	route("DELETE /networks/{id}", wrapAuth(h.deleteNetwork))
	route("GET /networks/{id}/devices", wrapAuth(h.getNetworkDevices))
	route("GET /networks/{id}/utilization", wrapAuth(h.getNetworkUtilization))
	route("GET /networks/{id}/pools", wrapAuth(h.listNetworkPools))
	route("POST /networks/{id}/pools", wrapAuth(h.createNetworkPool))

	// Pool routes (RBAC enforced in service layer)
	route("GET /pools/{id}", wrapAuth(h.getNetworkPool))
	route("PUT /pools/{id}", wrapAuth(h.updateNetworkPool))
	route("DELETE /pools/{id}", wrapAuth(h.deleteNetworkPool))
	route("GET /pools/{id}/next-ip", wrapAuth(h.getNextIP))
	route("GET /pools/{id}/heatmap", wrapAuth(h.getPoolHeatmap))

	// Device routes (RBAC enforced in service layer)
	route("GET /devices", wrapAuth(h.listDevices))
	route("POST /devices", wrapAuth(h.createDevice))
	route("GET /devices/status-counts", wrapAuth(h.getDeviceStatusCounts))
	route("GET /devices/search", wrapAuth(h.searchDevices))
	route("GET /devices/{id}", wrapAuth(h.getDevice))
	route("PUT /devices/{id}", wrapAuth(h.updateDevice))
	route("DELETE /devices/{id}", wrapAuth(h.deleteDevice))

	// Dashboard routes (RBAC enforced in service layer)
	route("GET /dashboard", wrapAuth(h.getDashboardStats))
	route("GET /dashboard/trend", wrapAuth(h.getUtilizationTrend))

	// Relationship routes (RBAC enforced in service layer)
	route("GET /relationships", wrapAuth(h.listAllRelationships))
	route("POST /devices/{id}/relationships", wrapAuth(h.addRelationship))
	route("GET /devices/{id}/relationships", wrapAuth(h.getRelationships))
	route("GET /devices/{id}/related", wrapAuth(h.getRelatedDevices))
	route("PATCH /devices/{id}/relationships/{child_id}/{type}", wrapAuth(h.updateRelationshipNotes))
	route("DELETE /devices/{id}/relationships/{child_id}/{type}", wrapAuth(h.removeRelationship))

	// Discovery routes (RBAC enforced in service layer)
	route("POST /discovery/networks/{id}/scan", wrapAuth(h.startScan))
	route("GET /discovery/scans", wrapAuth(h.listScans))
	route("GET /discovery/scans/{id}", wrapAuth(h.getScan))
	route("POST /discovery/scans/{id}/cancel", wrapAuth(h.cancelScan))
	route("DELETE /discovery/scans/{id}", wrapAuth(h.deleteDiscoveryScan))
	route("GET /discovery/devices", wrapAuth(h.listDiscoveredDevices))
	route("DELETE /discovery/devices", wrapAuth(h.deleteDiscoveredDevicesByNetwork))
	route("DELETE /discovery/devices/{id}", wrapAuth(h.deleteDiscoveredDevice))
	route("POST /discovery/devices/{id}/promote", wrapAuth(h.promoteDevice))
	route("GET /discovery/rules", wrapAuth(h.listDiscoveryRules))
	route("POST /discovery/rules", wrapAuth(h.createDiscoveryRule))
	route("GET /discovery/rules/{id}", wrapAuth(h.getDiscoveryRule))
	route("PUT /discovery/rules/{id}", wrapAuth(h.updateDiscoveryRule))
	route("DELETE /discovery/rules/{id}", wrapAuth(h.deleteDiscoveryRule))

	// Credentials routes (if storage is configured)
	if h.credStore != nil {
		route("GET /credentials", wrapAuth(h.listCredentials))
		route("POST /credentials", wrapAuth(h.createCredential))
		route("GET /credentials/{id}", wrapAuth(h.getCredential))
		route("PUT /credentials/{id}", wrapAuth(h.updateCredential))
		route("DELETE /credentials/{id}", wrapAuth(h.deleteCredential))
	}

	// Scan Profiles routes (if storage is configured)
	if h.profileStore != nil {
		route("GET /scan-profiles", wrapAuth(h.listProfiles))
		route("POST /scan-profiles", wrapAuth(h.createProfile))
		route("GET /scan-profiles/{id}", wrapAuth(h.getProfile))
		route("PUT /scan-profiles/{id}", wrapAuth(h.updateProfile))
		route("DELETE /scan-profiles/{id}", wrapAuth(h.deleteProfile))
	}

	// Scheduled Scans routes (if storage is configured)
	if h.scheduledStore != nil {
		route("GET /scheduled-scans", wrapAuth(h.listScheduledScans))
		route("POST /scheduled-scans", wrapAuth(h.createScheduledScan))
		route("GET /scheduled-scans/{id}", wrapAuth(h.getScheduledScan))
		route("PUT /scheduled-scans/{id}", wrapAuth(h.updateScheduledScan))
		route("DELETE /scheduled-scans/{id}", wrapAuth(h.deleteScheduledScan))
	}

	// API Key routes (RBAC enforced in service layer)
	route("GET /keys", wrapAuth(h.listAPIKeys))
	route("POST /keys", wrapSensitiveAuth(h.createAPIKey))
	route("GET /keys/{id}", wrapAuth(h.getAPIKey))
	route("DELETE /keys/{id}", wrapAuth(h.deleteAPIKey))

	// Bulk device operations (RBAC enforced in service layer)
	route("POST /devices/bulk", wrapSensitiveAuth(h.bulkCreateDevices))
	route("PUT /devices/bulk", wrapSensitiveAuth(h.bulkUpdateDevices))
	route("DELETE /devices/bulk", wrapSensitiveAuth(h.bulkDeleteDevices))
	route("POST /devices/bulk/tags", wrapSensitiveAuth(h.bulkAddTags))
	route("DELETE /devices/bulk/tags", wrapSensitiveAuth(h.bulkRemoveTags))

	// Bulk network operations (RBAC enforced in service layer)
	route("POST /networks/bulk", wrapSensitiveAuth(h.bulkCreateNetworks))
	route("DELETE /networks/bulk", wrapSensitiveAuth(h.bulkDeleteNetworks))

	// Search routes (RBAC enforced in service layer)
	route("GET /search", wrapAuth(h.search))

	// Audit log routes (RBAC enforced in service layer)
	route("GET /audit", wrapAuth(h.listAuditLogs))
	route("GET /audit/export", wrapAuth(h.exportAuditLogs))
	route("GET /audit/{id}", wrapAuth(h.getAuditLog))

	// Recent logs routes (RBAC enforced in service layer)
	route("GET /logs", wrapAuth(h.listLogs))
	route("GET /logs/export", wrapAuth(h.exportLogs))
	route("GET /logs/{id}", wrapAuth(h.getLogEntry))

	// Auth routes (no auth required for login)
	loginHandler := LimitBody(h.login)
	if h.loginRateLimiter != nil {
		loginHandler = LoginRateLimitMiddleware(h.loginRateLimiter, h.proxyResolver, loginHandler)
	}
	route("POST /auth/login", loginHandler)
	route("POST /auth/logout", wrapAuth(h.logout))
	route("GET /auth/me", wrapAuth(h.getCurrentUser))

	// User routes (RBAC enforced in service layer)
	route("GET /users", wrapAuth(h.listUsers))
	route("POST /users", wrapSensitiveAuth(h.createUser))
	route("GET /users/{id}", wrapAuth(h.getUser))
	route("PUT /users/{id}", wrapAuth(h.updateUser))
	route("DELETE /users/{id}", wrapAuth(h.deleteUser))
	route("POST /users/{id}/password", wrapSensitiveAuth(h.changePassword))
	route("POST /users/{id}/reset-password", wrapSensitiveAuth(h.resetPassword))

	// Role routes (RBAC enforced in service layer)
	route("GET /roles", wrapAuth(h.listRoles))
	route("POST /roles", wrapAuth(h.createRole))
	route("GET /roles/{id}", wrapAuth(h.getRole))
	route("PUT /roles/{id}", wrapAuth(h.updateRole))
	route("DELETE /roles/{id}", wrapAuth(h.deleteRole))
	route("GET /roles/{id}/permissions", wrapAuth(h.getRolePermissions))
	route("GET /permissions", wrapAuth(h.listPermissions))
	route("POST /users/grant-role", wrapAuth(h.grantRoleToUser))
	route("POST /users/revoke-role", wrapAuth(h.revokeRoleFromUser))
	route("GET /users/{id}/roles", wrapAuth(h.getUserRoles))
	route("GET /users/{id}/permissions", wrapAuth(h.getUserPermissions))

	// OAuth 2.1 routes (conditional on OAuth service being configured)
	if h.svc != nil && h.svc.OAuth != nil {
		// Well-known metadata endpoints (no auth required)
		mux.HandleFunc("GET /.well-known/oauth-protected-resource", LimitBody(h.oauthProtectedResource))
		mux.HandleFunc("GET /.well-known/oauth-authorization-server", LimitBody(h.oauthAuthorizationServerMetadata))

		// OAuth flow endpoints (no auth required per OAuth/MCP spec)
		mux.HandleFunc("POST /mcp-oauth/register", wrapSensitiveNoAuth(h.oauthRegister))
		mux.HandleFunc("GET /mcp-oauth/authorize", LimitBody(h.oauthAuthorize))
		mux.HandleFunc("POST /mcp-oauth/authorize", wrapSensitiveNoAuth(h.oauthAuthorizeSubmit))
		mux.HandleFunc("POST /mcp-oauth/token", wrapSensitiveNoAuth(h.oauthToken))
		mux.HandleFunc("POST /mcp-oauth/revoke", LimitBody(h.oauthRevoke))

		// OAuth client management (requires auth)
		route("GET /oauth/clients", wrapAuth(h.oauthListClients))
		route("DELETE /oauth/clients/{id}", wrapAuth(h.oauthDeleteClient))
	}

	// Conflict routes (RBAC enforced in service layer)
	route("GET /conflicts", wrapAuth(h.listConflicts))
	route("GET /conflicts/summary", wrapAuth(h.getConflictSummary))
	route("POST /conflicts/detect", wrapAuth(h.detectConflicts))
	route("GET /conflicts/{id}", wrapAuth(h.getConflict))
	route("POST /conflicts/{id}/resolve", wrapAuth(h.resolveConflict))
	route("DELETE /conflicts/{id}", wrapAuth(h.deleteConflict))
	route("GET /devices/{id}/conflicts", wrapAuth(h.getDeviceConflicts))

	// Reservation routes (RBAC enforced in service layer)
	route("GET /reservations", wrapAuth(h.listReservations))
	route("POST /reservations", wrapAuth(h.createReservationWithDefaults))
	route("GET /reservations/{id}", wrapAuth(h.getReservation))
	route("PUT /reservations/{id}", wrapAuth(h.updateReservation))
	route("DELETE /reservations/{id}", wrapAuth(h.deleteReservation))
	route("POST /reservations/{id}/release", wrapAuth(h.releaseReservation))
	route("GET /pools/{id}/reservations", wrapAuth(h.listPoolReservations))

	// Webhook routes (RBAC enforced in service layer)
	route("GET /webhooks", wrapAuth(h.listWebhooks))
	route("POST /webhooks", wrapAuth(h.createWebhook))
	route("GET /webhooks/events", wrapAuth(h.getEventTypes))
	route("GET /webhooks/{id}", wrapAuth(h.getWebhook))
	route("PUT /webhooks/{id}", wrapAuth(h.updateWebhook))
	route("DELETE /webhooks/{id}", wrapAuth(h.deleteWebhook))
	route("POST /webhooks/{id}/ping", wrapAuth(h.pingWebhook))
	route("GET /webhooks/{id}/deliveries", wrapAuth(h.listWebhookDeliveries))
	route("GET /webhooks/{id}/deliveries/{deliveryId}", wrapAuth(h.getWebhookDelivery))

	// Custom field routes (RBAC enforced in service layer)
	route("GET /custom-fields", wrapAuth(h.listCustomFieldDefinitions))
	route("POST /custom-fields", wrapAuth(h.createCustomFieldDefinition))
	route("GET /custom-fields/types", wrapAuth(h.getCustomFieldTypes))
	route("GET /custom-fields/{id}", wrapAuth(h.getCustomFieldDefinition))
	route("PUT /custom-fields/{id}", wrapAuth(h.updateCustomFieldDefinition))
	route("DELETE /custom-fields/{id}", wrapAuth(h.deleteCustomFieldDefinition))

	// Circuit routes
	route("GET /circuits", wrapAuth(h.listCircuits))
	route("POST /circuits", wrapAuth(h.createCircuit))
	route("GET /circuits/{id}", wrapAuth(h.getCircuit))
	route("PUT /circuits/{id}", wrapAuth(h.updateCircuit))
	route("DELETE /circuits/{id}", wrapAuth(h.deleteCircuit))

	// NAT routes
	route("GET /nat", wrapAuth(h.listNATMappings))
	route("POST /nat", wrapAuth(h.createNATMapping))
	route("GET /nat/{id}", wrapAuth(h.getNATMapping))
	route("PUT /nat/{id}", wrapAuth(h.updateNATMapping))
	route("DELETE /nat/{id}", wrapAuth(h.deleteNATMapping))

	// DNS routes (RBAC enforced in service layer)
	if h.svc != nil && h.svc.DNS != nil {
		// Provider routes
		route("GET /dns/providers", wrapAuth(h.listDNSProviders))
		route("POST /dns/providers", wrapAuth(h.createDNSProvider))
		route("GET /dns/providers/{id}", wrapAuth(h.getDNSProvider))
		route("PUT /dns/providers/{id}", wrapAuth(h.updateDNSProvider))
		route("DELETE /dns/providers/{id}", wrapAuth(h.deleteDNSProvider))
		route("POST /dns/providers/{id}/test", wrapAuth(h.testDNSProvider))
		route("GET /dns/providers/{id}/zones", wrapAuth(h.listDNSProviderZones))
		// Zone routes
		route("GET /dns/zones", wrapAuth(h.listDNSZones))
		route("POST /dns/zones", wrapAuth(h.createDNSZone))
		route("GET /dns/zones/{id}", wrapAuth(h.getDNSZone))
		route("PUT /dns/zones/{id}", wrapAuth(h.updateDNSZone))
		route("DELETE /dns/zones/{id}", wrapAuth(h.deleteDNSZone))
		route("POST /dns/zones/{id}/sync", wrapAuth(h.syncDNSZone))
		route("POST /dns/zones/{id}/import", wrapAuth(h.importDNSZone))
		route("GET /dns/zones/{id}/records", wrapAuth(h.listDNSZoneRecords))
		// Record routes
		route("GET /dns/records/{id}", wrapAuth(h.getDNSRecord))
		route("PUT /dns/records/{id}", wrapAuth(h.updateDNSRecord))
		route("DELETE /dns/records/{id}", wrapAuth(h.deleteDNSRecord))
		route("POST /dns/records/{id}/link", wrapAuth(h.linkDNSRecord))
		route("POST /dns/records/{id}/promote", wrapAuth(h.promoteDNSRecord))
	}

	// Health check routes (no auth required)
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /readyz", h.readyz)

	// Metrics route (requires auth)
	mux.HandleFunc("GET /metrics", wrapAuth(h.metricsHandler))

	// Unmatched API requests get JSON errors instead of the UI's index.html.
	h.registerAPIFallback(mux)
}

// authenticated wraps handler with the body limit and session or API-key
// authentication, which puts the caller in the request context.
func (h *Handler) authenticated(handler http.HandlerFunc) http.HandlerFunc {
	handler = LimitBody(handler)
	if h.sessionManager != nil {
		return AuthMiddlewareWithSessions(h.store, h.sessionManager, handler)
	}
	return AuthMiddleware(h.store, handler)
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
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
	case errors.As(err, new(*service.PrivilegeError)):
		h.writeError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, service.ErrForbidden):
		h.writeError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden")
	case errors.Is(err, service.ErrLastAdmin):
		h.writeError(w, http.StatusConflict, "LAST_ADMIN", "This would leave no active admin")
	case errors.Is(err, service.ErrUnauthenticated):
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	case errors.Is(err, service.ErrAlreadyExists):
		h.writeError(w, http.StatusConflict, "ALREADY_EXISTS", err.Error())
	case errors.Is(err, discovery.ErrTooManyScans):
		h.writeError(w, http.StatusTooManyRequests, "TOO_MANY_SCANS", err.Error())
	case errors.Is(err, service.ErrIPNotAvailable):
		h.writeError(w, http.StatusConflict, "IP_NOT_AVAILABLE", "No IP addresses available")
	case errors.Is(err, service.ErrValidation):
		h.writeValidationErrors(w, toValidationErrors(err))
	case errors.Is(err, service.ErrSelfDelete):
		h.writeError(w, http.StatusBadRequest, "CANNOT_DELETE_SELF", err.Error())
	case errors.Is(err, service.ErrSystemRole):
		h.writeError(w, http.StatusBadRequest, "SYSTEM_ROLE", err.Error())
	case errors.Is(err, service.ErrDNSProviderTestFailed):
		// Upstream connectivity failure (SSRF block, timeout, refused, bad
		// credentials) — not a server bug. Surface the cause to the operator
		// in both the response and the server log (this path previously had
		// no logging at all, leaving failures invisible even at debug level).
		log.Error("DNS provider test failed", "error", err)
		h.writeError(w, http.StatusBadGateway, "PROVIDER_UNREACHABLE", err.Error())
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
	_ = json.NewEncoder(w).Encode(map[string]any{
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

// parsePagination reads limit/offset from query params and clamps to safe bounds.
func parsePagination(r *http.Request) model.Pagination {
	p := model.Pagination{
		Limit:  parseIntParam(r, "limit", model.DefaultPageSize),
		Offset: parseIntParam(r, "offset", 0),
	}
	p.Clamp()
	return p
}

// parseOptionalPagination returns the page asked for with ?limit= and ?offset=,
// or no page (every result) when neither is given.
func parseOptionalPagination(r *http.Request) model.Pagination {
	q := r.URL.Query()
	if q.Get("limit") == "" && q.Get("offset") == "" {
		return model.Pagination{}
	}
	return parsePagination(r)
}

// badRequest writes a standardized 400 error. Use for all request validation failures.
func (h *Handler) badRequest(w http.ResponseWriter, message string) {
	h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", message)
}

// invalidJSON writes a standardized 400 error for JSON decode failures.
func (h *Handler) invalidJSON(w http.ResponseWriter) {
	h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON")
}
