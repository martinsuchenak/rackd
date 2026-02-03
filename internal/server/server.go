package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/martinsuchenak/rackd/internal/api"
	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/discovery"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/mcp"
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
	features ...Feature,
) error {
	mux := http.NewServeMux()

	scanner := discovery.NewScanner(store, cfg)
	scheduler := worker.NewScheduler(store, scanner, cfg)
	scheduler.Start()

	// Initialize advanced discovery service
	advancedDiscovery := discovery.NewAdvancedDiscoveryService(store, store, credStore, 30*time.Second)

	// Initialize scheduled scan worker
	scheduledWorker := worker.NewScheduledScanWorker(scheduledStore, profileStore, advancedDiscovery)
	if err := scheduledWorker.Start(); err != nil {
		log.Error("Failed to start scheduled scan worker", "error", err)
	}

	// API routes
	handler := api.NewHandler(store, scanner)
	handler.SetCredentialsStorage(credStore)
	handler.SetProfileStorage(profileStore)
	handler.SetScheduledScanStorage(scheduledStore)
	handler.RegisterRoutes(mux)

	// MCP server
	mcpServer := mcp.NewServer(store, false)
	mux.HandleFunc("POST /mcp", mcpServer.HandleRequest)

	// UI config with nav items for new features
	uiBuilder := api.NewUIConfigBuilder()
	uiBuilder.AddNavItem(api.NavItem{Label: "Credentials", Path: "/credentials", Icon: "key", Order: 50})
	uiBuilder.AddNavItem(api.NavItem{Label: "Scan Profiles", Path: "/scan-profiles", Icon: "cog", Order: 51})
	uiBuilder.AddNavItem(api.NavItem{Label: "Scheduled Scans", Path: "/scheduled-scans", Icon: "clock", Order: 52})

	// Register features
	for _, f := range features {
		log.Info("Registering feature", "name", f.Name())
		f.RegisterRoutes(mux)
		f.RegisterMCPTools(mcpServer)
		f.ConfigureUI(uiBuilder)
	}

	// UI config endpoint
	mux.HandleFunc("GET /api/config", uiBuilder.Handler())

	// Static UI
	ui.RegisterRoutes(mux)

	// Apply middleware chain
	var httpHandler http.Handler = mux
	if cfg.RateLimitEnabled {
		log.Info("Rate limiting enabled", "requests", cfg.RateLimitRequests, "window", cfg.RateLimitWindow)
		limiter := api.NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitWindow)
		httpHandler = api.RateLimitMiddleware(limiter)(httpHandler)
	}
	httpHandler = api.LoggingMiddleware(api.SecurityHeaders(httpHandler))

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

	scanner := discovery.NewScanner(store, cfg)
	scheduler := worker.NewScheduler(store, scanner, cfg)
	scheduler.Start()

	// API routes
	handler := api.NewHandler(store, scanner)
	handler.RegisterRoutes(mux)

	// MCP server
	mcpServer := mcp.NewServer(store, false)
	mux.HandleFunc("POST /mcp", mcpServer.HandleRequest)

	// UI config
	uiBuilder := api.NewUIConfigBuilder()

	// Register features
	for _, f := range features {
		log.Info("Registering feature", "name", f.Name())
		f.RegisterRoutes(mux)
		f.RegisterMCPTools(mcpServer)
		f.ConfigureUI(uiBuilder)
	}

	// UI config endpoint
	mux.HandleFunc("GET /api/config", uiBuilder.Handler())

	// Static UI
	ui.RegisterRoutes(mux)

	// Apply middleware chain
	var httpHandler http.Handler = mux
	if cfg.RateLimitEnabled {
		log.Info("Rate limiting enabled", "requests", cfg.RateLimitRequests, "window", cfg.RateLimitWindow)
		limiter := api.NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitWindow)
		httpHandler = api.RateLimitMiddleware(limiter)(httpHandler)
	}
	httpHandler = api.LoggingMiddleware(api.SecurityHeaders(httpHandler))

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
