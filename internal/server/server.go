package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/martinsuchenak/rackd/internal/api"
	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/discovery"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/mcp"
	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/internal/ui"
	"github.com/martinsuchenak/rackd/internal/worker"
)

// Feature interface for extensions
type Feature interface {
	Name() string
	RegisterRoutes(mux *http.ServeMux)
	RegisterMCPTools(mcpServer *mcp.Server)
	ConfigureUI(builder *api.UIConfigBuilder)
}

// Run starts the server with optional features
func Run(cfg *config.Config, store storage.ExtendedStorage, features ...Feature) error {
	return RunWithCustomRoutes(cfg, store, nil, features...)
}

// RunWithAdvancedFeatures starts the server with credentials, profiles, and scheduled scans features
func RunWithAdvancedFeatures(
	cfg *config.Config,
	store storage.ExtendedStorage,
	credStore credentials.Storage,
	profileStore storage.ProfileStorage,
	scheduledStore storage.ScheduledScanStorage,
	encryptionKey []byte,
	features ...Feature,
) error {
	mux := http.NewServeMux()

	// Initialize session manager
	sessionManager := auth.NewSessionManager(cfg.SessionTTL)
	defer sessionManager.Stop()

	// Bootstrap initial admin user
	if err := storage.BootstrapInitialAdmin(store, cfg, sessionManager); err != nil {
		return fmt.Errorf("failed to bootstrap initial admin: %w", err)
	}

	scanner := discovery.NewUnifiedScanner(store, store, credStore, 30*time.Second)
	scheduler := worker.NewScheduler(store, scanner, cfg)
	scheduler.Start()

	// Initialize scheduled scan worker (unified scanner supports both basic and advanced scans)
	scheduledWorker := worker.NewScheduledScanWorker(scheduledStore, profileStore, scanner)
	if err := scheduledWorker.Start(); err != nil {
		log.Error("Failed to start scheduled scan worker", "error", err)
	}

	// Create services registry
	services := service.NewServices(store, sessionManager, scanner)

	// Set optional services with their storage types
	services.SetCredentialsStorage(credStore)
	services.SetProfileStorage(profileStore)
	services.SetScheduledScanStorage(scheduledStore)

	// DNS service setup (requires encryption for provider credentials)
	if encryptionKey != nil {
		encryptor, err := credentials.NewEncryptor(encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to create encryptor for DNS service: %w", err)
		}
		services.SetDNSService(store, encryptor)

		// Initialize and start DNS sync worker if interval is configured
		if cfg.DNSSyncInterval > 0 {
			dnsWorker := worker.NewDNSWorker(services.DNS, cfg)
			dnsWorker.Start()
			defer dnsWorker.Stop()
		} else {
			log.Info("DNS sync disabled (interval set to 0)")
		}
	}

	// OAuth setup (conditional) - must be before RegisterRoutes
	if cfg.MCPOAuthEnabled {
		oauthService := service.NewOAuthService(store, sessionManager, cfg.MCPOAuthIssuerURL)
		oauthService.SetTokenTTLs(cfg.MCPOAuthAccessTokenTTL, cfg.MCPOAuthRefreshTokenTTL)
		oauthService.StartCleanup()
		services.OAuth = oauthService
		log.Info("MCP OAuth 2.1 enabled", "issuer", cfg.MCPOAuthIssuerURL)
	}

	// API routes
	handler := api.NewHandler(store, scanner)
	handler.SetSessionManager(sessionManager)
	handler.SetCredentialsStorage(credStore)
	handler.SetProfileStorage(profileStore)
	handler.SetScheduledScanStorage(scheduledStore)
	handler.SetLoginRateLimiter(api.NewRateLimiter(cfg.LoginRateLimitRequests, cfg.LoginRateLimitWindow))
	handler.SetCookieConfig(cfg.CookieSecure, cfg.SessionTTL)
	handler.SetTrustProxy(cfg.TrustProxy)
	handler.SetServices(services)
	log.Info("Login rate limiting enabled", "requests", cfg.LoginRateLimitRequests, "window", cfg.LoginRateLimitWindow)
	handler.RegisterRoutes(mux)

	// MCP server (require auth when OAuth is enabled or session manager is configured)
	mcpRequireAuth := cfg.MCPOAuthEnabled || sessionManager != nil
	mcpServer := mcp.NewServer(services, store, mcpRequireAuth)
	if services.OAuth != nil {
		mcpServer.SetOAuthService(services.OAuth)
	}
	mux.HandleFunc("POST /mcp", mcpServer.HandleRequest)

	// UI config with nav items for new features
	uiBuilder := api.NewUIConfigBuilder()
	uiBuilder.AddNavItem(api.NavItem{Label: "Users", Path: "/users", Icon: "user", Order: 15, RequiredPermissions: []api.PermissionCheck{{Resource: "users", Action: "list"}}})
	uiBuilder.AddNavItem(api.NavItem{Label: "Roles", Path: "/roles", Icon: "shield", Order: 16, RequiredPermissions: []api.PermissionCheck{{Resource: "roles", Action: "list"}}})
	uiBuilder.AddNavItem(api.NavItem{Label: "Credentials", Path: "/credentials", Icon: "key", Order: 50})
	uiBuilder.AddNavItem(api.NavItem{Label: "Scan Profiles", Path: "/scan-profiles", Icon: "cog", Order: 51})
	uiBuilder.AddNavItem(api.NavItem{Label: "Scheduled Scans", Path: "/scheduled-scans", Icon: "clock", Order: 52})
	uiBuilder.AddNavItem(api.NavItem{Label: "Webhooks", Path: "/webhooks", Icon: "zap", Order: 53, RequiredPermissions: []api.PermissionCheck{{Resource: "webhook", Action: "list"}}})
	uiBuilder.AddNavItem(api.NavItem{Label: "Custom Fields", Path: "/custom-fields", Icon: "tag", Order: 54, RequiredPermissions: []api.PermissionCheck{{Resource: "custom-fields", Action: "list"}}})

	// Add DNS nav items if DNS service is available
	if services.DNS != nil {
		uiBuilder.AddNavItem(api.NavItem{Label: "DNS Providers", Path: "/dns/providers", Icon: "server", Order: 57})
		uiBuilder.AddNavItem(api.NavItem{Label: "DNS Zones", Path: "/dns/zones", Icon: "globe", Order: 58})
	}

	// Register features
	for _, f := range features {
		log.Info("Registering feature", "name", f.Name())
		f.RegisterRoutes(mux)
		f.RegisterMCPTools(mcpServer)
		f.ConfigureUI(uiBuilder)
	}

	// UI config endpoint
	mux.HandleFunc("GET /api/config", uiBuilder.HandlerWithSession(sessionManager, store))

	// Static UI
	ui.RegisterRoutes(mux)

	// Apply middleware chain
	var httpHandler http.Handler = mux
	if cfg.RateLimitEnabled {
		log.Info("Rate limiting enabled", "requests", cfg.RateLimitRequests, "window", cfg.RateLimitWindow)
		limiter := api.NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitWindow)
		httpHandler = api.RateLimitMiddleware(limiter, cfg.TrustProxy)(httpHandler)
	}
	httpHandler = api.LoggingMiddleware(api.SecurityHeaders(httpHandler))
	// Storage-level audit logging is always active for all entry points (API, MCP, CLI, scheduler)
	if cfg.AuditEnabled {
		log.Info("Audit logging enabled (storage-level)", "retention_days", cfg.AuditRetentionDays)
	}

	server := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      httpHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	errCh := make(chan error, 1)
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Info("Shutting down...")
		scheduler.Stop()
		scheduledWorker.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		errCh <- server.Shutdown(ctx)
	}()

	log.Info("Starting server", "addr", cfg.ListenAddr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return <-errCh
}

// RunWithCustomRoutes starts the server with optional features and custom route registration
func RunWithCustomRoutes(cfg *config.Config, store storage.ExtendedStorage, registerRoutes func(mux *http.ServeMux), features ...Feature) error {
	mux := http.NewServeMux()

	// Register custom routes if provided
	if registerRoutes != nil {
		registerRoutes(mux)
	}

	// Initialize session manager
	sessionManager := auth.NewSessionManager(cfg.SessionTTL)
	defer sessionManager.Stop()

	// Bootstrap initial admin user
	if err := storage.BootstrapInitialAdmin(store, cfg, sessionManager); err != nil {
		return fmt.Errorf("failed to bootstrap initial admin: %w", err)
	}

	scanner := discovery.NewUnifiedScanner(store, store, nil, 30*time.Second)
	scheduler := worker.NewScheduler(store, scanner, cfg)
	scheduler.Start()

	// Create services registry
	services := service.NewServices(store, sessionManager, scanner)

	// OAuth setup (conditional) - must be before RegisterRoutes
	if cfg.MCPOAuthEnabled {
		oauthService := service.NewOAuthService(store, sessionManager, cfg.MCPOAuthIssuerURL)
		oauthService.SetTokenTTLs(cfg.MCPOAuthAccessTokenTTL, cfg.MCPOAuthRefreshTokenTTL)
		oauthService.StartCleanup()
		services.OAuth = oauthService
		log.Info("MCP OAuth 2.1 enabled", "issuer", cfg.MCPOAuthIssuerURL)
	}

	// API routes
	handler := api.NewHandler(store, scanner)
	handler.SetSessionManager(sessionManager)
	handler.SetLoginRateLimiter(api.NewRateLimiter(cfg.LoginRateLimitRequests, cfg.LoginRateLimitWindow))
	handler.SetCookieConfig(cfg.CookieSecure, cfg.SessionTTL)
	handler.SetTrustProxy(cfg.TrustProxy)
	handler.SetServices(services)
	handler.RegisterRoutes(mux)

	// MCP server (require auth when OAuth is enabled or session manager is configured)
	mcpRequireAuth := cfg.MCPOAuthEnabled || sessionManager != nil
	mcpServer := mcp.NewServer(services, store, mcpRequireAuth)
	if services.OAuth != nil {
		mcpServer.SetOAuthService(services.OAuth)
	}
	mux.HandleFunc("POST /mcp", mcpServer.HandleRequest)

	// UI config
	uiBuilder := api.NewUIConfigBuilder()
	uiBuilder.AddNavItem(api.NavItem{Label: "Users", Path: "/users", Icon: "user", Order: 15, RequiredPermissions: []api.PermissionCheck{{Resource: "users", Action: "list"}}})
	uiBuilder.AddNavItem(api.NavItem{Label: "Webhooks", Path: "/webhooks", Icon: "zap", Order: 53, RequiredPermissions: []api.PermissionCheck{{Resource: "webhook", Action: "list"}}})
	uiBuilder.AddNavItem(api.NavItem{Label: "Custom Fields", Path: "/custom-fields", Icon: "tag", Order: 54, RequiredPermissions: []api.PermissionCheck{{Resource: "custom-fields", Action: "list"}}})

	// Register features
	for _, f := range features {
		log.Info("Registering feature", "name", f.Name())
		f.RegisterRoutes(mux)
		f.RegisterMCPTools(mcpServer)
		f.ConfigureUI(uiBuilder)
	}

	// UI config endpoint
	mux.HandleFunc("GET /api/config", uiBuilder.HandlerWithSession(sessionManager, store))

	// Static UI
	ui.RegisterRoutes(mux)

	// Apply middleware chain
	var httpHandler http.Handler = mux
	if cfg.RateLimitEnabled {
		log.Info("Rate limiting enabled", "requests", cfg.RateLimitRequests, "window", cfg.RateLimitWindow)
		limiter := api.NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitWindow)
		httpHandler = api.RateLimitMiddleware(limiter, cfg.TrustProxy)(httpHandler)
	}
	httpHandler = api.LoggingMiddleware(api.SecurityHeaders(httpHandler))
	// Storage-level audit logging is always active for all entry points (API, MCP, CLI, scheduler)
	if cfg.AuditEnabled {
		log.Info("Audit logging enabled (storage-level)", "retention_days", cfg.AuditRetentionDays)
	}

	server := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      httpHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	errCh := make(chan error, 1)
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Info("Shutting down...")
		scheduler.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		errCh <- server.Shutdown(ctx)
	}()

	log.Info("Starting server", "addr", cfg.ListenAddr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return <-errCh
}
