# Architecture

This document covers the project philosophy, architecture principles, and technology stack for Rackd.

## Purpose

**Rackd** is an open-source **IP Address Management (IPAM) and Device Inventory System** for managing network infrastructure. It provides:

- Device tracking (servers, switches, routers, etc.)
- Datacenter and physical location management
- Network subnet and VLAN definitions
- IP address pools and allocation
- Device relationships and dependencies
- Network discovery and inventory
- MCP (Model Context Protocol) integration for AI/automation

## Architecture Principles

The project follows a **Clean Architecture** approach adapted for Go:

- **Core (OSS)**: Standalone, functional, SQLite-based, fully featured IPAM
- **Enterprise (Enterprise)**: Wraps Core, injects Postgres storage, adds SSO/RBAC features

**Key Rule**: The Core package must define *interfaces* for everything it consumes (Storage, Features, Discovery). It must never depend on concrete Enterprise implementations.

## Technology Stack

| Component | Technology | Notes |
|-----------|------------|-------|
| Language | Go 1.25+ | CGO-free, pure Go |
| Database | SQLite via `modernc.org/sqlite` | Embedded, no external dependencies |
| Frontend | TypeScript + Alpine.js + TailwindCSS v4 | Lightweight, reactive SPA |
| Build | Bun (frontend), Make (orchestration) | Fast builds |
| CLI | `paularlott/cli` | Subcommand-based CLI |
| MCP | `paularlott/mcp` | AI/automation integration |
| Logging | `paularlott/logger` | Structured logging |
| HTTP | Go's `net/http` with ServeMux | Pattern-based routing (Go 1.22+) |

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                        User Interfaces                               │
│  ┌──────────────┬──────────────┬──────────────┬──────────────────┐  │
│  │   Web UI     │     CLI      │  MCP Server  │  REST API        │  │
│  │ (Alpine.js)  │  Commands    │  (AI Tools)  │  (JSON)          │  │
│  └──────────────┴──────────────┴──────────────┴──────────────────┘  │
├─────────────────────────────────────────────────────────────────────┤
│                      HTTP API Layer                                  │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │  API Handlers & Routing (Go 1.22+ ServeMux)                    │ │
│  │  - Device CRUD        - Datacenter CRUD                        │ │
│  │  - Network/Pool CRUD  - Discovery & Relationships              │ │
│  │  - Auth Middleware    - Security Headers                       │ │
│  └────────────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────────────┤
│                   Business Logic & Services                          │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │  Discovery Scanner     │  Scheduler (Background Jobs)          │ │
│  │  Configuration Mgmt    │  Structured Logging                   │ │
│  │  Feature Injection     │  Enterprise Interface Hooks              │ │
│  └────────────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────────────┤
│                      Data Layer (Storage)                            │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │  Storage Interfaces (DeviceStorage, NetworkStorage, etc.)      │ │
│  │  ┌─────────────────┐  ┌─────────────────┐                     │ │
│  │  │ SQLite (OSS)    │  │ Postgres (Prem) │                     │ │
│  │  └─────────────────┘  └─────────────────┘                     │ │
│  └────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```
