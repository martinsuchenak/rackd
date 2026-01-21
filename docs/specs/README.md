# Rackd Technical Specifications

This directory contains the technical specifications and architecture documentation for Rackd, an open-source IP Address Management (IPAM) and Device Inventory System.

## Overview

Rackd is designed with a clean architecture that supports both OSS (open source) and Enterprise (enterprise) editions through interface-based extension points.

## Specifications

### Core Architecture

| Document | Description |
|----------|-------------|
| [01-architecture.md](01-architecture.md) | Project philosophy, principles, and technology stack |
| [02-oss-enterprise-split.md](02-oss-enterprise-split.md) | Two-repository architecture and feature injection patterns |
| [03-feature-matrix.md](03-feature-matrix.md) | OSS vs Enterprise feature classification |
| [04-directory-structure.md](04-directory-structure.md) | Project layout and dependencies |

### Implementation Details

| Document | Description |
|----------|-------------|
| [05-data-models.md](05-data-models.md) | Data structures for devices, networks, datacenters, discovery |
| [06-storage.md](06-storage.md) | Storage interfaces and SQLite implementation |
| [07-api.md](07-api.md) | HTTP API handlers, middleware, and MCP server |
| [08-web-ui.md](08-web-ui.md) | Frontend architecture and Enterprise UI extension patterns |

### Commands & Features

| Document | Description |
|----------|-------------|
| [09-cli.md](09-cli.md) | Command-line interface structure |
| [10-discovery.md](10-discovery.md) | Network discovery scanner and scheduler |

### Operations

| Document | Description |
|----------|-------------|
| [11-build-deploy.md](11-build-deploy.md) | Makefile, Docker, GoReleaser, Nomad deployment |
| [12-configuration.md](12-configuration.md) | Environment variables and configuration options |

### Reference

| Document | Description |
|----------|-------------|
| [13-database-schema.md](13-database-schema.md) | SQLite schema and entity relationships |
| [14-api-reference.md](14-api-reference.md) | Complete API endpoint reference |
| [15-testing.md](15-testing.md) | Testing strategy and patterns |
| [16-security.md](16-security.md) | Security considerations, policies, and practices |
| [17-monitoring.md](17-monitoring.md) | Monitoring, logging, and observability strategy |
| [18-user-guide.md](18-user-guide.md) | High-level user guide and common use cases |
| [19-ui-layout.md](19-ui-layout.md) | UI layout, design philosophy, and component inventory |

## Quick Links

- **Getting Started**: See [04-directory-structure.md](04-directory-structure.md) for project setup
- **API Development**: See [07-api.md](07-api.md) and [14-api-reference.md](14-api-reference.md)
- **Frontend Development**: See [08-web-ui.md](08-web-ui.md)
- **Enterprise Extension**: See [02-oss-enterprise-split.md](02-oss-enterprise-split.md)

## Technology Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.25+ (CGO-free) |
| Database | SQLite via `modernc.org/sqlite` |
| Frontend | TypeScript + Alpine.js + TailwindCSS v4 |
| Build | Bun (frontend), Make (orchestration) |
| CLI | `paularlott/cli` v0.7.2 |
| MCP | `paularlott/mcp` v0.9.2 |
| Logging | `paularlott/logger` v0.3.0 |

## Implementation Order

1. Models (`internal/model/`)
2. Storage (`internal/storage/`)
3. Config (`internal/config/`)
4. Logging (`internal/log/`)
5. API Handlers (`internal/api/`)
6. MCP Server (`internal/mcp/`)
7. Discovery (`internal/discovery/`)
8. Worker (`internal/worker/`)
9. Server (`internal/server/`)
10. CLI Commands (`cmd/*/`)
11. Web UI (`webui/`)
12. Main (`main.go`)
13. Deployment
