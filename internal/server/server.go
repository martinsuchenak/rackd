package server

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/martinsuchenak/rackd/internal/api"
	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/discovery"
	"github.com/martinsuchenak/rackd/internal/extensions"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/mcp"
	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/internal/ui"
	"github.com/martinsuchenak/rackd/internal/webhook"
	"github.com/martinsuchenak/rackd/internal/worker"
	"github.com/martinsuchenak/rackd/pkg/extension"
	"github.com/redis/go-redis/v9"
)

// Options configures Run.
type Options struct {
	Config *config.Config
	Store  storage.ExtendedStorage
	// Credentials, Profiles and ScheduledScans enable credential-based and
	// scheduled discovery; they and EncryptionKey are nil without an
	// ENCRYPTION_KEY. EncryptionKey also enables DNS and encrypts webhook
	// secrets at rest.
	Credentials    credentials.Storage
	Profiles       storage.ProfileStorage
	ScheduledScans storage.ScheduledScanStorage
	EncryptionKey  []byte
	// Features are the extensions to run.
	Features []extension.Feature
	// Edition is reported in /api/v1/config; empty means "oss".
	Edition string
	// UI replaces the built-in web UI; nil serves the built-in one. It must
	// hold index.html at its root.
	UI fs.FS
}

// Run starts the server and blocks until ctx is cancelled, then shuts down:
// it finishes in-flight requests, stops extensions and workers, and returns.
// The caller owns Store and closes it afterwards.
func Run(ctx context.Context, opts Options) error {
	cfg, store := opts.Config, opts.Store
	mux := http.NewServeMux()

	sessionStore, err := newSessionStore(cfg, store)
	if err != nil {
		return err
	}
	sessionManager := auth.NewSessionManager(cfg.SessionTTL, sessionStore)
	defer sessionManager.Stop()

	// Bootstrap initial admin user
	if err := storage.BootstrapInitialAdmin(store, cfg, sessionManager); err != nil {
		return fmt.Errorf("failed to bootstrap initial admin: %w", err)
	}

	scanner := discovery.NewUnifiedScanner(store, opts.Credentials, 30*time.Second, cfg.DiscoverySNMPv2cEnabled)
	scheduler := worker.NewScheduler(store, scanner, cfg)
	scheduler.Start()
	defer scheduler.Stop()

	if opts.ScheduledScans != nil && opts.Profiles != nil {
		scheduledWorker := worker.NewScheduledScanWorker(opts.ScheduledScans, opts.Profiles, scanner)
		if err := scheduledWorker.Start(); err != nil {
			log.Error("Failed to start scheduled scan worker", "error", err)
		}
		defer scheduledWorker.Stop()
	}

	services := service.NewServices(store, sessionManager, scanner)
	if opts.Credentials != nil {
		services.SetCredentialsStorage(opts.Credentials)
	}
	if opts.Profiles != nil {
		services.SetProfileStorage(opts.Profiles)
	}
	if opts.ScheduledScans != nil {
		services.SetScheduledScanStorage(opts.ScheduledScans)
	}

	// DNS service setup (requires encryption for provider credentials)
	if opts.EncryptionKey != nil {
		encryptor, err := credentials.NewEncryptor(opts.EncryptionKey)
		if err != nil {
			return fmt.Errorf("failed to create encryptor for DNS service: %w", err)
		}
		services.SetDNSService(store, encryptor)

		// Encrypt webhook signing secrets at rest with the same key
		if sqliteStore, ok := store.(*storage.SQLiteStorage); ok {
			sqliteStore.SetWebhookEncryptor(encryptor)
		}

		if cfg.DNSSyncInterval > 0 {
			dnsWorker := worker.NewDNSWorker(services.DNS, cfg)
			dnsWorker.Start()
			defer dnsWorker.Stop()
		} else {
			log.Info("DNS sync disabled (interval set to 0)")
		}
	}

	// Webhook delivery: subscribed webhooks receive the events services publish.
	// It starts after the webhook secret encryptor is set, and is deferred
	// before the extensions so it stops after them.
	webhookWorker := webhook.NewWorker(store, webhook.DefaultDeliveryConfig())
	webhookWorker.Start()
	defer webhookWorker.Stop()

	// Utilization snapshots feed the dashboard's trend chart.
	if cfg.SnapshotInterval > 0 {
		snapshotWorker := worker.NewSnapshotWorker(store, cfg)
		snapshotWorker.Start()
		defer snapshotWorker.Stop()
	} else {
		log.Info("Utilization snapshots disabled (interval set to 0)")
	}

	// OAuth setup (conditional) - must be before RegisterRoutes
	if cfg.MCPOAuthEnabled {
		oauthService := service.NewOAuthService(store, sessionManager, cfg.MCPOAuthIssuerURL)
		oauthService.SetTokenTTLs(cfg.MCPOAuthAccessTokenTTL, cfg.MCPOAuthRefreshTokenTTL)
		oauthService.StartCleanup()
		services.OAuth = oauthService
		log.Info("MCP OAuth 2.1 enabled", "issuer", cfg.MCPOAuthIssuerURL)
	}

	site, err := ui.New(opts.UI)
	if err != nil {
		return err
	}

	// Extensions: migrations, permissions and event types before anything uses them
	runtime, err := newRuntime(ctx, opts.Features, store, services)
	if err != nil {
		return err
	}

	// API routes
	log.Info("Login rate limiting enabled", "requests", cfg.LoginRateLimitRequests, "window", cfg.LoginRateLimitWindow)
	proxyResolver := api.NewTrustedProxyResolver(cfg.TrustProxy, cfg.TrustedProxies)
	handler := api.NewHandler(store, scanner,
		api.WithSessionManager(sessionManager),
		api.WithCredentialsStorage(opts.Credentials),
		api.WithProfileStorage(opts.Profiles),
		api.WithScheduledScanStorage(opts.ScheduledScans),
		api.WithLoginRateLimiter(api.NewRateLimiter(cfg.LoginRateLimitRequests, cfg.LoginRateLimitWindow)),
		api.WithCookieConfig(cfg.CookieSecure, cfg.SessionTTL),
		api.WithProxyResolver(proxyResolver),
		api.WithServices(services),
		api.WithIndexHTML(site.IndexHTML()),
	)
	handler.RegisterRoutes(mux)

	// MCP server (require auth when OAuth is enabled or session manager is configured)
	mcpRequireAuth := cfg.MCPOAuthEnabled || sessionManager != nil
	mcpServer := mcp.NewServer(services, store, mcpRequireAuth)
	if services.OAuth != nil {
		mcpServer.SetOAuthService(services.OAuth)
	}
	if err := runtime.RegisterTools(mcpServer); err != nil {
		return err
	}
	// The MCP library decodes the whole body; cap it like every API route.
	mcpHandler := api.LimitBody(mcpServer.HandleRequest)
	mux.Handle("POST /mcp", mcpHandler)
	mux.Handle("OPTIONS /mcp", mcpHandler)
	mux.Handle("DELETE /mcp", mcpHandler)

	// UI config endpoint
	uiBuilder := api.NewUIConfigBuilder()
	if opts.Edition != "" {
		uiBuilder.SetEdition(opts.Edition)
	}
	runtime.ConfigureUI(uiBuilder)
	api.HandleAPI(mux, "GET /config", uiBuilder.HandlerWithSession(sessionManager, store))

	// Extension routes go last, so a clash with any core route is reported as
	// an error rather than a ServeMux panic in core's registration.
	if err := runtime.RegisterRoutes(handler, mux); err != nil {
		return err
	}

	// Static UI
	runtime.RegisterUIAssets(mux)
	site.RegisterRoutes(mux)

	httpHandler := buildHandler(mux, cfg, proxyResolver)
	if cfg.AuditEnabled {
		log.Info("Audit logging enabled (storage-level)", "retention_days", cfg.AuditRetentionDays)
	}

	if err := runtime.Start(); err != nil {
		return err
	}
	defer runtime.Stop(10 * time.Second)

	server := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      httpHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: writeTimeout(cfg.RequestTimeout),
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("Starting server", "addr", cfg.ListenAddr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	log.Info("Shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

// newRuntime prepares the extensions: their migrations, permissions and event
// types.
func newRuntime(ctx context.Context, features []extension.Feature, store storage.ExtendedStorage, services *service.Services) (*extensions.Runtime, error) {
	extStore, ok := store.(extensions.Store)
	if !ok && len(features) > 0 {
		return nil, fmt.Errorf("extensions need SQLite storage")
	}
	runtime, err := extensions.New(features, extStore, services)
	if err != nil {
		return nil, err
	}
	if err := runtime.Prepare(ctx); err != nil {
		return nil, err
	}
	return runtime, nil
}

func newSessionStore(cfg *config.Config, store storage.ExtendedStorage) (auth.SessionStore, error) {
	switch cfg.SessionStoreType {
	case "valkey", "redis":
		opt, err := redis.ParseURL(cfg.ValkeyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid Valkey URL: %w", err)
		}
		valkeyClient := redis.NewClient(opt)
		if err := valkeyClient.Ping(context.Background()).Err(); err != nil {
			return nil, fmt.Errorf("failed to connect to Valkey session store: %w", err)
		}
		log.Info("Using Valkey/Redis session store", "addr", opt.Addr, "db", opt.DB)
		return storage.NewValkeySessionStore(valkeyClient), nil

	case "sqlite":
		if sqliteStore, ok := store.(*storage.SQLiteStorage); ok {
			log.Info("Using SQLite Database session store")
			return storage.NewSQLiteSessionStore(sqliteStore.DB()), nil
		}
		log.Warn("SQLite session store requested but underlying store is not SQLite")

	default:
		log.Warn("Unknown session store type or memory specified, falling back to memory", "type", cfg.SessionStoreType)
	}
	return nil, nil
}

// buildHandler wraps the router in the middleware chain every server uses.
func buildHandler(mux *http.ServeMux, cfg *config.Config, proxyResolver *api.TrustedProxyResolver) http.Handler {
	handler := api.RecordRoutePattern(mux)
	if cfg.RateLimitEnabled {
		log.Info("Rate limiting enabled", "requests", cfg.RateLimitRequests, "window", cfg.RateLimitWindow)
		limiter := api.NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitWindow)
		handler = api.RateLimitMiddleware(limiter, proxyResolver)(handler)
	}
	handler = api.LoggingMiddleware(api.SecurityHeaders(handler))

	// Enforce request timeout (M-10)
	if cfg.RequestTimeout > 0 {
		handler = http.TimeoutHandler(handler, cfg.RequestTimeout, `{"error": "Request timeout"}`)
	}
	return handler
}

// writeTimeout returns the HTTP server write timeout. It must outlast the
// request timeout: once the write deadline passes the connection is closed, so
// http.TimeoutHandler could no longer send its timeout response and slow
// requests would get an empty reply instead of an error.
func writeTimeout(requestTimeout time.Duration) time.Duration {
	const minWriteTimeout = 15 * time.Second
	return max(minWriteTimeout, requestTimeout+5*time.Second)
}
