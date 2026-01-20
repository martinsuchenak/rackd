package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/martinsuchenak/rackd/internal/api"
	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/martinsuchenak/rackd/pkg/discovery"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/mcp"
	"github.com/martinsuchenak/rackd/pkg/registry"
	"github.com/martinsuchenak/rackd/internal/scanner"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/internal/ui"
	"github.com/martinsuchenak/rackd/internal/worker"
	"github.com/paularlott/cli"
)

// initializeDiscoveryFromRegistry attempts to initialize discovery features from the premium registry.
// Falls back to built-in implementations if premium features are not available.
// Returns (scanner, scheduler, handler, usePremium)
func initializeDiscoveryFromRegistry(
	cfg *config.Config,
	discoveryStore storage.DiscoveryStorage,
) (discovery.Scanner, *worker.Scheduler, *api.DiscoveryHandler, bool) {
	reg := registry.GetRegistry()

	var discoveryScanner discovery.Scanner
	var discoveryScheduler *worker.Scheduler
	usePremium := false

	// Try to get premium scanner from registry
	if scannerFactory, ok := reg.GetScannerProvider("discovery"); ok {
		scannerInterface, err := scannerFactory(map[string]interface{}{
			"storage": discoveryStore,
			"config":  cfg,
		})
		if err != nil {
			log.Warn("Failed to create premium scanner, falling back to built-in", "error", err)
		} else {
			var ok bool
			discoveryScanner, ok = scannerInterface.(discovery.Scanner)
			if !ok {
				log.Warn("Premium scanner does not implement discovery.Scanner interface, falling back to built-in")
			} else {
				log.Info("Using premium discovery scanner")
				usePremium = true
			}
		}
	}

	// Fall back to built-in scanner if premium wasn't loaded
	if discoveryScanner == nil {
		discoveryScanner = scanner.NewDiscoveryScanner(discoveryStore)
		log.Info("Using built-in discovery scanner")
	}

	// Create discovery handler
	discoveryHandler := api.NewDiscoveryHandler(discoveryStore, discoveryScanner)

	// Initialize scheduler if discovery is enabled
	if cfg.DiscoveryEnabled {
		log.Info("Discovery enabled, initializing scheduler",
			"interval", cfg.DiscoveryInterval,
			"max_concurrent", cfg.DiscoveryMaxConcurrent,
			"default_scan_type", cfg.DiscoveryDefaultScanType)

		// Try to get premium scheduler from registry
		if schedulerFactory, ok := reg.GetWorkerProvider("discovery-scheduler"); ok {
			schedulerInterface, err := schedulerFactory(map[string]interface{}{
				"storage": discoveryStore,
				"scanner": discoveryScanner,
				"config":  cfg,
			})
			if err != nil {
				log.Warn("Failed to create premium scheduler, falling back to built-in", "error", err)
			} else {
				var ok bool
				discoveryScheduler, ok = schedulerInterface.(*worker.Scheduler)
				if !ok {
					log.Warn("Premium scheduler is not *worker.Scheduler, falling back to built-in")
				} else {
					log.Info("Using premium discovery scheduler")
					usePremium = true
				}
			}
		}

		// Fall back to built-in scheduler if premium wasn't loaded
		if discoveryScheduler == nil {
			discoveryScheduler = worker.NewScheduler(discoveryStore, discoveryScanner)
			log.Info("Using built-in discovery scheduler")
		}

		// Start scheduler in background
		discoveryScheduler.Start()
		log.Info("Discovery scheduler started")
	} else {
		log.Info("Discovery disabled (scheduler not running). Manual scans via UI/API are still available.")
	}

	return discoveryScanner, discoveryScheduler, discoveryHandler, usePremium
}

// initializeEnterpriseRoutes registers enterprise-specific routes from the registry
func initializeEnterpriseRoutes(mux *http.ServeMux, reg *registry.Registry, store storage.Storage) {
	handlerFactory, ok := reg.GetAPIHandler("enterprise")
	if !ok {
		return
	}

	// Try to cast to PremiumStorage for enterprise features
	premiumStore, ok := store.(storage.PremiumStorage)
	if !ok {
		return
	}

	handlerInterface := handlerFactory(map[string]interface{}{
		"storage": premiumStore,
	})
	if handlerInterface == nil {
		return
	}

	// Check if handler implements RegisterRoutes method
	type routeRegisterer interface {
		RegisterRoutes(*http.ServeMux)
	}

	handler, ok := handlerInterface.(routeRegisterer)
	if !ok {
		return
	}

	handler.RegisterRoutes(mux)
	log.Info("Enterprise API routes registered")
}

// initializeEnterpriseAssets registers enterprise-specific asset handlers from the registry
func initializeEnterpriseAssets(mux *http.ServeMux, reg *registry.Registry) {
	assetHandler, exists := reg.GetFeature("enterprise-assets")
	if !exists {
		return
	}

	// The registry stores a function that takes *http.ServeMux
	handlerFunc, ok := assetHandler.(func(*http.ServeMux))
	if !ok {
		log.Warn("Enterprise assets found but has wrong type", "type", fmt.Sprintf("%T", assetHandler))
		return
	}

	handlerFunc(mux)
	log.Info("Enterprise asset handlers registered")
}

// ServerConfig holds configuration for running the server
type ServerConfig struct {
	Config             *config.Config
	Store              storage.Storage
	DiscoveryStore     storage.DiscoveryStorage
	DiscoveryHandler   *api.DiscoveryHandler
	DiscoveryScanner   discovery.Scanner
	DiscoveryScheduler *worker.Scheduler
	MCPServer          *mcp.Server
	APIHandler         *api.Handler
	CustomUIHandler    http.HandlerFunc // Optional: override default UI handler
}

// RunServer starts the Rackd server with the given configuration
func RunServer(cfg *ServerConfig) error {
	// Setup HTTP routes
	mux := http.NewServeMux()

	// API routes
	cfg.APIHandler.RegisterRoutes(mux)

	// Discovery API routes
	if cfg.DiscoveryHandler != nil {
		cfg.DiscoveryHandler.RegisterRoutes(mux)
	}

	// Enterprise routes (if registered)
	initializeEnterpriseRoutes(mux, registry.GetRegistry(), cfg.Store)

	// Enterprise assets (if registered)
	initializeEnterpriseAssets(mux, registry.GetRegistry())

	// MCP endpoint
	mux.HandleFunc("/mcp", cfg.MCPServer.GetHTTPHandler())

	// Serve web UI at root (handles all / and /assets/* requests)
	// Use custom UI handler if provided, otherwise use default
	uiHandler := cfg.CustomUIHandler
	if uiHandler == nil {
		uiHandler = ui.AssetHandler()
		log.Info("Using default UI handler")
	} else {
		log.Info("Using custom UI handler")
	}
	mux.Handle("/", uiHandler)

	// Apply middleware
	var handler http.Handler = mux
	if cfg.Config.IsAPIAuthEnabled() {
		handler = api.AuthMiddleware(cfg.Config.APIAuthToken, handler)
	}
	handler = api.SecurityHeadersMiddleware(handler)

	// Start server
	server := &http.Server{
		Addr:    cfg.Config.ListenAddr,
		Handler: handler,
	}

	// Handle shutdown gracefully
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Info("Shutting down server...")
		server.Close()
	}()

	// Log startup info
	log.Info("Starting Rackd server", "addr", cfg.Config.ListenAddr)
	log.Info("Web UI available", "url", "http://localhost"+cfg.Config.ListenAddr)
	log.Info("API available", "url", "http://localhost"+cfg.Config.ListenAddr+"/api/")
	log.Info("MCP available", "url", "http://localhost"+cfg.Config.ListenAddr+"/mcp")
	if cfg.Config.IsMCPEnabled() {
		log.Info("MCP authentication enabled")
	}
	if cfg.Config.IsAPIAuthEnabled() {
		log.Info("API authentication enabled")
	}
	cfg.MCPServer.LogStartup()

	// Start serving
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("Server error", "error", err)
		return err
	}

	log.Info("Server stopped")
	return nil
}

func Command() *cli.Command {
	return &cli.Command{
		Name:        "server",
		Usage:       "Start the Rackd server",
		Description: "Start the HTTP server with web UI, API, and MCP endpoints",
		Flags:       config.GetFlags(),
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := config.Load()

			log.Info("Configuration loaded", "data_dir", cfg.DataDir, "listen_addr", cfg.ListenAddr)

			// Initialize storage (SQLite only)
			store, err := storage.NewStorage(cfg.DataDir, "", "")
			if err != nil {
				log.Error("Failed to initialize storage", "error", err)
				return err
			}
			log.Info("Storage initialized", "backend", "SQLite", "path", cfg.DataDir)

			// Create API handler
			apiHandler := api.NewHandler(store)

			// Get discovery storage and create discovery handler
			discoveryStore, ok := store.(storage.DiscoveryStorage)
			var discoveryHandler *api.DiscoveryHandler
			var discoveryScheduler *worker.Scheduler
			var discoveryScanner discovery.Scanner

			if ok {
				log.Info("Discovery storage initialized")

				// Initialize discovery features from registry (with fallback to built-in)
				discoveryScanner, discoveryScheduler, discoveryHandler, _ = initializeDiscoveryFromRegistry(cfg, discoveryStore)

				// Defer stopping the scheduler if it was created
				if discoveryScheduler != nil {
					defer func() {
						log.Info("Stopping discovery scheduler...")
						discoveryScheduler.Stop()
						log.Info("Discovery scheduler stopped")
					}()
				}
			} else {
				log.Warn("Storage does not support discovery, discovery features will be unavailable")
			}

			// Create MCP server
			mcpServer := mcp.NewServer(store, cfg.MCPAuthToken)

			// Check for custom UI handler from registry (for enterprise)
			var customUIHandler http.HandlerFunc
			if uiHandler, exists := registry.GetRegistry().GetFeature("ui-handler"); exists {
				if handler, ok := uiHandler.(http.HandlerFunc); ok {
					customUIHandler = handler
					log.Info("Using custom UI handler from registry")
				}
			}

			// Build server config
			serverConfig := &ServerConfig{
				Config:             cfg,
				Store:              store,
				DiscoveryStore:     discoveryStore,
				DiscoveryHandler:   discoveryHandler,
				DiscoveryScanner:   discoveryScanner,
				DiscoveryScheduler: discoveryScheduler,
				MCPServer:          mcpServer,
				APIHandler:         apiHandler,
				CustomUIHandler:    customUIHandler,
			}

			return RunServer(serverConfig)
		},
	}
}