# Architecture

This document describes the overall architecture, design principles, and technology stack of Rackd.

## Overview

Rackd is an open-source IP Address Management (IPAM) and Device Inventory System designed to be:

- **Self-contained**: Single binary with embedded SQLite database
- **Lightweight**: No external dependencies required
- **Extensible**: Clean interfaces for future enhancements
- **Multi-interface**: Web UI, CLI, REST API, and MCP server

## Design Principles

### 1. Simplicity First
- Single binary deployment
- Embedded SQLite database (no separate DB server)
- Zero-configuration startup
- Sensible defaults

### 2. Clean Architecture
- Clear separation of concerns
- Interface-based design
- Dependency injection
- Testable components

### 3. API-First Design
- All functionality exposed via REST API
- CLI and UI consume the same API
- MCP server for AI/automation integration

### 4. No CGO Dependencies
- Pure Go implementation
- Uses `modernc.org/sqlite` (CGO-free SQLite)
- Easy cross-compilation

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        User Interfaces                           │
│  ┌──────────┬──────────┬──────────────┬──────────────────────┐  │
│  │  Web UI  │   CLI    │  MCP Server  │  REST API Clients    │  │
│  │ Alpine.js│ Commands │  (AI Tools)  │  (curl, scripts)     │  │
│  └──────────┴──────────┴──────────────┴──────────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│                      HTTP Server Layer                           │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Go 1.22+ ServeMux (Pattern-based Routing)                 │ │
│  │  - Security Headers    - Auth Middleware                   │ │
│  │  - Request Logging     - Body Size Limits                  │ │
│  └────────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│                      API Handler Layer                           │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Device Handlers       Network Handlers                    │ │
│  │  Datacenter Handlers   Discovery Handlers                  │ │
│  │  Relationship Handlers Pool Handlers                       │ │
│  │  Config Handlers       MCP Request Handler                 │ │
│  └────────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│                   Business Logic Layer                           │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Discovery Scanner     Scheduled Scan Worker               │ │
│  │  Configuration Mgmt    Credential Encryption               │ │
│  │  Validation Logic      IP Address Utilities                │ │
│  └────────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│                      Storage Layer                               │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Storage Interfaces (DeviceStorage, NetworkStorage, etc.)  │ │
│  │  ┌──────────────────────────────────────────────────────┐ │ │
│  │  │  SQLite Implementation (modernc.org/sqlite)          │ │ │
│  │  │  - WAL mode enabled                                  │ │ │
│  │  │  - Foreign keys enforced                             │ │ │
│  │  │  - Automatic migrations                              │ │ │
│  │  └──────────────────────────────────────────────────────┘ │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## Technology Stack

| Component | Technology | Version | Purpose |
|-----------|------------|---------|---------|
| **Backend** |
| Language | Go | 1.25+ | Core application logic |
| Database | SQLite | 3.x | Embedded data storage |
| SQLite Driver | modernc.org/sqlite | latest | CGO-free SQLite driver |
| CLI Framework | paularlott/cli | v0.7.2 | Command-line interface |
| MCP Framework | paularlott/mcp | v0.9.2 | Model Context Protocol |
| Logger | paularlott/logger | v0.3.0 | Structured logging |
| **Frontend** |
| Language | TypeScript | 5.7+ | Type-safe JavaScript |
| Framework | Alpine.js | 3.14+ | Reactive UI framework |
| CSS | TailwindCSS | 4.0+ | Utility-first CSS |
| Build Tool | Bun | latest | Fast JS bundler |
| **Build & Deploy** |
| Build Tool | Make | - | Build orchestration |
| Container | Docker | - | Containerization |
| Orchestration | Nomad | - | Optional deployment |
| Release | GoReleaser | - | Multi-platform releases |

## Directory Structure

```
rackd/
├── cmd/                    # CLI commands
│   ├── server/            # Server command
│   ├── device/            # Device management commands
│   ├── network/           # Network management commands
│   ├── datacenter/        # Datacenter management commands
│   ├── discovery/         # Discovery commands
│   └── client/            # Shared CLI client utilities
├── internal/              # Private application code
│   ├── api/              # HTTP API handlers
│   ├── config/           # Configuration management
│   ├── credentials/      # Credential storage & encryption
│   ├── discovery/        # Network discovery scanner
│   ├── log/              # Logging utilities
│   ├── mcp/              # MCP server implementation
│   ├── model/            # Data models
│   ├── server/           # HTTP server setup
│   ├── storage/          # Storage layer & SQLite impl
│   ├── ui/               # Embedded UI assets
│   └── worker/           # Background workers
├── pkg/                   # Public API packages
│   ├── rackd/            # Public types
│   └── server/           # Server package
├── webui/                 # Frontend source code
│   ├── src/
│   │   ├── components/   # Alpine.js components
│   │   ├── core/         # Core utilities & API client
│   │   ├── partials/     # HTML partials
│   │   ├── app.ts        # Main application
│   │   ├── index.html    # HTML template
│   │   └── styles.css    # Tailwind styles
│   ├── dist/             # Built assets
│   └── package.json      # Frontend dependencies
├── api/                   # API specifications
│   └── openapi.yaml      # OpenAPI 3.0 spec
├── docs/                  # Documentation
├── deploy/                # Deployment configs
│   └── nomad.hcl         # Nomad job spec
├── data/                  # Runtime data directory
│   └── rackd.db          # SQLite database
├── main.go               # Application entry point
├── Makefile              # Build automation
├── Dockerfile            # Container image
├── docker-compose.yml    # Docker Compose config
└── go.mod                # Go dependencies
```

## Data Flow

### 1. HTTP Request Flow

```
User Request
    ↓
HTTP Server (ServeMux)
    ↓
Security Headers Middleware
    ↓
Auth Middleware (if enabled)
    ↓
API Handler
    ↓
Validation
    ↓
Storage Layer
    ↓
SQLite Database
    ↓
Response (JSON)
```

### 2. CLI Command Flow

```
CLI Command
    ↓
Command Parser (paularlott/cli)
    ↓
HTTP Client
    ↓
REST API Request
    ↓
[Same as HTTP Request Flow]
```

### 3. MCP Request Flow

```
MCP Client (AI Tool)
    ↓
MCP Server Handler
    ↓
Tool Dispatcher
    ↓
Storage Layer
    ↓
SQLite Database
    ↓
MCP Response (JSON-RPC)
```

### 4. Discovery Flow

```
Scheduled Scan Trigger
    ↓
Discovery Scanner
    ↓
Network Scan (ping, ARP, SSH, SNMP)
    ↓
Discovered Devices
    ↓
Storage Layer
    ↓
SQLite Database
    ↓
Optional: Promote to Inventory
```

## Key Design Decisions

### 1. SQLite as Primary Database

**Rationale**: 
- Zero configuration
- No separate database server
- Excellent performance for IPAM workloads
- ACID compliance
- Easy backup (single file)

**Trade-offs**:
- Single-writer limitation (acceptable for IPAM use case)
- Not suitable for massive multi-user deployments
- No built-in replication

### 2. CGO-Free Implementation

**Rationale**:
- Easy cross-compilation
- No C dependencies
- Simpler deployment
- Better portability

**Implementation**:
- Uses `modernc.org/sqlite` instead of `mattn/go-sqlite3`

### 3. Embedded Web UI

**Rationale**:
- Single binary deployment
- No separate web server needed
- Consistent versioning

**Implementation**:
- UI built during compilation
- Assets embedded using Go embed
- Served via HTTP handlers

### 4. Alpine.js for Frontend

**Rationale**:
- Lightweight (15KB)
- No build step required for development
- Reactive without complexity
- Easy to learn

**Trade-offs**:
- Less powerful than React/Vue
- Smaller ecosystem
- Limited component libraries

### 5. Pattern-Based Routing (Go 1.22+)

**Rationale**:
- No external router dependency
- Native HTTP method routing
- Path parameter extraction
- Better performance

**Example**:
```go
mux.HandleFunc("GET /api/devices/{id}", handler)
mux.HandleFunc("PUT /api/devices/{id}", handler)
```

## Security Architecture

### Authentication
- Optional bearer token authentication
- Separate tokens for API and MCP
- Constant-time comparison to prevent timing attacks

### Authorization
- Currently single-user (no RBAC in OSS)
- All authenticated requests have full access

### Data Protection
- Credentials encrypted at rest (AES-256-GCM)
- Encryption key from environment variable
- HTTPS recommended for production

### Security Headers
- X-Content-Type-Options: nosniff
- X-Frame-Options: DENY
- X-XSS-Protection: 1; mode=block
- Content-Security-Policy (strict)

## Performance Considerations

### Database Optimization
- WAL mode enabled for better concurrency
- Indexes on frequently queried columns
- Foreign keys for referential integrity
- Prepared statements for common queries

### Caching Strategy
- No application-level caching (SQLite is fast enough)
- Browser caching for static assets
- ETags for API responses (future)

### Concurrency
- SQLite handles concurrent reads efficiently
- Single writer with WAL mode
- Background workers use separate connections

## Extensibility Points

### 1. Storage Interface
```go
type ExtendedStorage interface {
    DeviceStorage
    DatacenterStorage
    NetworkStorage
    DiscoveryStorage
    // ... other interfaces
}
```

### 2. Feature Interface
```go
type Feature interface {
    Name() string
    RegisterRoutes(mux *http.ServeMux)
    RegisterMCPTools(mcpServer interface{})
}
```

### 3. Discovery Scanner Interface
```go
type Scanner interface {
    Scan(ctx context.Context, network string) error
    GetScanStatus(scanID string) (*ScanStatus, error)
    CancelScan(scanID string) error
}
```

## Future Considerations

### Potential Enhancements
- PostgreSQL storage backend (for larger deployments)
- Multi-user support with RBAC
- SSO/OIDC integration
- Audit logging
- Metrics and monitoring
- GraphQL API
- WebSocket for real-time updates
- Plugin system

### Scalability Path
- Read replicas (if PostgreSQL backend added)
- Horizontal scaling (stateless API servers)
- Distributed discovery workers
- Message queue for async operations

## Related Documentation

- [Development Guide](development.md) - Building and contributing
- [Database Schema](database.md) - Database structure
- [API Reference](api.md) - REST API documentation
- [Security](security.md) - Security best practices
