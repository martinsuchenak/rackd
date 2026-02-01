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
	"github.com/martinsuchenak/rackd/internal/discovery"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/mcp"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/internal/ui"
	"github.com/martinsuchenak/rackd/internal/worker"
)

// Feature interface for enterprise extension
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

// RunWithCustomRoutes starts the server with optional features and custom route registration
func RunWithCustomRoutes(cfg *config.Config, store storage.ExtendedStorage, registerRoutes func(mux *http.ServeMux), features ...Feature) error {
	if cfg.APIAuthToken == "" {
		log.Warn("API_AUTH_TOKEN not set - API is unauthenticated")
	}
	if cfg.MCPAuthToken == "" {
		log.Warn("MCP_AUTH_TOKEN not set - MCP endpoint is unauthenticated")
	}

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
	if cfg.APIAuthToken != "" {
		handler.RegisterRoutes(mux, api.WithAuth(cfg.APIAuthToken))
	} else {
		handler.RegisterRoutes(mux)
	}

	// MCP server
	mcpServer := mcp.NewServer(store, cfg.MCPAuthToken)
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

	// Health check
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      api.SecurityHeaders(mux),
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
