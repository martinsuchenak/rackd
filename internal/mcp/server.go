package mcp

import (
	"net/http"
	"strings"

	"github.com/paularlott/mcp"

	"github.com/martinsuchenak/rackd/internal/api"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type Server struct {
	mcpServer    *mcp.Server
	svc          *service.Services
	store        storage.ExtendedStorage
	requireAuth  bool
	oauthService *service.OAuthService
	oauthEnabled bool
}

func (s *Server) SetOAuthService(svc *service.OAuthService) {
	s.oauthService = svc
	s.oauthEnabled = svc != nil
}

func NewServer(services *service.Services, store storage.ExtendedStorage, requireAuth bool) *Server {
	s := &Server{
		mcpServer:   mcp.NewServer("rackd", "1.0.0"),
		svc:         services,
		store:       store,
		requireAuth: requireAuth,
	}
	s.mcpServer.SetInstructions(`rackd is a network infrastructure management system.
Use the native tools for common operations (search, device CRUD, network/datacenter lookup, IP allocation).
Use tool_search to discover additional tools for: circuits, NAT mappings, reservations, webhooks,
custom fields, discovery scans, conflict detection, DNS management, and audit logs.`)
	s.registerTools()
	return s
}

func (s *Server) Inner() *mcp.Server {
	return s.mcpServer
}

func (s *Server) registerTools() {
	s.registerSearchTools()
	s.registerDeviceTools()
	s.registerNetworkTools()
	s.registerCircuitTools()
	s.registerNATTools()
	s.registerReservationTools()
	s.registerWebhookTools()
	s.registerCustomFieldTools()
	s.registerDiscoveryTools()
	s.registerConflictTools()
	s.registerAuditTools()
	s.registerDNSTools()
}

func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request) {
	log.Debug("MCP request received", "remote_addr", r.RemoteAddr, "method", r.Method)

	// OPTIONS (CORS preflight) bypasses auth per CORS spec
	if r.Method == http.MethodOptions {
		s.mcpServer.HandleRequest(w, r)
		return
	}

	if s.requireAuth {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			if authHeader == "" {
				log.Debug("MCP auth failed: no Authorization header")
			} else {
				log.Debug("MCP auth failed: missing Bearer prefix")
			}
			s.writeUnauthorized(w)
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")

		var caller *service.Caller

		// Strategy 1: Try OAuth token validation (if OAuth enabled)
		if s.oauthService != nil {
			oauthToken, err := s.oauthService.ValidateAccessToken(token)
			if err == nil {
				caller, err = s.oauthService.ResolveCallerFromOAuthToken(oauthToken, r.RemoteAddr)
				if err != nil {
					log.Debug("MCP OAuth auth failed: could not resolve caller", "error", err)
					s.writeUnauthorized(w)
					return
				}
				log.Trace("MCP auth successful (OAuth)", "user_id", caller.UserID)
			}
		}

		// Strategy 2: Fall back to API key authentication using shared logic
		if caller == nil {
			var err error
			caller, err = api.AuthenticateAPIKey(r.Context(), s.store, token, r.RemoteAddr, "mcp")
			if err != nil {
				log.Debug("MCP auth failed", "error", err)
				s.writeUnauthorized(w)
				return
			}
		}

		r = r.WithContext(service.WithCaller(r.Context(), caller))
	} else {
		r = r.WithContext(service.SystemContext(r.Context(), "mcp"))
	}

	s.mcpServer.HandleRequest(w, r)
}

func (s *Server) writeUnauthorized(w http.ResponseWriter) {
	if s.oauthEnabled {
		w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="/.well-known/oauth-protected-resource"`)
	} else {
		w.Header().Set("WWW-Authenticate", `Bearer realm="rackd", error="invalid_token", error_description="Bearer token required"`)
	}
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
