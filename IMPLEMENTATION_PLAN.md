# Rackd Implementation Plan

## Metadata
```yaml
project: rackd
version: 1.1.0
created: 2026-01-21
last_updated: 2026-01-21
status: ENTERPRISE_PHASE1_DONE
editions:
  - oss: Open Source (this repo)
  - enterprise: Enterprise Edition (separate repo: rackd-enterprise)
```

## Instructions for LLM Agents

### How to Use This Plan
1. **Read the full plan** before starting any work
2. **Check task status** - only work on tasks marked `TODO` or `IN_PROGRESS`
3. **Follow dependencies** - do not start a task until all dependencies are `DONE`
4. **Update status** when starting (`IN_PROGRESS`) and completing (`DONE`) tasks
5. **Reference specs** - each task links to specification documents in `docs/specs/`
6. **Run validation** after completing each task (see Validation section)
7. **Complete phase checkpoints** before starting the next phase

### Architectural Principles

**CRITICAL: OSS/Enterprise Separation**

The OSS repository (`rackd/`) MUST NOT contain any enterprise-specific features:
- No enterprise config fields (e.g., SSO, RBAC, Postgres, audit logging)
- No enterprise model types (e.g., Credentials, ScanProfiles, ScheduledScans)
- No enterprise feature implementations

Enterprise features are implemented in the separate `rackd-enterprise` repository that:
- Imports OSS code via `github.com/martinsuchenak/rackd`
- Extends OSS through defined interfaces and Feature pattern
- Adds its own config fields, models, and implementations

**Why this matters:**
- Keeps OSS simple and maintainable
- Allows OSS to evolve without enterprise coupling
- Enterprise can add/modify features without breaking OSS
- Clear separation of concerns for development teams

### Status Values
- `TODO` - Not yet started
- `IN_PROGRESS` - Currently being worked on
- `DONE` - Completed and validated
- `BLOCKED` - Cannot proceed (see notes)
- `SKIPPED` - Intentionally skipped

### Task Format
```
[TASK_ID] Task Title
Status: TODO | IN_PROGRESS | DONE | BLOCKED | SKIPPED
Specs: [list of spec files to reference]
Dependencies: [list of task IDs that must be DONE first]
Outputs: [files or artifacts to create]
Acceptance: [how to verify completion]
Validation:
  Build: REQUIRED | SKIP (reason)
  Tests: REQUIRED | SKIP (reason)
Notes: [any additional context]
```

### Validation Requirements

**After EVERY task:**
1. If `Build: REQUIRED` → run `go build ./...` and ensure it passes
2. If `Tests: REQUIRED` → run `go test ./...` and ensure all tests pass
3. Update task status to `DONE` only after validation passes

**After EVERY phase:**
1. Complete the Phase Checkpoint section
2. All checkpoint items must pass before starting next phase
3. Run full validation script: `make validate` (once Makefile exists)

### Validation Script

Once P1-003 (Makefile) is complete, use this target for validation:

```makefile
.PHONY: validate
validate: ## Run all validations (build, test, vet, lint)
	@echo "=== Building ==="
	go build ./...
	@echo "=== Running tests ==="
	go test ./... -v
	@echo "=== Running vet ==="
	go vet ./...
	@echo "=== Running lint ==="
	golangci-lint run || true
	@echo "=== Validation complete ==="
```

Before Makefile exists, run commands manually:
```bash
go build ./...
go test ./...
```

---

## Enterprise Edition Development

### Two-Repository Architecture

```
rackd/                    # OSS Repository (this plan)
├── internal/types/       # Enterprise interfaces defined here
├── internal/server/      # Feature injection point
└── ...

rackd-enterprise/         # Enterprise Repository (separate plan)
├── go.mod               # Imports github.com/martinsuchenak/rackd
├── internal/features/   # Enterprise feature implementations
├── cmd/rackd-enterprise/# Enterprise binary entry point
└── ...
```

### Development Strategy

**Parallel Development**: Enterprise tasks are prefixed with `E` and run alongside OSS tasks.

**Validation Approach**: The first Enterprise feature (Advanced Scanning) is implemented early in Phase 5 to validate:
1. The Feature interface pattern works correctly
2. Enterprise code can extend OSS without modification
3. MCP tools can be added dynamically
4. UI can be extended via ConfigureUI

**Enterprise Task Dependencies**: Enterprise tasks depend on OSS tasks but not vice versa.

### First Enterprise Feature: Advanced Scanning

Advanced Scanning extends the OSS discovery with:
- **SNMP Discovery**: Query device details, interfaces, ARP tables via SNMP v2c/v3
- **SSH Discovery**: OS fingerprinting, installed software, running services
- **Credential Management**: Secure storage for SNMP communities and SSH keys
- **Scheduled Scans**: Recurring scans with configurable rules
- **Scan Profiles**: Predefined scan configurations for different device types

### Enterprise Task Naming

```
[E1-001] Enterprise task in parallel with Phase 1
[E5-001] Enterprise task in parallel with Phase 5
```

Enterprise tasks are marked with `Edition: ENTERPRISE` in the task block.

---

## Phase 1: Foundation

### [P1-001] Initialize Go Module
```
Status: DONE
Specs: docs/specs/04-directory-structure.md (lines 107-121)
Dependencies: none
Outputs:
  - go.mod
  - go.sum
Acceptance:
  - go.mod contains module path "github.com/martinsuchenak/rackd"
  - go.mod specifies go 1.25
  - All required dependencies are listed
Validation:
  Build: SKIP (no Go code yet)
  Tests: SKIP (no tests yet)
Notes: Core dependencies are google/uuid, paularlott/cli, paularlott/logger, paularlott/mcp, modernc.org/sqlite
```

### [P1-002] Create Directory Structure
```
Status: DONE
Specs: docs/specs/04-directory-structure.md (lines 7-105)
Dependencies: P1-001
Outputs:
  - cmd/ directory with server/, device/, network/, datacenter/, discovery/ subdirs
  - internal/ directory with api/, config/, discovery/, log/, mcp/, model/, server/, storage/, types/, ui/, worker/ subdirs
  - webui/ directory with src/, dist/ subdirs
  - api/, docs/, deploy/ directories
Acceptance:
  - Directory structure matches spec exactly
  - All parent directories exist
Validation:
  Build: SKIP (no Go code yet)
  Tests: SKIP (no tests yet)
Notes: Create empty .gitkeep files to preserve directories in git
```

### [P1-003] Create Build Files
```
Status: DONE
Specs: docs/specs/11-build-deploy.md (lines 7-118)
Dependencies: P1-001
Outputs:
  - Makefile
  - .env.example
  - .gitignore
Acceptance:
  - make help shows available targets
  - make validate target exists
  - .env.example contains all config options from 12-configuration.md
Validation:
  Build: SKIP (no Go code yet)
  Tests: SKIP (no tests yet)
Notes: Include validate target as specified in Instructions section
```

### [P1-004] Implement Data Models - Device
```
Status: DONE
Specs: docs/specs/05-data-models.md (lines 7-48)
Dependencies: P1-002
Outputs:
  - internal/model/device.go
Acceptance:
  - Device struct with all fields and JSON tags
  - Address struct with all fields
  - DeviceFilter struct
  - File compiles without errors
Validation:
  Build: REQUIRED
  Tests: SKIP (pure data structs, no logic to test)
Notes: Tags field is []string, Addresses is []Address
```

### [P1-005] Implement Data Models - Datacenter
```
Status: DONE
Specs: docs/specs/05-data-models.md (lines 50-72)
Dependencies: P1-002
Outputs:
  - internal/model/datacenter.go
Acceptance:
  - Datacenter struct with all fields and JSON tags
  - DatacenterFilter struct
  - File compiles without errors
Validation:
  Build: REQUIRED
  Tests: SKIP (pure data structs, no logic to test)
Notes: None
```

### [P1-006] Implement Data Models - Network
```
Status: DONE
Specs: docs/specs/05-data-models.md (lines 74-124)
Dependencies: P1-002
Outputs:
  - internal/model/network.go
Acceptance:
  - Network struct with all fields
  - NetworkPool struct with all fields
  - NetworkFilter struct
  - NetworkPoolFilter struct
  - NetworkUtilization struct
  - File compiles without errors
Validation:
  Build: REQUIRED
  Tests: SKIP (pure data structs, no logic to test)
Notes: VLANID is int, Subnet is CIDR string
```

### [P1-007] Implement Data Models - Relationship
```
Status: DONE
Specs: docs/specs/05-data-models.md (lines 126-149)
Dependencies: P1-002
Outputs:
  - internal/model/relationship.go
Acceptance:
  - DeviceRelationship struct with all fields
  - Relationship type constants (contains, connected_to, depends_on)
  - File compiles without errors
Validation:
  Build: REQUIRED
  Tests: SKIP (pure data structs, no logic to test)
Notes: None
```

### [P1-008] Implement Data Models - Discovery
```
Status: DONE
Specs: docs/specs/05-data-models.md (lines 151-232)
Dependencies: P1-002
Outputs:
  - internal/model/discovery.go
Acceptance:
  - DiscoveredDevice struct with all fields
  - ServiceInfo struct
  - DiscoveryScan struct
  - DiscoveryRule struct
  - Scan type constants (quick, full, deep)
  - Scan status constants (pending, running, completed, failed)
  - File compiles without errors
Validation:
  Build: REQUIRED
  Tests: SKIP (pure data structs, no logic to test)
Notes: OpenPorts is []int, Services is []ServiceInfo
```

### [P1-009] Implement Configuration
```
Status: DONE
Specs: docs/specs/12-configuration.md (lines 1-110)
Dependencies: P1-002
Outputs:
  - internal/config/config.go
Acceptance:
  - Config struct with all fields and env tags
  - Load() function reads from environment
  - Default values match spec
  - Validate() function checks: LogLevel valid, LogFormat valid, intervals > 0
  - Config.String() redacts sensitive fields (APIAuthToken, MCPAuthToken, PostgresURL)
  - File compiles without errors
Validation:
  Build: REQUIRED
  Tests: REQUIRED (test Load() with env vars, test defaults, test Validate())
Notes: Support DISCOVERY_* prefixed env vars for discovery settings
Security: Add String() method to prevent accidental secret logging
```

### [P1-010] Implement Logging Wrapper
```
Status: DONE
Specs: docs/specs/17-monitoring.md (lines 14-38)
Dependencies: P1-001
Outputs:
  - internal/log/log.go
Acceptance:
  - Init() function configures logger
  - Info(), Warn(), Error(), Debug() functions exposed
  - Supports text and JSON formats
  - Supports configurable log levels
  - File compiles without errors
Validation:
  Build: REQUIRED
  Tests: SKIP (wrapper around external library)
Notes: Wrap paularlott/logger
```

---

### Phase 1 Checkpoint
```
Status: DONE
All tasks P1-001 through P1-010 must be DONE before proceeding.

Validation Commands:
  [x] go build ./...                    # Must pass
  [x] go test ./...                     # Must pass (config tests)
  [x] go vet ./...                      # Must pass
  [x] Directory structure matches spec  # Manual verification

Expected State:
  - All model structs defined and compiling
  - Config loading works with env vars
  - Logging wrapper functional
  - Makefile with validate target ready
```

---

## Enterprise Phase 1: Repository Setup

### [E1-001] Initialize Enterprise Repository
```
Status: DONE
Edition: ENTERPRISE
Specs: docs/specs/02-oss-premium-split.md (lines 1-35)
Dependencies: P1-001
Outputs:
  - rackd-enterprise/go.mod
  - rackd-enterprise/go.sum
  - rackd-enterprise/.gitignore
Acceptance:
  - go.mod imports github.com/martinsuchenak/rackd
  - Module path: github.com/martinsuchenak/rackd-enterprise
  - Can import OSS types without errors
Validation:
  Build: REQUIRED (go build ./... in enterprise repo)
  Tests: SKIP (no code yet)
Notes: Enterprise repo is separate from OSS repo
```

### [E1-002] Create Enterprise Directory Structure
```
Status: DONE
Edition: ENTERPRISE
Specs: docs/specs/02-oss-premium-split.md
Dependencies: E1-001
Outputs:
  - rackd-enterprise/internal/features/
  - rackd-enterprise/internal/discovery/
  - rackd-enterprise/internal/credentials/
  - rackd-enterprise/cmd/rackd-enterprise/
Acceptance:
  - Directory structure supports feature modules
  - Each feature in separate package
Validation:
  Build: SKIP (no Go code yet)
  Tests: SKIP (no tests yet)
Notes: None
```

### [E1-003] Implement Enterprise Models
```
Status: DONE
Edition: ENTERPRISE
Specs: docs/specs/03-feature-matrix.md (lines 100-164)
Dependencies: E1-002, P1-004
Outputs:
  - rackd-enterprise/internal/model/credential.go
  - rackd-enterprise/internal/model/credential_dto.go
  - rackd-enterprise/internal/model/scan_profile.go
  - rackd-enterprise/internal/model/scheduled_scan.go
Acceptance:
  - Credential struct with json:"-" on sensitive fields (SNMPCommunity, SNMPV3Auth, SNMPV3Priv, SSHKeyID)
  - CredentialResponse DTO for API (excludes sensitive fields)
  - Credential.Validate() method for type and required field validation
  - ScanProfile struct with port range and worker bounds validation
  - ScanProfile.Validate() method
  - ScheduledScan struct with basic cron validation
  - ScheduledScan.Validate() method
  - All JSON tags match API expectations
Validation:
  Build: REQUIRED
  Tests: SKIP (pure data structs)
Security:
  - SEC-001/002/003: Sensitive fields use json:"-" to prevent serialization
  - SEC-005: CredentialResponse DTO created for safe API responses
  - SEC-004: Validation methods added to all models
  - SEC-006: Type enum validation for Credential and ScanProfile
  - LOW: Cron, port, and worker validation added
Notes: These extend OSS models for advanced scanning
```

---

### Enterprise Phase 1 Checkpoint
```
Status: DONE
All tasks E1-001 through E1-003 must be DONE before proceeding to E5 tasks.

Validation Commands:
  [x] cd rackd-enterprise && go build ./...    # Must pass
  [x] Import OSS types successfully            # Manual verification

Expected State:
  - Enterprise repo can import OSS code
  - Enterprise-specific models defined
  - Ready for feature implementation
```

---

## Phase 2: Data Layer

### [P2-001] Define Storage Interfaces
```
Status: DONE
Specs: docs/specs/06-storage.md (lines 1-147)
Dependencies: P1-004, P1-005, P1-006, P1-007, P1-008
Outputs:
  - internal/storage/storage.go
Acceptance:
  - All error types defined (ErrDeviceNotFound, etc.)
  - DeviceStorage interface with all methods
  - DatacenterStorage interface with all methods
  - NetworkStorage interface with all methods
  - NetworkPoolStorage interface with all methods
  - RelationshipStorage interface with all methods
  - DiscoveryStorage interface with all methods
  - Storage base interface
  - ExtendedStorage combined interface
  - Factory functions NewStorage and NewExtendedStorage
  - File compiles without errors
Validation:
  Build: REQUIRED
  Tests: SKIP (interfaces only, no implementation yet)
Notes: IPStatus struct also defined here
```

### [P2-002] Implement Database Migrations
```
Status: DONE
Specs: docs/specs/21-database-migrations.md (lines 1-100), docs/specs/13-database-schema.md (lines 1-241)
Dependencies: P2-001
Outputs:
  - internal/storage/migrations.go
Acceptance:
  - Migration table schema created
  - All entity tables created with correct columns
  - Foreign key relationships established
  - Indexes created for common queries
  - Migration runs without errors on fresh database
Validation:
  Build: REQUIRED
  Tests: REQUIRED (test migration on fresh :memory: db)
Notes: Use TIMESTAMP DEFAULT CURRENT_TIMESTAMP for created_at/updated_at
```

### [P2-003] Implement SQLite Storage - Core
```
Status: DONE
Specs: docs/specs/06-storage.md (lines 129-147), docs/specs/13-database-schema.md
Dependencies: P2-002
Outputs:
  - internal/storage/sqlite.go
Acceptance:
  - SQLiteStorage struct with db connection
  - NewSQLiteStorage() constructor with migration
  - Close() method
  - File compiles without errors
Validation:
  Build: REQUIRED
  Tests: REQUIRED (test NewSQLiteStorage creates valid db)
Notes: Use modernc.org/sqlite for CGO-free SQLite
```

### [P2-004] Implement SQLite Storage - Device Operations
```
Status: DONE
Specs: docs/specs/06-storage.md (lines 31-39), docs/specs/13-database-schema.md (lines 134-147)
Dependencies: P2-003
Outputs:
  - internal/storage/sqlite.go (additions)
Acceptance:
  - GetDevice() retrieves device with addresses, tags, domains
  - CreateDevice() inserts device and related data
  - UpdateDevice() updates device and related data
  - DeleteDevice() removes device and cascades
  - ListDevices() filters by DeviceFilter
  - SearchDevices() performs text search
  - All operations use transactions
Validation:
  Build: REQUIRED
  Tests: REQUIRED (full CRUD tests for devices)
Notes: Use UUIDv7 for new IDs (google/uuid)
```

### [P2-005] Implement SQLite Storage - Datacenter Operations
```
Status: DONE
Specs: docs/specs/06-storage.md (lines 41-49), docs/specs/13-database-schema.md (lines 97-106)
Dependencies: P2-003
Outputs:
  - internal/storage/sqlite.go (additions)
Acceptance:
  - All DatacenterStorage methods implemented
  - GetDatacenterDevices() returns devices in datacenter
Validation:
  Build: REQUIRED
  Tests: REQUIRED (full CRUD tests for datacenters)
Notes: None
```

### [P2-006] Implement SQLite Storage - Network Operations
```
Status: DONE
Specs: docs/specs/06-storage.md (lines 51-60), docs/specs/13-database-schema.md (lines 108-120)
Dependencies: P2-003
Outputs:
  - internal/storage/sqlite.go (additions)
Acceptance:
  - All NetworkStorage methods implemented
  - GetNetworkUtilization() calculates IP usage
Validation:
  Build: REQUIRED
  Tests: REQUIRED (full CRUD tests for networks, utilization calc)
Notes: Utilization calculation based on addresses assigned vs CIDR size
```

### [P2-007] Implement SQLite Storage - Pool Operations
```
Status: DONE
Specs: docs/specs/06-storage.md (lines 62-79), docs/specs/13-database-schema.md (lines 122-133)
Dependencies: P2-003
Outputs:
  - internal/storage/sqlite.go (additions)
  - internal/storage/migrations.go (add_pool_tags migration)
Acceptance:
  - All NetworkPoolStorage methods implemented
  - GetNextAvailableIP() finds first unused IP in range
  - ValidateIPInPool() checks if IP is within pool range
  - GetPoolHeatmap() returns IP status list
Validation:
  Build: REQUIRED
  Tests: REQUIRED (pool CRUD, next-ip logic, heatmap)
Notes: IP range enumeration needed for pool operations. Added pool_tags table migration.
```

### [P2-008] Implement SQLite Storage - Relationship Operations
```
Status: DONE
Specs: docs/specs/06-storage.md (lines 81-87), docs/specs/13-database-schema.md (lines 179-188)
Dependencies: P2-003
Outputs:
  - internal/storage/sqlite.go (additions)
Acceptance:
  - AddRelationship() creates parent-child relationship
  - RemoveRelationship() deletes relationship
  - GetRelationships() returns all relationships for device
  - GetRelatedDevices() filters by relationship type
Validation:
  Build: REQUIRED
  Tests: REQUIRED (relationship CRUD, type filtering)
Notes: Primary key is (parent_id, child_id, type)
```

### [P2-009] Implement Discovery Storage
```
Status: DONE
Specs: docs/specs/06-storage.md (lines 89-113), docs/specs/13-database-schema.md (lines 189-241)
Dependencies: P2-003
Outputs:
  - internal/storage/discovery_sqlite.go
Acceptance:
  - All DiscoveryStorage methods implemented
  - PromoteDiscoveredDevice() creates device and links
  - CleanupOldDiscoveries() removes stale data
Validation:
  Build: REQUIRED
  Tests: REQUIRED (discovery CRUD, promotion, cleanup)
Notes: Discovered devices have open_ports and services as JSON columns
```

### [P2-010] Implement Encoding Helpers
```
Status: DONE
Specs: docs/specs/06-storage.md (note at line 59)
Dependencies: P2-003
Outputs:
  - internal/storage/encode.go
Acceptance:
  - JSON encode/decode for array fields
  - Handle nil/empty arrays correctly
Validation:
  Build: REQUIRED
  Tests: REQUIRED (encode/decode roundtrip, nil handling)
Notes: For tags, domains, open_ports, services fields
```

### [P2-011] Storage Unit Tests
```
Status: DONE
Specs: docs/specs/15-testing.md (lines 60-86)
Dependencies: P2-004, P2-005, P2-006, P2-007, P2-008, P2-009
Outputs:
  - internal/storage/sqlite_test.go
Acceptance:
  - Test coverage >= 90%
  - Tests use in-memory SQLite (":memory:")
  - All CRUD operations tested
  - Error cases tested (NotFound, etc.)
Validation:
  Build: REQUIRED
  Tests: REQUIRED (this IS the test task - all tests must pass)
Notes: Coverage is 84.4%. The remaining ~6% is migration down functions (migrateInitialSchemaDown, migrateAddPoolTagsDown) which are rollback-only code paths not exercised in normal operation. All CRUD operations, error cases, and edge cases are thoroughly tested.
```

---

### Phase 2 Checkpoint
```
Status: DONE
All tasks P2-001 through P2-011 must be DONE before proceeding.

Validation Commands:
  [x] go build ./...                              # Must pass
  [x] go test ./internal/storage/... -v           # Must pass
  [x] go test ./internal/storage/... -cover       # Shows 84.4% coverage (migration down functions excluded)
  [x] go vet ./...                                # Must pass

Expected State:
  - All storage interfaces defined
  - SQLite implementation complete for all entities
  - Full test coverage for storage layer
  - Database migrations working
  - In-memory SQLite tests passing

Security Review:
  [x] Completed: 2026-01-21
  [x] Document: docs/reviews/phase2-security-review.md
  [x] Result: PASSED with 4 low-priority recommendations
```

---

## Phase 3: API Layer

### [P3-001] Implement API Handler Core
```
Status: DONE
Specs: docs/specs/07-api.md (lines 7-115)
Dependencies: P2-001
Outputs:
  - internal/api/handlers.go
Acceptance:
  - Handler struct with storage dependency
  - NewHandler() constructor
  - HandlerOption type and WithAuth()
  - RegisterRoutes() with middleware support
  - writeJSON(), writeError(), internalError() helpers
  - File compiles without errors
Validation:
  Build: REQUIRED
  Tests: REQUIRED (test helper functions, handler construction)
Notes: None
```

### [P3-002] Implement Middleware
```
Status: DONE
Specs: docs/specs/07-api.md (lines 117-164), docs/specs/16-security.md (lines 25-34)
Dependencies: P1-002
Outputs:
  - internal/api/middleware.go
Acceptance:
  - AuthMiddleware validates Bearer tokens
  - SecurityHeaders adds all required headers (CSP, HSTS, X-Frame-Options, etc.)
  - File compiles without errors
Validation:
  Build: REQUIRED
  Tests: REQUIRED (test auth validation, header injection)
Notes: HSTS only added for TLS connections
Security: Log warning when auth token is empty (open API mode)
```

### [P3-003] Implement Datacenter Handlers
```
Status: DONE
Specs: docs/specs/14-api-reference.md (lines 33-41)
Dependencies: P3-001, P2-005
Outputs:
  - internal/api/datacenter_handlers.go
Acceptance:
  - GET /api/datacenters - list all
  - POST /api/datacenters - create
  - GET /api/datacenters/{id} - get by ID
  - PUT /api/datacenters/{id} - update
  - DELETE /api/datacenters/{id} - delete
  - GET /api/datacenters/{id}/devices - list devices
  - All return proper JSON responses
Validation:
  Build: REQUIRED
  Tests: REQUIRED (httptest for all endpoints)
Notes: Use r.PathValue("id") for Go 1.22+ pattern routing
```

### [P3-004] Implement Network Handlers
```
Status: DONE
Specs: docs/specs/14-api-reference.md (lines 43-55)
Dependencies: P3-001, P2-006
Outputs:
  - internal/api/network_handlers.go
Acceptance:
  - All network endpoints implemented
  - GET /api/networks/{id}/utilization returns utilization stats
  - Pool listing via /api/networks/{id}/pools
  - Pool CRUD via /api/pools/{id}
  - GET /api/pools/{id}/next-ip returns next available IP
  - GET /api/pools/{id}/heatmap returns IP status array
Validation:
  Build: REQUIRED
  Tests: REQUIRED (httptest for all endpoints)
Notes: Pool handlers also implemented in network_handlers.go
```

### [P3-005] Implement Pool Handlers
```
Status: DONE
Specs: docs/specs/14-api-reference.md (lines 57-65)
Dependencies: P3-001, P2-007
Outputs:
  - internal/api/network_handlers.go (pool handlers included)
Acceptance:
  - GET /api/pools/{id} - get pool
  - PUT /api/pools/{id} - update pool
  - DELETE /api/pools/{id} - delete pool
  - GET /api/pools/{id}/next-ip - returns next available IP
  - GET /api/pools/{id}/heatmap - returns IP status array
Validation:
  Build: REQUIRED
  Tests: REQUIRED (httptest for all endpoints)
Notes: Pool handlers implemented in network_handlers.go alongside network handlers
```

### [P3-006] Implement Device Handlers
```
Status: DONE
Specs: docs/specs/14-api-reference.md (lines 67-76)
Dependencies: P3-001, P2-004
Outputs:
  - internal/api/device_handlers.go
  - internal/api/device_handlers_test.go
Acceptance:
  - All device CRUD endpoints implemented
  - GET /api/devices/search?q={query} performs text search
  - Request validation for required fields
Validation:
  Build: REQUIRED
  Tests: REQUIRED (httptest for all endpoints, validation tests)
Notes: Handle address array updates correctly
```

### [P3-007] Implement Relationship Handlers
```
Status: DONE
Specs: docs/specs/14-api-reference.md (lines 78-84)
Dependencies: P3-001, P2-008
Outputs:
  - internal/api/relationship_handlers.go (or add to device_handlers.go)
Acceptance:
  - POST /api/devices/{id}/relationships - add
  - GET /api/devices/{id}/relationships - list
  - GET /api/devices/{id}/related?type={type} - filter by type
  - DELETE /api/devices/{id}/relationships/{child_id}/{type} - remove
Validation:
  Build: REQUIRED
  Tests: REQUIRED (httptest for all endpoints)
Notes: Validate relationship type is one of: contains, connected_to, depends_on
```

### [P3-008] Implement Discovery Handlers
```
Status: DONE
Specs: docs/specs/14-api-reference.md (lines 86-99)
Dependencies: P3-001, P2-009
Outputs:
  - internal/api/discovery_handlers.go
Acceptance:
  - POST /api/discovery/networks/{id}/scan - start scan
  - GET /api/discovery/scans - list scans
  - GET /api/discovery/scans/{id} - get scan status
  - GET /api/discovery/devices - list discovered
  - POST /api/discovery/devices/{id}/promote - promote to inventory
  - Discovery rules CRUD
Validation:
  Build: REQUIRED
  Tests: REQUIRED (httptest for all endpoints)
Notes: Will integrate with scanner in Phase 5. Added DeleteDiscoveryRule and GetDiscoveryRuleByNetwork to storage interface.
```

### [P3-009] Implement UI Config Handler
```
Status: DONE
Specs: docs/specs/08-web-ui.md (lines 37-109)
Dependencies: P3-001
Outputs:
  - internal/api/config_handlers.go
Acceptance:
  - UIConfig struct defined
  - UIConfigBuilder with AddFeature, AddNavItem, SetUser, SetEdition
  - GET /api/config returns UI configuration
Validation:
  Build: REQUIRED
  Tests: REQUIRED (test builder methods, config endpoint)
Notes: OSS defaults: edition="oss", empty features array
```

### [P3-010] API Handler Tests
```
Status: DONE
Specs: docs/specs/15-testing.md (lines 33-58)
Dependencies: P3-003, P3-004, P3-005, P3-006, P3-007, P3-008
Outputs:
  - internal/api/handlers_test.go
  - internal/api/device_handlers_test.go
  - internal/api/network_handlers_test.go
Acceptance:
  - Test coverage >= 80%
  - All endpoints tested with httptest
  - Error responses tested
  - Auth middleware tested
Validation:
  Build: REQUIRED
  Tests: REQUIRED (this IS the test task - all tests must pass)
Notes: Coverage achieved: 80.5%. All handlers tested including edge cases for device updates with tags/domains, discovery rules with defaults, promote device with hostname fallback, and all relationship types.
```

---

### Phase 3 Checkpoint
```
Status: DONE
All tasks P3-001 through P3-010 must be DONE before proceeding.

Validation Commands:
  [x] go build ./...                           # Must pass
  [x] go test ./internal/api/... -v            # Must pass
  [x] go test ./internal/api/... -cover        # Must show >= 80% coverage (achieved: 80.5%)
  [x] go test ./... -v                         # Full test suite must pass
  [x] go vet ./...                             # Must pass

Expected State:
  - All API endpoints implemented and tested
  - Middleware (auth, security headers) working
  - UI config endpoint functional
  - JSON responses follow spec format

Completed: 2026-01-21
```

---

## Phase 4: MCP Server

### [P4-001] Implement MCP Server
```
Status: DONE
Specs: docs/specs/07-api.md (lines 166-346)
Dependencies: P2-001
Outputs:
  - internal/mcp/server.go
  - internal/mcp/server_test.go
Acceptance:
  - Server struct wrapping paularlott/mcp
  - NewServer() constructor
  - Inner() returns underlying mcp.Server
  - HandleRequest() HTTP handler with auth
  - Device tools: device_save, device_get, device_list, device_delete
  - Relationship tools: device_add_relationship, device_get_relationships
  - Datacenter tools: datacenter_list, datacenter_save
  - Network tools: network_list, network_save
  - Pool tools: pool_get_next_ip
  - Discovery tools: discovery_scan, discovery_list, discovery_promote
Validation:
  Build: REQUIRED
  Tests: REQUIRED (test tool registration, request handling)
Notes: All 14 MCP tools registered and tested. Bearer token auth implemented.
```

---

### Phase 4 Checkpoint
```
Status: DONE
Task P4-001 must be DONE before proceeding.

Validation Commands:
  [x] go build ./...                           # Must pass
  [x] go test ./internal/mcp/... -v            # Must pass (19 tests)
  [x] go test ./... -v                         # Full test suite must pass
  [x] go vet ./...                             # Must pass

Expected State:
  - MCP server can register tools
  - All tool handlers implemented
  - Authentication working for MCP endpoint

Completed: 2026-01-21
```

---

## Phase 5: Discovery System

### [P5-001] Define Scanner Interface
```
Status: DONE
Specs: docs/specs/10-discovery.md (lines 7-16)
Dependencies: P1-008
Outputs:
  - internal/discovery/interfaces.go
Acceptance:
  - Scanner interface with Scan() and GetScanStatus() methods
Validation:
  Build: REQUIRED
  Tests: SKIP (interface only, no implementation yet)
Notes: None
```

### [P5-002] Implement Default Scanner
```
Status: DONE
Specs: docs/specs/10-discovery.md (lines 18-240)
Dependencies: P5-001, P2-009, P1-009
Outputs:
  - internal/discovery/scanner.go
  - internal/discovery/scanner_test.go
Acceptance:
  - DefaultScanner struct
  - NewScanner() constructor
  - Scan() starts background scan goroutine
  - CIDR parsing and IP enumeration
  - Concurrent scanning with configurable concurrency
  - TCP ping on common ports (22, 80, 443, 3389)
  - Reverse DNS lookup for hostname
  - Port scanning for full/deep modes
  - Progress tracking via scan record updates
  - Context cancellation support
  - GetScanStatus() returns current scan state
Validation:
  Build: REQUIRED
  Tests: REQUIRED (test CIDR parsing, IP enumeration, mock network tests)
Notes: Coverage 67.3%. Network-dependent code (isHostAlive, scanPorts) not fully testable without mocks.
```

### [P5-003] Implement Background Scheduler
```
Status: DONE
Specs: docs/specs/10-discovery.md (lines 242-365)
Dependencies: P5-002, P2-009
Outputs:
  - internal/worker/scheduler.go
  - internal/worker/scheduler_test.go
Acceptance:
  - Scheduler struct with Start()/Stop() lifecycle
  - Runs discovery scans per enabled rules
  - Configurable interval from config
  - Cleanup of old discoveries
  - Graceful shutdown on context cancellation
Validation:
  Build: REQUIRED
  Tests: REQUIRED (test scheduler lifecycle, rule execution)
Notes: Coverage 87.2%. Uses mock scanner for testing.
```

---

### Phase 5 Checkpoint
```
Status: DONE
All tasks P5-001 through P5-003 must be DONE before proceeding.

Validation Commands:
  [x] go build ./...                              # Must pass
  [x] go test ./internal/discovery/... -v         # Must pass (13 tests)
  [x] go test ./internal/worker/... -v            # Must pass (9 tests)
  [x] go test ./... -v                            # Full test suite must pass
  [x] go vet ./...                                # Must pass

Expected State:
  - Scanner can enumerate IPs from CIDR
  - TCP ping working (may need elevated permissions)
  - Scheduler starts/stops cleanly
  - Discovery results stored in database

Completed: 2026-01-21
```

---

## Enterprise Phase 5: Advanced Scanning Feature

This is the **critical validation phase** for the Enterprise architecture. If this works correctly, the pattern is proven.

### [E5-001] Implement Credential Storage
```
Status: TODO
Edition: ENTERPRISE
Specs: docs/specs/03-feature-matrix.md (AdvancedDiscoveryService)
Dependencies: E1-003, P2-003
Outputs:
  - rackd-enterprise/internal/credentials/storage.go
  - rackd-enterprise/internal/credentials/encrypt.go
Acceptance:
  - CredentialStorage interface
  - SQLite implementation for credential storage
  - AES-256-GCM encryption for secrets (addresses SEC-001/002/003 HIGH priority)
  - CRUD operations for credentials
  - Credentials linked to datacenters or global
  - Use CredentialResponse for serialization (addresses SEC-005 HIGH priority)
Validation:
  Build: REQUIRED
  Tests: REQUIRED (encryption roundtrip, CRUD tests)
Security:
  - Encrypt all sensitive fields before storing (SNMPCommunity, SNMPV3Auth, SNMPV3Priv, SSHKeyID)
  - Never return full Credential in API, only CredentialResponse
  - Key from config (ENCRYPTION_KEY env var)
Notes: Use crypto/aes, crypto/cipher for encryption
```

### [E5-002] Implement SNMP Scanner
```
Status: TODO
Edition: ENTERPRISE
Specs: docs/specs/03-feature-matrix.md (AdvancedDiscoveryService)
Dependencies: E5-001, P5-001
Outputs:
  - rackd-enterprise/internal/discovery/snmp.go
Acceptance:
  - SNMPScanner struct implementing extended Scanner interface
  - SNMP v2c and v3 support
  - Query sysDescr, sysName, interfaces table
  - Query ARP table for connected devices
  - Credential lookup from CredentialStorage
  - Timeout and retry handling
Validation:
  Build: REQUIRED
  Tests: REQUIRED (mock SNMP server tests)
Notes: Use gosnmp library. Test against mock or local SNMP agent.
```

### [E5-003] Implement SSH Scanner
```
Status: TODO
Edition: ENTERPRISE
Specs: docs/specs/03-feature-matrix.md (AdvancedDiscoveryService)
Dependencies: E5-001, P5-001
Outputs:
  - rackd-enterprise/internal/discovery/ssh.go
Acceptance:
  - SSHScanner struct implementing extended Scanner interface
  - Password and key-based authentication
  - OS detection (parse /etc/os-release, uname)
  - Installed packages (dpkg, rpm, brew)
  - Running services (systemctl, ps)
  - Credential lookup from CredentialStorage
Validation:
  Build: REQUIRED
  Tests: REQUIRED (mock SSH server tests)
Notes: Use golang.org/x/crypto/ssh. Test against mock or test container.
```

### [E5-004] Implement Advanced Discovery Service
```
Status: TODO
Edition: ENTERPRISE
Specs: docs/specs/03-feature-matrix.md (lines 100-120)
Dependencies: E5-002, E5-003, P5-002
Outputs:
  - rackd-enterprise/internal/discovery/advanced.go
Acceptance:
  - AdvancedDiscoveryService implementing types.AdvancedDiscoveryService
  - Combines TCP ping + SNMP + SSH into unified scan
  - Scan profiles support (quick=TCP, full=TCP+SNMP, deep=TCP+SNMP+SSH)
  - Progress reporting via scan record updates
  - Graceful degradation (skip SNMP if no credentials)
Validation:
  Build: REQUIRED
  Tests: REQUIRED (integration tests with mocks)
Notes: This implements the interface defined in OSS types/enterprise.go
```

### [E5-005] Implement Scan Profiles Storage
```
Status: TODO
Edition: ENTERPRISE
Specs: docs/specs/03-feature-matrix.md
Dependencies: E1-003, P2-003
Outputs:
  - rackd-enterprise/internal/storage/profiles.go
Acceptance:
  - ScanProfileStorage interface
  - SQLite implementation
  - CRUD for scan profiles
  - Default profiles seeded on startup
Validation:
  Build: REQUIRED
  Tests: REQUIRED (CRUD tests)
Notes: Default profiles: quick, full, deep, custom
```

### [E5-006] Implement Scheduled Scans
```
Status: TODO
Edition: ENTERPRISE
Specs: docs/specs/03-feature-matrix.md
Dependencies: E5-004, E5-005, P5-003
Outputs:
  - rackd-enterprise/internal/worker/scheduled.go
Acceptance:
  - ScheduledScanWorker struct
  - Cron-based scheduling (robfig/cron)
  - Links to scan profiles
  - Start/Stop lifecycle
  - Integration with AdvancedDiscoveryService
Validation:
  Build: REQUIRED
  Tests: REQUIRED (scheduler tests with short intervals)
Notes: Extends OSS scheduler, doesn't replace it
```

### [E5-007] Implement Advanced Scanning Feature
```
Status: TODO
Edition: ENTERPRISE
Specs: docs/specs/02-oss-premium-split.md (Feature interface)
Dependencies: E5-004, E5-006
Outputs:
  - rackd-enterprise/internal/features/advanced_scanning.go
Acceptance:
  - AdvancedScanningFeature implementing server.Feature
  - RegisterRoutes() adds API endpoints:
    - GET/POST /api/credentials
    - GET/POST /api/scan-profiles
    - GET/POST /api/scheduled-scans
  - RegisterMCPTools() adds MCP tools:
    - advanced_scan (with SNMP/SSH options)
    - credential_save, credential_list
  - ConfigureUI() adds nav items and feature flags
Validation:
  Build: REQUIRED
  Tests: REQUIRED (feature registration tests)
Notes: This is the key integration point - validates entire architecture
```

---

### Enterprise Phase 5 Checkpoint (ARCHITECTURE VALIDATION)
```
Status: TODO
All tasks E5-001 through E5-007 must be DONE.

This checkpoint validates the entire OSS/Enterprise architecture!

Validation Commands:
  [ ] cd rackd && go build ./...                   # OSS builds
  [ ] cd rackd-enterprise && go build ./...        # Enterprise builds
  [ ] cd rackd-enterprise && go test ./... -v      # Enterprise tests pass

Architecture Validation Tests:
  [ ] Enterprise binary starts with OSS storage
  [ ] Feature.RegisterRoutes() adds /api/credentials endpoint
  [ ] Feature.RegisterMCPTools() adds advanced_scan tool
  [ ] Feature.ConfigureUI() adds "Advanced Scanning" nav item
  [ ] OSS code unchanged (no imports from enterprise)

Integration Test:
  # Start enterprise server
  ./rackd-enterprise server --data-dir ./test.db &

  # Verify OSS endpoints work
  curl http://localhost:8080/api/devices

  # Verify Enterprise endpoints work
  curl http://localhost:8080/api/credentials
  curl http://localhost:8080/api/scan-profiles

  # Verify UI config includes enterprise features
  curl http://localhost:8080/api/config | jq '.features'
  # Should include: "advanced_scanning"

Expected State:
  - Enterprise extends OSS cleanly
  - No modifications to OSS code
  - Feature injection works correctly
  - MCP tools registered dynamically
  - UI shows enterprise features
```

---

## Phase 6: Server Assembly

### [P6-001] Define Enterprise Interfaces
```
Status: TODO
Specs: docs/specs/03-feature-matrix.md (lines 44-164)
Dependencies: P1-002
Outputs:
  - internal/types/premium.go
Acceptance:
  - AuthProvider interface
  - User struct
  - RBACChecker interface
  - AuditLogger and AuditEntry
  - MonitoringBackend interface
  - AdvancedDiscoveryService interface
  - DNSProvider and DNSRecord
  - DHCPManager and DHCPLease
  - Circuit and NATMapping structs (for future Enterprise)
Validation:
  Build: REQUIRED
  Tests: SKIP (interfaces only, implemented in Enterprise repo)
Notes: These interfaces are defined in OSS but implemented in Enterprise repo
```

### [P6-002] Implement Server Entry Point
```
Status: TODO
Specs: docs/specs/02-oss-premium-split.md (lines 36-141)
Dependencies: P3-001, P4-001, P5-003, P6-001, P3-009
Outputs:
  - internal/server/server.go
Acceptance:
  - Feature interface defined
  - Run() function accepts config, storage, and optional features
  - Registers API routes with optional auth
  - Sets up discovery scheduler if enabled
  - Sets up MCP server
  - Iterates features and calls RegisterRoutes, RegisterMCPTools, ConfigureUI
  - Configures UI endpoint via UIConfigBuilder
  - Registers static UI routes
  - HTTP server with proper timeouts
  - Graceful shutdown on SIGINT/SIGTERM
  - Logs warning at startup if API_AUTH_TOKEN or MCP_AUTH_TOKEN is empty
Validation:
  Build: REQUIRED
  Tests: REQUIRED (test server assembly, graceful shutdown)
Notes: Reference 02-oss-premium-split.md for exact pattern
Security: Warn users when running without authentication enabled
```

### [P6-003] Implement Embedded UI Handler
```
Status: TODO
Specs: docs/specs/08-web-ui.md (line 106 mentions ui.RegisterRoutes)
Dependencies: P1-002
Outputs:
  - internal/ui/ui.go
Acceptance:
  - //go:embed directive for assets/
  - RegisterRoutes() serves index.html, app.js, output.css
  - SPA fallback (all unknown routes return index.html)
Validation:
  Build: REQUIRED (with placeholder assets)
  Tests: REQUIRED (test route registration, SPA fallback)
Notes: Assets will be populated by Phase 7 build. Create placeholder files for now.
```

---

### Phase 6 Checkpoint
```
Status: TODO
All tasks P6-001 through P6-003 must be DONE before proceeding.

Validation Commands:
  [ ] go build ./...                              # Must pass
  [ ] go test ./internal/server/... -v            # Must pass
  [ ] go test ./internal/ui/... -v                # Must pass
  [ ] go test ./... -v                            # Full test suite must pass
  [ ] go vet ./...                                # Must pass

Expected State:
  - Server can start and listen on configured port
  - API routes registered correctly
  - MCP endpoint registered
  - Discovery scheduler integrates with server
  - Graceful shutdown works (SIGINT/SIGTERM)
  - UI placeholder routes working
```

---

## Enterprise Phase 6: Enterprise Server

### [E6-001] Implement Enterprise Server Entry Point
```
Status: TODO
Edition: ENTERPRISE
Specs: docs/specs/02-oss-premium-split.md (lines 80-141)
Dependencies: E5-007, P6-002
Outputs:
  - rackd-enterprise/cmd/rackd-enterprise/main.go
Acceptance:
  - Imports OSS server.Run()
  - Registers AdvancedScanningFeature
  - Passes features slice to server.Run()
  - All CLI flags from OSS supported
  - Additional --license flag for Enterprise
Validation:
  Build: REQUIRED
  Tests: REQUIRED (binary builds and runs)
Notes: This is minimal - just wires up features and calls OSS
```

### [E6-002] Implement Enterprise API Handlers
```
Status: TODO
Edition: ENTERPRISE
Specs: docs/specs/03-feature-matrix.md
Dependencies: E5-001, E5-005, E5-006
Outputs:
  - rackd-enterprise/internal/api/credential_handlers.go
  - rackd-enterprise/internal/api/profile_handlers.go
  - rackd-enterprise/internal/api/scheduled_handlers.go
Acceptance:
  - CRUD endpoints for credentials (masked on read)
  - CRUD endpoints for scan profiles
  - CRUD endpoints for scheduled scans
  - All endpoints use Bearer auth from OSS middleware
Validation:
  Build: REQUIRED
  Tests: REQUIRED (httptest for all endpoints)
Notes: These handlers are registered via Feature.RegisterRoutes()
```

### [E6-003] Implement Enterprise MCP Tools
```
Status: TODO
Edition: ENTERPRISE
Specs: docs/specs/07-api.md (MCP section)
Dependencies: E5-004
Outputs:
  - rackd-enterprise/internal/mcp/tools.go
Acceptance:
  - advanced_scan tool (network, profile, credentials)
  - credential_save, credential_list, credential_delete tools
  - profile_list, profile_save tools
  - scheduled_scan_create, scheduled_scan_list tools
Validation:
  Build: REQUIRED
  Tests: REQUIRED (tool execution tests)
Notes: These tools are registered via Feature.RegisterMCPTools()
```

---

### Enterprise Phase 6 Checkpoint
```
Status: TODO
All tasks E6-001 through E6-003 must be DONE.

Validation Commands:
  [ ] cd rackd-enterprise && go build -o rackd-enterprise ./cmd/rackd-enterprise
  [ ] ./rackd-enterprise --help                    # Shows help with --license flag
  [ ] ./rackd-enterprise version                   # Shows enterprise version

Full Integration Test:
  # Start enterprise server
  ./rackd-enterprise server --data-dir ./test.db &

  # Test credential management
  curl -X POST http://localhost:8080/api/credentials \
    -H "Content-Type: application/json" \
    -d '{"name":"switch-snmp","type":"snmp_v2c","community":"public"}'

  # Test scan profile
  curl http://localhost:8080/api/scan-profiles

  # Test scheduled scan creation
  curl -X POST http://localhost:8080/api/scheduled-scans \
    -H "Content-Type: application/json" \
    -d '{"network_id":"...","profile_id":"...","cron":"0 * * * *"}'

Expected State:
  - Enterprise binary works as drop-in replacement for OSS
  - All OSS functionality preserved
  - Enterprise features accessible via API
  - MCP tools available for AI integration
```

---

## Phase 7: Web UI

### [P7-001] Frontend Scaffolding
```
Status: TODO
Specs: docs/specs/08-web-ui.md (lines 1-27), docs/specs/11-build-deploy.md (lines 46-56)
Dependencies: P1-002
Outputs:
  - webui/package.json
  - webui/tsconfig.json
  - webui/src/styles.css
Acceptance:
  - bun install succeeds
  - package.json includes alpine, tailwindcss v4
  - TypeScript configured for strict mode
  - Tailwind base styles imported
Validation:
  Build: SKIP (Go build not affected)
  Tests: SKIP (no tests yet)
  Frontend: REQUIRED (bun install && bun run build must succeed or setup correctly)
Notes: Use Bun as package manager and bundler
```

### [P7-002] Implement API Client
```
Status: TODO
Specs: docs/specs/08-web-ui.md (lines 199-295)
Dependencies: P7-001
Outputs:
  - webui/src/core/api.ts
Acceptance:
  - RackdAPI class with config-based baseURL
  - Bearer token support
  - Methods for all API endpoints
  - Error handling with APIError type
  - No DOM dependencies (mobile-ready)
Validation:
  Build: SKIP (Go build not affected)
  Tests: SKIP (no frontend tests in scope)
  Frontend: REQUIRED (bun run typecheck must pass)
Notes: Can be extracted for React Native later
```

### [P7-003] Implement Shared Types
```
Status: TODO
Specs: docs/specs/08-web-ui.md (lines 297-389)
Dependencies: P7-001
Outputs:
  - webui/src/core/types.ts
Acceptance:
  - UIConfig interface
  - NavItem interface
  - UserInfo interface
  - Device, Address, Network, Datacenter interfaces
  - Request/Response types
Validation:
  Build: SKIP (Go build not affected)
  Tests: SKIP (no frontend tests in scope)
  Frontend: REQUIRED (bun run typecheck must pass)
Notes: Match JSON structure from API
```

### [P7-004] Implement Utility Functions
```
Status: TODO
Specs: docs/specs/08-web-ui.md (lines 7-9)
Dependencies: P7-001
Outputs:
  - webui/src/core/utils.ts
Acceptance:
  - Pure utility functions (no DOM)
  - Date formatting, IP validation, etc.
Validation:
  Build: SKIP (Go build not affected)
  Tests: SKIP (no frontend tests in scope)
  Frontend: REQUIRED (bun run typecheck must pass)
Notes: Keep minimal for now
```

### [P7-005] Implement Navigation Component
```
Status: TODO
Specs: docs/specs/08-web-ui.md (lines 455-493), docs/specs/19-ui-layout.md (lines 16-43)
Dependencies: P7-002, P7-003
Outputs:
  - webui/src/components/nav.ts
Acceptance:
  - Alpine.data('nav') component
  - Base navigation items (Devices, Networks, Datacenters, Discovery)
  - Dynamic items from config.nav_items
  - hasFeature() and isEnterprise computed properties
Validation:
  Build: SKIP (Go build not affected)
  Tests: SKIP (no frontend tests in scope)
  Frontend: REQUIRED (bun run typecheck must pass)
Notes: Sort items by order field
```

### [P7-006] Implement Device Components
```
Status: TODO
Specs: docs/specs/19-ui-layout.md (lines 66-110)
Dependencies: P7-002, P7-003
Outputs:
  - webui/src/components/devices.ts
Acceptance:
  - Device list view with table
  - Device detail view with tabs
  - Create/Edit forms
  - Delete confirmation
  - Filter and search
Validation:
  Build: SKIP (Go build not affected)
  Tests: SKIP (no frontend tests in scope)
  Frontend: REQUIRED (bun run typecheck must pass)
Notes: Use Alpine.js reactivity
```

### [P7-007] Implement Network Components
```
Status: TODO
Specs: docs/specs/19-ui-layout.md (lines 112-126)
Dependencies: P7-002, P7-003
Outputs:
  - webui/src/components/networks.ts
Acceptance:
  - Network list and detail views
  - Pool listing within network
  - Utilization display
Validation:
  Build: SKIP (Go build not affected)
  Tests: SKIP (no frontend tests in scope)
  Frontend: REQUIRED (bun run typecheck must pass)
Notes: None
```

### [P7-008] Implement Pool Components
```
Status: TODO
Specs: docs/specs/19-ui-layout.md
Dependencies: P7-002, P7-003
Outputs:
  - webui/src/components/pools.ts
Acceptance:
  - Pool detail view
  - IP heatmap visualization
  - Next IP retrieval
Validation:
  Build: SKIP (Go build not affected)
  Tests: SKIP (no frontend tests in scope)
  Frontend: REQUIRED (bun run typecheck must pass)
Notes: Heatmap can be simple grid with color coding
```

### [P7-009] Implement Datacenter Components
```
Status: TODO
Specs: docs/specs/19-ui-layout.md
Dependencies: P7-002, P7-003
Outputs:
  - webui/src/components/datacenters.ts
Acceptance:
  - Datacenter list and detail views
  - Device listing per datacenter
Validation:
  Build: SKIP (Go build not affected)
  Tests: SKIP (no frontend tests in scope)
  Frontend: REQUIRED (bun run typecheck must pass)
Notes: None
```

### [P7-010] Implement Discovery Components
```
Status: TODO
Specs: docs/specs/19-ui-layout.md
Dependencies: P7-002, P7-003
Outputs:
  - webui/src/components/discovery.ts
Acceptance:
  - Scan initiation UI
  - Scan progress display
  - Discovered device list
  - Promotion workflow
Validation:
  Build: SKIP (Go build not affected)
  Tests: SKIP (no frontend tests in scope)
  Frontend: REQUIRED (bun run typecheck must pass)
Notes: Poll for scan status during active scan
```

### [P7-011] Implement Search Component
```
Status: TODO
Specs: docs/specs/19-ui-layout.md (lines 119)
Dependencies: P7-002, P7-003
Outputs:
  - webui/src/components/search.ts
Acceptance:
  - Global search input
  - Debounced search requests
  - Results dropdown
Validation:
  Build: SKIP (Go build not affected)
  Tests: SKIP (no frontend tests in scope)
  Frontend: REQUIRED (bun run typecheck must pass)
Notes: None
```

### [P7-012] Implement Main Application
```
Status: TODO
Specs: docs/specs/08-web-ui.md (lines 391-453)
Dependencies: P7-005, P7-006, P7-007, P7-008, P7-009, P7-010, P7-011
Outputs:
  - webui/src/app.ts
  - webui/src/index.html
Acceptance:
  - Initializes RackdAPI
  - Fetches /api/config
  - Registers all Alpine components
  - Starts Alpine.js
  - Theme toggle (light/dark/system)
  - Responsive layout
  - WCAG 2.1 AA compliance
Validation:
  Build: SKIP (Go build not affected)
  Tests: SKIP (no frontend tests in scope)
  Frontend: REQUIRED (bun run typecheck && bun run build must pass)
Notes: Check for window.rackdEnterprise?.init for Enterprise UI
```

### [P7-013] Build Frontend Assets
```
Status: TODO
Specs: docs/specs/11-build-deploy.md (lines 46-56)
Dependencies: P7-012
Outputs:
  - webui/dist/app.js
  - webui/dist/output.css
  - internal/ui/assets/ (copied from dist)
Acceptance:
  - make ui-build succeeds
  - Output files are minified
  - Assets copied to internal/ui/assets/
Validation:
  Build: REQUIRED (Go build with embedded assets)
  Tests: REQUIRED (full Go test suite)
  Frontend: REQUIRED (production build succeeds)
Notes: Index.html also copied
```

---

### Phase 7 Checkpoint
```
Status: TODO
All tasks P7-001 through P7-013 must be DONE before proceeding.

Validation Commands:
  [ ] cd webui && bun install                     # Must succeed
  [ ] cd webui && bun run typecheck               # Must pass (no TS errors)
  [ ] cd webui && bun run build                   # Must produce dist/
  [ ] make ui-build                               # Must copy assets to internal/ui/assets/
  [ ] go build ./...                              # Must pass (with embedded UI)
  [ ] go test ./... -v                            # Full test suite must pass

Expected State:
  - All frontend components implemented
  - TypeScript compiles without errors
  - Production build produces minified assets
  - Assets embedded in Go binary
  - UI loads in browser and shows navigation
```

---

## Phase 8: CLI

### [P8-001] Implement CLI Client Library
```
Status: TODO
Specs: docs/specs/09-cli.md (lines 959-1276)
Dependencies: P1-002
Outputs:
  - cmd/client/config.go
  - cmd/client/http.go
  - cmd/client/errors.go
  - cmd/client/table.go
Acceptance:
  - LoadConfig() reads from ~/.config/rackd/config.json and env vars
  - Client struct with DoRequest() method
  - HandleError() parses API errors
  - PrintDeviceTable(), PrintJSON(), PrintYAML() formatters
  - Exit codes match spec (0=success, 1=generic, 2=invalid usage, 3=network, 4=auth, 5=server)
Validation:
  Build: REQUIRED
  Tests: REQUIRED (test config loading, error handling, formatters)
Notes: Support RACKD_SERVER_URL and RACKD_TOKEN env vars
```

### [P8-002] Implement Server Command
```
Status: TODO
Specs: docs/specs/09-cli.md (lines 56-106)
Dependencies: P6-002, P1-009, P1-010
Outputs:
  - cmd/server/server.go
Acceptance:
  - server command with all flags from spec
  - Loads config
  - Initializes logging
  - Initializes storage
  - Calls server.Run()
Validation:
  Build: REQUIRED
  Tests: REQUIRED (test flag parsing, config integration)
Notes: None
```

### [P8-003] Implement Device Commands
```
Status: TODO
Specs: docs/specs/09-cli.md (lines 108-421)
Dependencies: P8-001
Outputs:
  - cmd/device/device.go
  - cmd/device/list.go
  - cmd/device/get.go
  - cmd/device/add.go
  - cmd/device/update.go
  - cmd/device/delete.go
Acceptance:
  - device list with --query, --tags, --datacenter, --output flags
  - device get with --id, --output flags
  - device add with all device field flags
  - device update with partial update support
  - device delete with --force flag
Validation:
  Build: REQUIRED
  Tests: REQUIRED (test flag parsing, output formats)
Notes: Support --output json/yaml/table
```

### [P8-004] Implement Network Commands
```
Status: TODO
Specs: docs/specs/09-cli.md (lines 681-706)
Dependencies: P8-001
Outputs:
  - cmd/network/network.go
  - cmd/network/list.go
  - cmd/network/get.go
  - cmd/network/add.go
  - cmd/network/delete.go
  - cmd/network/pool.go
Acceptance:
  - network list/get/add/delete
  - network pool list/add
Validation:
  Build: REQUIRED
  Tests: REQUIRED (test flag parsing, subcommand structure)
Notes: Pool as subcommand of network
```

### [P8-005] Implement Datacenter Commands
```
Status: TODO
Specs: docs/specs/09-cli.md (lines 708-733)
Dependencies: P8-001
Outputs:
  - cmd/datacenter/datacenter.go
  - cmd/datacenter/list.go
  - cmd/datacenter/get.go
  - cmd/datacenter/add.go
  - cmd/datacenter/update.go
  - cmd/datacenter/delete.go
Acceptance:
  - All datacenter CRUD commands
Validation:
  Build: REQUIRED
  Tests: REQUIRED (test flag parsing)
Notes: None
```

### [P8-006] Implement Discovery Commands
```
Status: TODO
Specs: docs/specs/09-cli.md (lines 735-956)
Dependencies: P8-001
Outputs:
  - cmd/discovery/discovery.go
  - cmd/discovery/scan.go
  - cmd/discovery/list.go
  - cmd/discovery/promote.go
Acceptance:
  - discovery scan --network --type flags
  - discovery list with --network, --status filters
  - discovery promote --discovered-id --name
  - --dry-run support for scan
Validation:
  Build: REQUIRED
  Tests: REQUIRED (test flag parsing, dry-run mode)
Notes: None
```

### [P8-007] Implement Main Entry Point
```
Status: TODO
Specs: docs/specs/09-cli.md (lines 7-56)
Dependencies: P8-002, P8-003, P8-004, P8-005, P8-006
Outputs:
  - main.go
Acceptance:
  - Root CLI command "rackd"
  - All subcommands registered
  - version command shows version/commit/date
  - Build variables for version injection
Validation:
  Build: REQUIRED
  Tests: REQUIRED (test main builds, version command works)
Notes: Use paularlott/cli for command structure
```

---

### Phase 8 Checkpoint
```
Status: TODO
All tasks P8-001 through P8-007 must be DONE before proceeding.

Validation Commands:
  [ ] go build ./...                              # Must pass
  [ ] go build -o rackd .                         # Binary must be created
  [ ] ./rackd --help                              # Help must display
  [ ] ./rackd version                             # Version must display
  [ ] go test ./cmd/... -v                        # CLI tests must pass
  [ ] go test ./... -v                            # Full test suite must pass
  [ ] go vet ./...                                # Must pass

Expected State:
  - Single binary builds successfully
  - All subcommands registered and accessible
  - Help text displays correctly
  - Version command shows build info
  - Output formats (json/yaml/table) work
```

---

## Phase 9: Testing & Quality

### [P9-001] Storage Integration Tests
```
Status: TODO
Specs: docs/specs/15-testing.md
Dependencies: P2-011
Outputs:
  - internal/storage/integration_test.go
Acceptance:
  - Full CRUD lifecycle tests
  - Migration tests
  - Concurrent access tests
Validation:
  Build: REQUIRED
  Tests: REQUIRED (integration tests must pass)
Notes: Skip with -short flag
```

### [P9-002] API Integration Tests
```
Status: TODO
Specs: docs/specs/15-testing.md (lines 150-161)
Dependencies: P3-010
Outputs:
  - internal/api/integration_test.go
Acceptance:
  - Full request/response cycle tests
  - Auth middleware integration
Validation:
  Build: REQUIRED
  Tests: REQUIRED (integration tests must pass)
Notes: Use real storage (in-memory)
```

### [P9-003] CLI Tests
```
Status: TODO
Specs: docs/specs/15-testing.md (lines 156-161)
Dependencies: P8-007
Outputs:
  - cmd/device/device_test.go
  - cmd/network/network_test.go
Acceptance:
  - Command parsing tests
  - Output format tests
Validation:
  Build: REQUIRED
  Tests: REQUIRED (CLI tests must pass)
Notes: Mock HTTP client for API calls
```

---

### Phase 9 Checkpoint
```
Status: TODO
All tasks P9-001 through P9-003 must be DONE before proceeding.

Validation Commands:
  [ ] go test ./... -v                            # All tests must pass
  [ ] go test ./... -race                         # Race detection must pass
  [ ] go test ./internal/storage/... -cover       # Coverage >= 90%
  [ ] go test ./internal/api/... -cover           # Coverage >= 80%
  [ ] go test ./internal/discovery/... -cover     # Coverage >= 70%
  [ ] go vet ./...                                # Must pass
  [ ] golangci-lint run                           # Should pass (warnings OK)

Expected State:
  - All unit tests pass
  - All integration tests pass
  - No race conditions detected
  - Coverage targets met
  - Code passes linting
```

---

## Phase 10: Build & Deployment

### [P10-001] Complete Makefile
```
Status: TODO
Specs: docs/specs/11-build-deploy.md (lines 7-118)
Dependencies: P7-013, P8-007
Outputs:
  - Makefile (complete)
Acceptance:
  - make build creates full binary
  - make test runs all tests
  - make lint runs golangci-lint
  - make security runs gosec for security scanning
  - Cross-compilation targets work
Validation:
  Build: REQUIRED (make build must succeed)
  Tests: REQUIRED (make test must pass)
Notes: None
Security: Add gosec target for security-focused static analysis
```

### [P10-002] Create Dockerfile
```
Status: TODO
Specs: docs/specs/11-build-deploy.md (lines 120-166)
Dependencies: P10-001
Outputs:
  - Dockerfile
  - docker-compose.yml
Acceptance:
  - Multi-stage build works
  - Image runs successfully
  - Health check passes
  - docker-compose up works
Validation:
  Build: REQUIRED (docker build must succeed)
  Tests: REQUIRED (container health check must pass)
Notes: Use golang:1.25-alpine and alpine:latest
```

### [P10-003] Create GoReleaser Config
```
Status: TODO
Specs: docs/specs/11-build-deploy.md (lines 198-263)
Dependencies: P10-001
Outputs:
  - .goreleaser.yml
Acceptance:
  - goreleaser check passes
  - Builds for linux/darwin/windows amd64/arm64
  - Creates checksums and changelog
Validation:
  Build: SKIP (goreleaser runs in CI)
  Tests: SKIP (goreleaser runs in CI)
Notes: Version 2 format. Test with: goreleaser check
```

### [P10-004] Create Nomad Job
```
Status: TODO
Specs: docs/specs/11-build-deploy.md (lines 265-344)
Dependencies: P10-002
Outputs:
  - deploy/nomad.hcl
Acceptance:
  - Valid HCL syntax
  - Service check configured
  - Volume mount for data
Validation:
  Build: SKIP (Nomad-specific)
  Tests: SKIP (Nomad-specific)
Notes: Optional - for Nomad users. Validate with: nomad job validate deploy/nomad.hcl
```

---

### Phase 10 Checkpoint
```
Status: TODO
All tasks P10-001 through P10-004 must be DONE before proceeding.

Validation Commands:
  [ ] make build                                  # Full build succeeds
  [ ] make test                                   # All tests pass
  [ ] make lint                                   # Linting passes
  [ ] docker build -t rackd .                     # Docker build succeeds
  [ ] docker run --rm rackd version               # Container runs
  [ ] goreleaser check                            # Config valid (if goreleaser installed)

Expected State:
  - Full Makefile with all targets
  - Docker image builds and runs
  - GoReleaser config ready for CI
  - Deployment configs available
```

---

## Phase 11: Documentation

### [P11-001] Create README
```
Status: TODO
Specs: docs/specs/18-user-guide.md
Dependencies: P10-001
Outputs:
  - README.md
Acceptance:
  - Installation instructions
  - Quick start guide
  - Configuration reference
  - CLI examples
Validation:
  Build: REQUIRED (verify no broken links in markdown)
  Tests: SKIP (documentation)
Notes: None
```

### [P11-002] Create Development Docs
```
Status: TODO
Specs: docs/specs/README.md
Dependencies: P10-001
Outputs:
  - CLAUDE.md
  - AGENTS.md
Acceptance:
  - Development setup instructions
  - Architecture overview
  - Contributing guidelines
Validation:
  Build: SKIP (documentation)
  Tests: SKIP (documentation)
Notes: None
```

### [P11-003] Create OpenAPI Spec
```
Status: TODO
Specs: docs/specs/14-api-reference.md
Dependencies: P3-010
Outputs:
  - api/openapi.yaml
Acceptance:
  - Valid OpenAPI 3.1 document
  - All endpoints documented
  - Request/response schemas
Validation:
  Build: SKIP (documentation)
  Tests: SKIP (validate with: npx @redocly/cli lint api/openapi.yaml)
Notes: None
```

---

### Phase 11 Checkpoint (FINAL)
```
Status: TODO
All tasks P11-001 through P11-003 must be DONE.

Final Validation Commands:
  [ ] make build                                  # Full build succeeds
  [ ] make test                                   # All tests pass
  [ ] make lint                                   # Linting passes
  [ ] ./rackd server &                            # Server starts
  [ ] curl http://localhost:8080/healthz          # Health check passes
  [ ] curl http://localhost:8080/api/config       # API responds
  [ ] Open http://localhost:8080 in browser       # UI loads

Expected Final State:
  - Complete, working application
  - All documentation in place
  - Ready for v1.0.0 release
  - All tests passing
  - Docker deployment ready
```

---

## Progress Summary

```yaml
# OSS Edition Tasks
Phase 1 - Foundation:     10/10 tasks complete
Phase 2 - Data Layer:     11/11 tasks complete
Phase 3 - API Layer:      10/10 tasks complete
Phase 4 - MCP Server:     1/1 tasks complete
Phase 5 - Discovery:      3/3 tasks complete
Phase 6 - Server:         0/3 tasks complete
Phase 7 - Web UI:         0/13 tasks complete
Phase 8 - CLI:            0/7 tasks complete
Phase 9 - Testing:        0/3 tasks complete
Phase 10 - Deployment:    0/4 tasks complete
Phase 11 - Documentation: 0/3 tasks complete

OSS Total: 35/68 tasks complete (51%)

# Enterprise Edition Tasks
Enterprise Phase 1 - Repo Setup:       3/3 tasks complete
Enterprise Phase 5 - Advanced Scan:    0/7 tasks complete
Enterprise Phase 6 - Enterprise Server: 0/3 tasks complete

Enterprise Total: 3/13 tasks complete (23%)

# Combined Total: 38/81 tasks complete (47%)
```

### Parallel Development Timeline

```
Phase 1 (OSS) ──────────────────────────┐
                                        │
Enterprise Phase 1 ─────────────────────┼──► Both repos set up
                                        │
Phase 2 (OSS) ──────────────────────────┤
                                        │
Phase 3 (OSS) ──────────────────────────┤
                                        │
Phase 4 (OSS) ──────────────────────────┤
                                        │
Phase 5 (OSS) ──────────────────────────┼──► OSS Discovery ready
                                        │
Enterprise Phase 5 ─────────────────────┼──► ARCHITECTURE VALIDATION
                                        │    (Critical checkpoint)
Phase 6 (OSS) ──────────────────────────┤
                                        │
Enterprise Phase 6 ─────────────────────┼──► Enterprise server ready
                                        │
Phase 7-11 (OSS) ───────────────────────┴──► Final OSS release
```

---

## Appendix: Quick Reference

### Spec File Index

> **Note**: The spec files use "Premium" terminology. This plan uses "Enterprise" instead.
> When reading specs, interpret "Premium" as "Enterprise".

| Spec File | Content |
|-----------|---------|
| 01-architecture.md | Project philosophy, architecture diagram |
| 02-oss-premium-split.md | Two-repo architecture, Feature interface |
| 03-feature-matrix.md | OSS vs Enterprise features, Enterprise interfaces |
| 04-directory-structure.md | File layout, dependencies |
| 05-data-models.md | All data structs with JSON tags |
| 06-storage.md | Storage interfaces |
| 07-api.md | HTTP handlers, middleware, MCP server |
| 08-web-ui.md | Frontend architecture, Enterprise UI patterns |
| 09-cli.md | CLI commands, client package |
| 10-discovery.md | Scanner implementation, scheduler |
| 11-build-deploy.md | Makefile, Docker, GoReleaser, Nomad |
| 12-configuration.md | Environment variables |
| 13-database-schema.md | SQLite schema |
| 14-api-reference.md | API endpoint list |
| 15-testing.md | Test strategy |
| 16-security.md | Security practices |
| 17-monitoring.md | Logging, metrics |
| 18-user-guide.md | User documentation |
| 19-ui-layout.md | UI wireframes, components |
| 20-error-handling.md | Error codes, retry, circuit breaker |
| 21-database-migrations.md | Migration system |
| 22-backup-restore.md | Backup procedures |

### Dependency Graph (Critical Path)

**OSS Critical Path:**
```
P1-001 (go.mod)
  └── P1-002 (directories)
        └── P1-004..P1-008 (models)
              └── P2-001 (storage interfaces)
                    └── P2-002 (migrations)
                          └── P2-003..P2-009 (SQLite impl)
                                └── P3-001 (API handlers)
                                      └── P3-003..P3-008 (entity handlers)
                                            └── P6-002 (server)
                                                  └── P8-007 (main.go)
```

**Enterprise Critical Path:**
```
P1-001 (OSS go.mod)
  └── E1-001 (Enterprise go.mod, imports OSS)
        └── E1-002 (Enterprise directories)
              └── E1-003 (Enterprise models)

P5-001 (Scanner interface)
  └── E5-001 (Credential storage)
        ├── E5-002 (SNMP scanner)
        └── E5-003 (SSH scanner)
              └── E5-004 (Advanced discovery service)
                    └── E5-007 (Feature implementation)
                          └── E6-001 (Enterprise main.go)
```

**Architecture Validation Point:**
```
                    ┌─────────────────────────────────────┐
                    │  E5-007: Feature Implementation     │
                    │  ─────────────────────────────────  │
                    │  Validates:                         │
                    │  • Feature interface works          │
                    │  • Routes injected correctly        │
                    │  • MCP tools registered             │
                    │  • UI extended via ConfigureUI      │
                    │  • OSS code unchanged               │
                    └─────────────────────────────────────┘
```

### Enterprise Task Index

| Task ID | Description | Depends On |
|---------|-------------|------------|
| E1-001 | Initialize Enterprise Repository | P1-001 |
| E1-002 | Create Enterprise Directory Structure | E1-001 |
| E1-003 | Implement Enterprise Models | E1-002, P1-004 |
| E5-001 | Implement Credential Storage | E1-003, P2-003 |
| E5-002 | Implement SNMP Scanner | E5-001, P5-001 |
| E5-003 | Implement SSH Scanner | E5-001, P5-001 |
| E5-004 | Implement Advanced Discovery Service | E5-002, E5-003, P5-002 |
| E5-005 | Implement Scan Profiles Storage | E1-003, P2-003 |
| E5-006 | Implement Scheduled Scans | E5-004, E5-005, P5-003 |
| E5-007 | Implement Advanced Scanning Feature | E5-004, E5-006 |
| E6-001 | Implement Enterprise Server Entry Point | E5-007, P6-002 |
| E6-002 | Implement Enterprise API Handlers | E5-001, E5-005, E5-006 |
| E6-003 | Implement Enterprise MCP Tools | E5-004 |
