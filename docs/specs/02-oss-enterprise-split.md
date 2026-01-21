# OSS/Enterprise Two-Repository Architecture

This document describes the separation between the open-source core and the enterprise edition.

## Repository Structure

```
┌─────────────────────────────────────────────────────────────────┐
│         github.com/martinsuchenak/rackd-enterprise               │
│              (Enterprise Wrapper Repository)                     │
│                                                                  │
│  • Depends on: github.com/martinsuchenak/rackd                  │
│  • Provides: Postgres storage, SSO, RBAC, Audit, Monitoring     │
│  • Cannot be imported by OSS core (no circular deps)            │
└────────────────────────────┬────────────────────────────────────┘
                             │ imports
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│           github.com/martinsuchenak/rackd (OSS Core)            │
│                                                                  │
│  • Fully functional standalone                                  │
│  • SQLite storage included                                      │
│  • All interfaces defined here                                  │
│  • Must NOT import enterprise package                           │
└─────────────────────────────────────────────────────────────────┘
```

## Dependency Rules

| Rule | Description |
|------|-------------|
| **OSS Standalone** | OSS repo must work independently with no Enterprise dependencies |
| **Interface Forward** | OSS defines all interfaces that Enterprise implements |
| **No Reverse Import** | OSS must never import anything from Enterprise repo |
| **Feature Injection** | Enterprise features are injected via Feature interface at startup |
| **Storage Abstraction** | Both SQLite and Postgres implement same storage interfaces |

## Enterprise Integration Pattern

### OSS Server Entry Point

```go
// ===== OSS REPO: internal/server/server.go =====
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
    "github.com/martinsuchenak/rackd/internal/worker"
)

// Feature interface for Enterprise extension
type Feature interface {
    Name() string
    RegisterRoutes(mux *http.ServeMux)
    RegisterMCPTools(mcpServer interface{})
}

// Run is the main server entry point - accepts optional Enterprise features
func Run(cfg *config.Config, store storage.ExtendedStorage, features ...Feature) error {
    mux := http.NewServeMux()

    // Register OSS core routes with auth middleware
    apiHandler := api.NewHandler(store)
    if cfg.APIAuthToken != "" {
        apiHandler.RegisterRoutes(mux, api.WithAuth(cfg.APIAuthToken))
    } else {
        apiHandler.RegisterRoutes(mux)
    }

    // Setup discovery (if enabled)
    var scheduler *worker.Scheduler
    if cfg.DiscoveryEnabled {
        scanner := discovery.NewScanner(store, cfg)
        scheduler = worker.NewScheduler(store, scanner, cfg)
        scheduler.Start()

        // Register discovery routes
        discoveryHandler := api.NewDiscoveryHandler(store, scanner)
        discoveryHandler.RegisterRoutes(mux)
    }

    // Setup MCP server
    mcpServer := mcp.NewServer(store, cfg.MCPAuthToken)
    mux.HandleFunc("POST /mcp", mcpServer.HandleRequest)

    // Register Enterprise features (if any)
    for _, f := range features {
        log.Info("Registering feature", "name", f.Name())
        f.RegisterRoutes(mux)
        f.RegisterMCPTools(mcpServer.Inner())
    }

    // Serve embedded Web UI
    ui.RegisterRoutes(mux)

    // Create HTTP server
    server := &http.Server{
        Addr:         cfg.ListenAddr,
        Handler:      api.SecurityHeaders(mux),
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Graceful shutdown
    go func() {
        sigChan := make(chan os.Signal, 1)
        signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
        <-sigChan

        log.Info("Shutting down server...")

        // Stop scheduler first
        if scheduler != nil {
            scheduler.Stop()
        }

        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        if err := server.Shutdown(ctx); err != nil {
            log.Error("Server shutdown error", "error", err)
        }
    }()

    log.Info("Starting server", "addr", cfg.ListenAddr)
    return server.ListenAndServe()
}
```

### Enterprise Server Entry Point

```go
// ===== ENTERPRISE REPO: cmd/server/main.go =====
package main

import (
    "github.com/martinsuchenak/rackd/internal/config"
    "github.com/martinsuchenak/rackd/internal/server"
    "github.com/martinsuchenak/rackd/internal/storage"

    // Enterprise imports - OSS does NOT import these
    "github.com/martinsuchenak/rackd-enterprise/internal/storage/postgres"
    "github.com/martinsuchenak/rackd-enterprise/internal/features/sso"
    "github.com/martinsuchenak/rackd-enterprise/internal/features/rbac"
    "github.com/martinsuchenak/rackd-enterprise/internal/features/audit"
)

func main() {
    cfg := config.Load()

    // Start with OSS SQLite storage
    var store storage.ExtendedStorage
    store, _ = storage.NewExtendedStorage(cfg.DataDir, "sqlite", "")

    // Swap to Postgres if configured (Enterprise)
    if cfg.PostgresURL != "" {
        store = postgres.NewStorage(cfg.PostgresURL)
    }

    // Collect Enterprise features
    features := []server.Feature{}

    if cfg.SSOEnabled {
        features = append(features, sso.NewFeature(cfg))
    }

    if cfg.RBACEnabled {
        features = append(features, rbac.NewFeature(cfg))
    }

    if cfg.AuditEnabled {
        features = append(features, audit.NewFeature(cfg))
    }

    // Run OSS server with Enterprise features injected
    server.Run(cfg, store, features...)
}
```
