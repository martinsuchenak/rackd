# API Layer

This document covers the HTTP API handlers, middleware, and MCP server implementation.

## API Handlers

**File**: `internal/api/handlers.go`

```go
package api

import (
    "encoding/json"
    "net/http"

    "github.com/martinsuchenak/rackd/internal/log"
    "github.com/martinsuchenak/rackd/internal/storage"
)

type Handler struct {
    storage storage.ExtendedStorage
}

func NewHandler(s storage.ExtendedStorage) *Handler {
    return &Handler{storage: s}
}

type HandlerOption func(*handlerConfig)

type handlerConfig struct {
    authToken string
}

func WithAuth(token string) HandlerOption {
    return func(c *handlerConfig) {
        c.authToken = token
    }
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, opts ...HandlerOption) {
    cfg := &handlerConfig{}
    for _, opt := range opts {
        opt(cfg)
    }

    // Apply auth middleware if token is set
    wrap := func(handler http.HandlerFunc) http.HandlerFunc {
        if cfg.authToken != "" {
            return AuthMiddleware(cfg.authToken, handler)
        }
        return handler
    }

    // Datacenter routes
    mux.HandleFunc("GET /api/datacenters", wrap(h.listDatacenters))
    mux.HandleFunc("POST /api/datacenters", wrap(h.createDatacenter))
    mux.HandleFunc("GET /api/datacenters/{id}", wrap(h.getDatacenter))
    mux.HandleFunc("PUT /api/datacenters/{id}", wrap(h.updateDatacenter))
    mux.HandleFunc("DELETE /api/datacenters/{id}", wrap(h.deleteDatacenter))
    mux.HandleFunc("GET /api/datacenters/{id}/devices", wrap(h.getDatacenterDevices))

    // Network routes
    mux.HandleFunc("GET /api/networks", wrap(h.listNetworks))
    mux.HandleFunc("POST /api/networks", wrap(h.createNetwork))
    mux.HandleFunc("GET /api/networks/{id}", wrap(h.getNetwork))
    mux.HandleFunc("PUT /api/networks/{id}", wrap(h.updateNetwork))
    mux.HandleFunc("DELETE /api/networks/{id}", wrap(h.deleteNetwork))
    mux.HandleFunc("GET /api/networks/{id}/devices", wrap(h.getNetworkDevices))
    mux.HandleFunc("GET /api/networks/{id}/utilization", wrap(h.getNetworkUtilization))
    mux.HandleFunc("GET /api/networks/{id}/pools", wrap(h.listNetworkPools))
    mux.HandleFunc("POST /api/networks/{id}/pools", wrap(h.createNetworkPool))

    // Network pool routes
    mux.HandleFunc("GET /api/pools/{id}", wrap(h.getNetworkPool))
    mux.HandleFunc("PUT /api/pools/{id}", wrap(h.updateNetworkPool))
    mux.HandleFunc("DELETE /api/pools/{id}", wrap(h.deleteNetworkPool))
    mux.HandleFunc("GET /api/pools/{id}/next-ip", wrap(h.getNextIP))
    mux.HandleFunc("GET /api/pools/{id}/heatmap", wrap(h.getPoolHeatmap))

    // Device routes
    mux.HandleFunc("GET /api/devices", wrap(h.listDevices))
    mux.HandleFunc("POST /api/devices", wrap(h.createDevice))
    mux.HandleFunc("GET /api/devices/{id}", wrap(h.getDevice))
    mux.HandleFunc("PUT /api/devices/{id}", wrap(h.updateDevice))
    mux.HandleFunc("DELETE /api/devices/{id}", wrap(h.deleteDevice))
    mux.HandleFunc("GET /api/devices/search", wrap(h.searchDevices))

    // Relationship routes
    mux.HandleFunc("POST /api/devices/{id}/relationships", wrap(h.addRelationship))
    mux.HandleFunc("GET /api/devices/{id}/relationships", wrap(h.getRelationships))
    mux.HandleFunc("GET /api/devices/{id}/related", wrap(h.getRelatedDevices))
    mux.HandleFunc("DELETE /api/devices/{id}/relationships/{child_id}/{type}", wrap(h.removeRelationship))
}

// ===== Device Handler Implementations =====

func (h *Handler) createDevice(w http.ResponseWriter, r *http.Request) {
    var device model.Device
    if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
        h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
        return
    }

    // Validate required fields
    if device.Name == "" {
        h.writeError(w, http.StatusBadRequest, "MISSING_REQUIRED_FIELD", "Device name is required").WithDetails("field", "name")
        return
    }

    // Create device
    if err := h.storage.CreateDevice(&device); err != nil {
        if errors.Is(err, storage.ErrDeviceNotFound) {
            h.writeError(w, http.StatusNotFound, "DATACENTER_NOT_FOUND", "Referenced datacenter not found")
            return
        }
        h.internalError(w, err)
        return
    }

    h.writeJSON(w, http.StatusCreated, device)
}

func (h *Handler) getDevice(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    device, err := h.storage.GetDevice(id)
    if err != nil {
        if errors.Is(err, storage.ErrDeviceNotFound) {
            h.writeError(w, http.StatusNotFound, "DEVICE_NOT_FOUND", "Device not found")
            return
        }
        h.internalError(w, err)
        return
    }

    h.writeJSON(w, http.StatusOK, device)
}

func (h *Handler) listDevices(w http.ResponseWriter, r *http.Request) {
    // Parse query parameters
    filter := &model.DeviceFilter{
        Tags:         parseArrayParam(r, "tags"),
        DatacenterID: r.URL.Query().Get("datacenter_id"),
        NetworkID:    r.URL.Query().Get("network_id"),
    }
    limit := parseIntParam(r, "limit", 100)

    devices, err := h.storage.ListDevices(filter)
    if err != nil {
        h.internalError(w, err)
        return
    }

    // Apply limit
    if len(devices) > limit {
        devices = devices[:limit]
    }

    h.writeJSON(w, http.StatusOK, devices)
}

func (h *Handler) updateDevice(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    
    // Get existing device
    device, err := h.storage.GetDevice(id)
    if err != nil {
        if errors.Is(err, storage.ErrDeviceNotFound) {
            h.writeError(w, http.StatusNotFound, "DEVICE_NOT_FOUND", "Device not found")
            return
        }
        h.internalError(w, err)
        return
    }

    // Parse updates
    var updates map[string]interface{}
    if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
        h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
        return
    }

    // Apply updates
    if name, ok := updates["name"].(string); ok {
        device.Name = name
    }
    if description, ok := updates["description"].(string); ok {
        device.Description = description
    }
    if makeModel, ok := updates["make_model"].(string); ok {
        device.MakeModel = makeModel
    }
    if os, ok := updates["os"].(string); ok {
        device.OS = os
    }
    if datacenterID, ok := updates["datacenter_id"].(string); ok {
        device.DatacenterID = datacenterID
    }

    // Update device
    if err := h.storage.UpdateDevice(&device); err != nil {
        if errors.Is(err, storage.ErrDeviceNotFound) {
            h.writeError(w, http.StatusNotFound, "DATACENTER_NOT_FOUND", "Referenced datacenter not found")
            return
        }
        h.internalError(w, err)
        return
    }

    h.writeJSON(w, http.StatusOK, device)
}

func (h *Handler) deleteDevice(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")

    if err := h.storage.DeleteDevice(id); err != nil {
        if errors.Is(err, storage.ErrDeviceNotFound) {
            h.writeError(w, http.StatusNotFound, "DEVICE_NOT_FOUND", "Device not found")
            return
        }
        h.internalError(w, err)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) searchDevices(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("q")
    if query == "" {
        h.writeError(w, http.StatusBadRequest, "MISSING_REQUIRED_FIELD", "Search query is required").WithDetails("field", "q")
        return
    }

    devices, err := h.storage.SearchDevices(query)
    if err != nil {
        h.internalError(w, err)
        return
    }

    h.writeJSON(w, http.StatusOK, devices)
}

// Helper functions for parameter parsing

func parseArrayParam(r *http.Request, name string) []string {
    values := r.URL.Query()[name]
    if len(values) == 0 {
        return nil
    }
    return values
}

func parseIntParam(r *http.Request, name string, defaultValue int) int {
    values := r.URL.Query()[name]
    if len(values) == 0 {
        return defaultValue
    }
    var result int
    fmt.Sscanf(values[0], "%d", &result)
    return result
}

    // Network routes
    mux.HandleFunc("GET /api/networks", wrap(h.listNetworks))
    mux.HandleFunc("POST /api/networks", wrap(h.createNetwork))
    mux.HandleFunc("GET /api/networks/{id}", wrap(h.getNetwork))
    mux.HandleFunc("PUT /api/networks/{id}", wrap(h.updateNetwork))
    mux.HandleFunc("DELETE /api/networks/{id}", wrap(h.deleteNetwork))
    mux.HandleFunc("GET /api/networks/{id}/devices", wrap(h.getNetworkDevices))
    mux.HandleFunc("GET /api/networks/{id}/utilization", wrap(h.getNetworkUtilization))

    // Network pool routes
    mux.HandleFunc("GET /api/networks/{id}/pools", wrap(h.listNetworkPools))
    mux.HandleFunc("POST /api/networks/{id}/pools", wrap(h.createNetworkPool))
    mux.HandleFunc("GET /api/pools/{id}", wrap(h.getNetworkPool))
    mux.HandleFunc("PUT /api/pools/{id}", wrap(h.updateNetworkPool))
    mux.HandleFunc("DELETE /api/pools/{id}", wrap(h.deleteNetworkPool))
    mux.HandleFunc("GET /api/pools/{id}/next-ip", wrap(h.getNextIP))
    mux.HandleFunc("GET /api/pools/{id}/heatmap", wrap(h.getPoolHeatmap))

    // Device routes
    mux.HandleFunc("GET /api/devices", wrap(h.listDevices))
    mux.HandleFunc("POST /api/devices", wrap(h.createDevice))
    mux.HandleFunc("GET /api/devices/{id}", wrap(h.getDevice))
    mux.HandleFunc("PUT /api/devices/{id}", wrap(h.updateDevice))
    mux.HandleFunc("DELETE /api/devices/{id}", wrap(h.deleteDevice))
    mux.HandleFunc("GET /api/devices/search", wrap(h.searchDevices))

    // Relationship routes
    mux.HandleFunc("POST /api/devices/{id}/relationships", wrap(h.addRelationship))
    mux.HandleFunc("GET /api/devices/{id}/relationships", wrap(h.getRelationships))
    mux.HandleFunc("GET /api/devices/{id}/related", wrap(h.getRelatedDevices))
    mux.HandleFunc("DELETE /api/devices/{id}/relationships/{child_id}/{type}", wrap(h.removeRelationship))
}

// Helper methods
func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
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

// ===== UI Asset Serving =====

**File**: `internal/ui/ui.go`

```go
package ui

import (
    "embed"
    "io/fs"
    "net/http"
    "mime"
)

//go:embed all:dist
var distFS embed.FS

//go:embed all:dist/*.css
var cssFS embed.FS

type UIHandler struct {
    fs fs.FS
}

func NewUIHandler() *UIHandler {
    return &UIHandler{ fs: distFS }
}

// RegisterRoutes registers UI static file routes
func RegisterRoutes(mux *http.ServeMux) {
    ui := NewUIHandler()
    
    // Serve index.html at root
    mux.HandleFunc("GET /", ui.serveIndex)
    mux.HandleFunc("GET /app.js", ui.serveJS)
    mux.HandleFunc("GET /output.css", ui.serveCSS)
    
    // Serve all other static files from dist/
    fileServer := http.FileServer(http.FS(distFS))
    mux.Handle("GET /assets/", fileServer)
}

func (ui *UIHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
    // Serve index.html for all routes (SPA routing)
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.Header().Set("Cache-Control", "no-cache")
    
    content, err := ui.fs.ReadFile("dist/index.html")
    if err != nil {
        http.Error(w, "UI not found", http.StatusNotFound)
        return
    }
    
    w.WriteHeader(http.StatusOK)
    w.Write(content)
}

func (ui *UIHandler) serveJS(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/javascript")
    w.Header().Set("Cache-Control", "public, max-age=3600")
    
    content, err := ui.fs.ReadFile("dist/app.js")
    if err != nil {
        http.Error(w, "JS bundle not found", http.StatusNotFound)
        return
    }
    
    w.WriteHeader(http.StatusOK)
    w.Write(content)
}

func (ui *UIHandler) serveCSS(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/css; charset=utf-8")
    w.Header().Set("Cache-Control", "public, max-age=3600")
    
    content, err := cssFS.ReadFile("dist/output.css")
    if err != nil {
        http.Error(w, "CSS not found", http.StatusNotFound)
        return
    }
    
    w.WriteHeader(http.StatusOK)
    w.Write(content)
}

// getMIMEType determines MIME type for file extension
func getMIMEType(filename string) string {
    ext := filename[strings.LastIndexByte(filename, '.'):]
    switch ext {
    case ".js":
        return "application/javascript"
    case ".css":
        return "text/css; charset=utf-8"
    case ".html":
        return "text/html; charset=utf-8"
    case ".svg":
        return "image/svg+xml"
    case ".png":
        return "image/png"
    default:
        return mime.TypeByExtension(ext)
    }
}
```

## Config Handler for UI

```go
// ===== internal/api/config_handlers.go =====

import (
    "encoding/json"
    "net/http"

    "github.com/martinsuchenak/rackd/internal/ui"
)

var uiConfig ui.UIConfig

// ConfigureUI returns a callback that collects UI config from features
func ConfigureUI(builder *UIConfigBuilder) func() {
    return func() {
        uiConfig = builder.Build()
    }
}

func (h *Handler) serveUIConfig(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(uiConfig)
}
```
```

## Middleware

**File**: `internal/api/middleware.go`

```go
package api

import (
    "net/http"
    "strings"
)

// AuthMiddleware validates bearer tokens
func AuthMiddleware(token string, next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        auth := r.Header.Get("Authorization")
        if !strings.HasPrefix(auth, "Bearer ") {
            http.Error(w, `{"error":"Unauthorized","code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
            return
        }

        providedToken := strings.TrimPrefix(auth, "Bearer ")
        if providedToken != token {
            http.Error(w, `{"error":"Unauthorized","code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
            return
        }

        next(w, r)
    }
}

// SecurityHeaders adds security headers to all responses
func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

        // HSTS for HTTPS
        if r.TLS != nil {
            w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        }

        next.ServeHTTP(w, r)
    })
}
```

## MCP Server

**File**: `internal/mcp/server.go`

```go
package mcp

import (
    "net/http"
    "strings"

    "github.com/paularlott/mcp"
    "github.com/martinsuchenak/rackd/internal/storage"
)

type Server struct {
    mcpServer   *mcp.Server
    storage     storage.ExtendedStorage
    bearerToken string
}

func NewServer(storage storage.ExtendedStorage, bearerToken string) *Server {
    s := &Server{
        mcpServer:   mcp.NewServer("rackd", "1.0.0"),
        storage:     storage,
        bearerToken: bearerToken,
    }
    s.registerTools()
    return s
}

// Inner returns the underlying MCP server for feature registration
func (s *Server) Inner() *mcp.Server {
    return s.mcpServer
}

func (s *Server) registerTools() {
    // Device tools
    s.mcpServer.RegisterTool(
        mcp.NewTool("device_save", "Create or update a device",
            mcp.String("id", "Device ID (omit for new device)"),
            mcp.String("name", "Device name", mcp.Required()),
            mcp.String("description", "Device description"),
            mcp.String("make_model", "Device make and model"),
            mcp.String("os", "Operating system"),
            mcp.String("datacenter_id", "Datacenter ID"),
            mcp.String("username", "Login username"),
            mcp.String("location", "Physical location"),
            mcp.Array("tags", "Device tags"),
            mcp.Array("addresses", "IP addresses"),
            mcp.Array("domains", "Domain names"),
        ),
        s.handleDeviceSave,
    )

    s.mcpServer.RegisterTool(
        mcp.NewTool("device_get", "Get a device by ID",
            mcp.String("id", "Device ID", mcp.Required()),
        ),
        s.handleDeviceGet,
    )

    s.mcpServer.RegisterTool(
        mcp.NewTool("device_list", "List devices with optional filters",
            mcp.String("query", "Search query"),
            mcp.Array("tags", "Filter by tags"),
            mcp.String("datacenter_id", "Filter by datacenter"),
        ),
        s.handleDeviceList,
    )

    s.mcpServer.RegisterTool(
        mcp.NewTool("device_delete", "Delete a device",
            mcp.String("id", "Device ID", mcp.Required()),
        ),
        s.handleDeviceDelete,
    )

    // Relationship tools
    s.mcpServer.RegisterTool(
        mcp.NewTool("device_add_relationship", "Add a relationship between devices",
            mcp.String("parent_id", "Parent device ID", mcp.Required()),
            mcp.String("child_id", "Child device ID", mcp.Required()),
            mcp.String("type", "Relationship type (contains, connected_to, depends_on)", mcp.Required()),
        ),
        s.handleAddRelationship,
    )

    s.mcpServer.RegisterTool(
        mcp.NewTool("device_get_relationships", "Get all relationships for a device",
            mcp.String("id", "Device ID", mcp.Required()),
        ),
        s.handleGetRelationships,
    )

    // Datacenter tools
    s.mcpServer.RegisterTool(
        mcp.NewTool("datacenter_list", "List all datacenters"),
        s.handleDatacenterList,
    )

    s.mcpServer.RegisterTool(
        mcp.NewTool("datacenter_save", "Create or update a datacenter",
            mcp.String("id", "Datacenter ID (omit for new)"),
            mcp.String("name", "Datacenter name", mcp.Required()),
            mcp.String("location", "Physical location"),
            mcp.String("description", "Description"),
        ),
        s.handleDatacenterSave,
    )

    // Network tools
    s.mcpServer.RegisterTool(
        mcp.NewTool("network_list", "List all networks",
            mcp.String("datacenter_id", "Filter by datacenter"),
        ),
        s.handleNetworkList,
    )

    s.mcpServer.RegisterTool(
        mcp.NewTool("network_save", "Create or update a network",
            mcp.String("id", "Network ID (omit for new)"),
            mcp.String("name", "Network name", mcp.Required()),
            mcp.String("subnet", "CIDR subnet (e.g., 192.168.1.0/24)", mcp.Required()),
            mcp.String("datacenter_id", "Datacenter ID"),
            mcp.Int("vlan_id", "VLAN ID"),
            mcp.String("description", "Description"),
        ),
        s.handleNetworkSave,
    )

    // Pool tools
    s.mcpServer.RegisterTool(
        mcp.NewTool("pool_get_next_ip", "Get the next available IP from a pool",
            mcp.String("pool_id", "Pool ID", mcp.Required()),
        ),
        s.handleGetNextIP,
    )

    // Discovery tools
    s.mcpServer.RegisterTool(
        mcp.NewTool("discovery_scan", "Start a network discovery scan",
            mcp.String("network_id", "Network ID to scan", mcp.Required()),
            mcp.String("scan_type", "Scan type: quick, full, deep"),
        ),
        s.handleStartScan,
    )

    s.mcpServer.RegisterTool(
        mcp.NewTool("discovery_list", "List discovered devices",
            mcp.String("network_id", "Network ID"),
        ),
        s.handleListDiscovered,
    )

    s.mcpServer.RegisterTool(
        mcp.NewTool("discovery_promote", "Promote a discovered device to inventory",
            mcp.String("discovered_id", "Discovered device ID", mcp.Required()),
            mcp.String("name", "Device name", mcp.Required()),
        ),
        s.handlePromoteDevice,
    )
}

func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request) {
    // Optional bearer token authentication
    if s.bearerToken != "" {
        auth := r.Header.Get("Authorization")
        if !strings.HasPrefix(auth, "Bearer ") {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        token := strings.TrimPrefix(auth, "Bearer ")
        if token != s.bearerToken {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
    }
    s.mcpServer.HandleRequest(w, r)
}
```
