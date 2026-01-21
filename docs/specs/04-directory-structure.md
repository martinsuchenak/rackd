# Directory Structure

This document outlines the project directory layout and file organization.

## Project Layout

```text
/
├── cmd/
│   ├── server/              # Main server entry point
│   │   └── server.go        # Server CLI command
│   ├── device/              # CLI subcommand for devices
│   │   ├── list.go          # 'rackd device list'
│   │   ├── get.go           # 'rackd device get'
│   │   ├── add.go           # 'rackd device add'
│   │   ├── delete.go        # 'rackd device delete'
│   │   └── update.go        # 'rackd device update'
│   ├── network/             # CLI subcommand for networks
│   │   ├── list.go
│   │   ├── get.go
│   │   └── ...
│   ├── datacenter/          # CLI subcommand for datacenters
│   │   ├── list.go
│   │   └── ...
│   └── discovery/           # CLI subcommand for discovery
│       ├── scan.go          # 'rackd discovery scan'
│       └── list.go          # 'rackd discovery list'
├── internal/
│   ├── api/                 # HTTP Handlers
│   │   ├── handlers.go              # Router setup & shared logic
│   │   ├── device_handlers.go       # Device CRUD
│   │   ├── network_handlers.go      # Network CRUD
│   │   ├── datacenter_handlers.go   # Datacenter CRUD
│   │   ├── pool_handlers.go         # Network Pool CRUD
│   │   ├── discovery_handlers.go    # Discovery endpoints
│   │   └── middleware.go            # Auth & security middleware
│   ├── config/              # Configuration loading
│   │   └── config.go
│   ├── discovery/           # Network discovery
│   │   ├── scanner.go       # Discovery scanner implementation
│   │   └── interfaces.go    # Scanner interfaces
│   ├── log/                 # Structured logging wrapper
│   │   └── log.go
│   ├── mcp/                 # MCP Server
│   │   └── server.go        # Server setup & tool registration
│   ├── model/               # Pure data structs
│   │   ├── device.go        # Device & Address models
│   │   ├── datacenter.go    # Datacenter model
│   │   ├── network.go       # Network & NetworkPool models
│   │   ├── relationship.go  # DeviceRelationship model
│   │   └── discovery.go     # Discovery models
│   ├── server/              # Server assembly & Feature registry
│   │   └── server.go
│   ├── storage/             # Storage Interfaces & SQLite
│   │   ├── storage.go       # Interface definitions
│   │   ├── sqlite.go        # SQLite implementation
│   │   ├── discovery_sqlite.go # Discovery storage
│   │   ├── migrations.go    # Schema migrations
│   │   └── encode.go        # Utility functions
│   ├── types/               # Enterprise interface definitions
│   │   └── enterprise.go
│   ├── ui/                  # Embedded Web UI
│   │   ├── ui.go            # Asset serving
│   │   └── assets/          # Compiled frontend assets
│   └── worker/              # Background job scheduler
│       ├── scheduler.go     # Job scheduling and execution
│       ├── jobs.go         # Job definitions (discovery, cleanup)
│       └── worker.go       # Generic worker interface
│       └── scheduler.go
├── webui/                   # Frontend Source
│   ├── src/
│   │   ├── core/            # Shared, extractable code (mobile-ready)
│   │   │   ├── api.ts       # API client (no DOM dependencies)
│   │   │   ├── types.ts     # Shared TypeScript types
│   │   │   └── utils.ts     # Pure utility functions
│   │   ├── components/      # Alpine.js components
│   │   │   ├── devices.ts   # Devices UI module
│   │   │   ├── networks.ts  # Networks UI module
│   │   │   ├── pools.ts     # IP pools UI module
│   │   │   ├── datacenters.ts # Datacenters UI module
│   │   │   ├── discovery.ts # Discovery UI module
│   │   │   ├── search.ts    # Global search
│   │   │   └── nav.ts       # Navigation component
│   │   ├── app.ts           # Main app initialization
│   │   ├── index.html       # Main HTML
│   │   └── styles.css       # Tailwind base styles
│   ├── dist/                # Build output
│   ├── package.json         # Bun config
│   └── tsconfig.json
├── api/                     # API Schema
│   └── openapi.yaml         # OpenAPI 3.1 specification
├── docs/                    # Documentation
│   ├── specs/               # Technical specifications
│   └── ...
├── deploy/                  # Deployment configs
│   ├── Dockerfile
│   ├── docker-compose.yml
│   └── nomad.hcl
├── main.go                  # Root CLI entry point
├── go.mod
├── go.sum
├── Makefile
├── .goreleaser.yml          # Release automation
├── .env.example             # Example environment file
├── AGENTS.md                # Development guidelines
├── CLAUDE.md                # Claude Code instructions
└── README.md
```

## Core Dependencies (`go.mod`)

```go
module github.com/martinsuchenak/rackd

go 1.25

require (
    github.com/google/uuid v1.6.0
    github.com/paularlott/cli v0.7.2
    github.com/paularlott/logger v0.3.0
    github.com/paularlott/mcp v0.9.2
    modernc.org/sqlite v1.42.2
)
```

### Dependency Notes

- `modernc.org/sqlite`: Pure Go SQLite implementation (CGO-free)
- `paularlott/*`: Custom CLI framework ecosystem
- `google/uuid`: UUIDv7 generation for entity IDs
