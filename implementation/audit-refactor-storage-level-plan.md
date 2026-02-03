# Storage-Level Audit Refactor Plan

## Overview

Move audit logging from API middleware to storage level to capture mutations from all entry points:
- API endpoints
- MCP server tools
- CLI commands
- Discovery scanner
- Scheduled workers
- Any future entry points

**Estimated Total Time:** 16-24 hours

## Current State

**Audited Entry Points:**
- ✅ API endpoints (POST/PUT/DELETE to `/api/*`) via `internal/api/audit_middleware.go`
- ✅ CLI commands (they call the API)

**Bypassed Entry Points:**
- ❌ MCP server tools - directly call storage
- ❌ Discovery scanner - creates/updates discovered devices
- ❌ Advanced discovery service
- ❌ Scheduled scan worker

## Architecture

### Current Flow
```
API Request → Audit Middleware → Handler → Storage → Database
                                           ↑
                                    Audit Log (middleware)
```

### Target Flow
```
API Request → Handler (create audit ctx) → Storage (extract ctx) → Database
                                                   ↑
                                              Audit Log (storage level)

MCP Tool → Handler (create audit ctx) → Storage (extract ctx) → Database
                                              ↑
                                         Audit Log (storage level)

Discovery → Worker (create audit ctx) → Storage (extract ctx) → Database
                                              ↑
                                         Audit Log (storage level)
```

### Key Components

1. **audit.Context** - Carries audit info via context
   - `UserID` - API key ID
   - `Username` - API key name or user
   - `IPAddress` - Client IP
   - `Source` - Entry point: "api", "mcp", "cli", "discovery", "scheduler"

2. **audit.WithContext()** - Wraps context with audit info

3. **audit.FromContext()** - Extracts audit info from context

4. **SQLiteStorage.auditLog()** - Helper to write audit logs asynchronously

---

## Phase 0: Database Migration (1-2 hours)

### Tasks
- [ ] Create migration `migrateAddAuditSourceUp/Down` in `internal/storage/migrations.go`
  - [ ] Add migration to migrations array with version "20260203170000"
  - [ ] Implement `migrateAddAuditSourceUp(ctx, tx)`:
    - [ ] Add `ALTER TABLE audit_logs ADD COLUMN source TEXT`
    - [ ] Add `CREATE INDEX idx_audit_logs_source ON audit_logs(source)`
  - [ ] Implement `migrateAddAuditSourceDown(ctx, tx)`:
    - [ ] Add `DROP INDEX IF EXISTS idx_audit_logs_source`
    - [ ] Add `ALTER TABLE audit_logs DROP COLUMN source` (if SQLite supports, else recreate table)
- [ ] Update `internal/model/audit.go`:
  - [ ] Add `Source string` field to `AuditLog` struct
  - [ ] Update documentation comments
- [ ] Test migration:
  - [ ] Run migration on test database
  - [ ] Verify new column exists
  - [ ] Verify index created
  - [ ] Test rollback

### Files to Create/Modify
- `internal/storage/migrations.go` (modify)
- `internal/model/audit.go` (modify)

---

## Phase 1: Foundation (1-2 hours)

### Tasks
- [ ] Create `internal/audit/context.go`:
  - [ ] Define `Context` struct with UserID, Username, IPAddress, Source fields
  - [ ] Implement `WithContext(parentCtx context.Context, auditCtx *Context) context.Context`
  - [ ] Implement `FromContext(ctx context.Context) (*Context, bool)`
  - [ ] Implement `MustFromContext(ctx context.Context) *Context`
  - [ ] Add package documentation
- [ ] Update `internal/storage/storage.go`:
  - [ ] Add `context.Context` as first parameter to all mutating methods:
    - [ ] DeviceStorage: CreateDevice, UpdateDevice, DeleteDevice
    - [ ] DatacenterStorage: CreateDatacenter, UpdateDatacenter, DeleteDatacenter
    - [ ] NetworkStorage: CreateNetwork, UpdateNetwork, DeleteNetwork
    - [ ] NetworkPoolStorage: CreateNetworkPool, UpdateNetworkPool, DeleteNetworkPool
    - [ ] RelationshipStorage: AddRelationship, RemoveRelationship, UpdateRelationshipNotes
    - [ ] DiscoveryStorage: CreateDiscoveredDevice, UpdateDiscoveredDevice, DeleteDiscoveredDevice,
                           PromoteDiscoveredDevice, DeleteDiscoveredDevicesByNetwork,
                           CreateDiscoveryScan, UpdateDiscoveryScan, DeleteDiscoveryScan,
                           SaveDiscoveryRule, DeleteDiscoveryRule
    - [ ] BulkOperations: BulkCreateDevices, BulkUpdateDevices, BulkDeleteDevices,
                         BulkAddTags, BulkRemoveTags, BulkCreateNetworks, BulkDeleteNetworks
- [ ] Update `internal/storage/sqlite.go`:
  - [ ] Update all mutating method signatures to accept `ctx context.Context`
  - [ ] Add `auditLog()` helper method:
    - [ ] Extract audit context using `audit.FromContext()`
    - [ ] Create AuditLog entry with all fields
    - [ ] Write asynchronously via goroutine
    - [ ] Handle nil context gracefully (no audit)
  - [ ] Update `CreateDevice()` to call `auditLog(ctx, "create", "device", device.ID, device)`
  - [ ] Update `UpdateDevice()` to call `auditLog(ctx, "update", "device", device.ID, device)`
  - [ ] Update `DeleteDevice()` to call `auditLog(ctx, "delete", "device", id, nil)`
  - [ ] Update `CreateDatacenter()` to call `auditLog(ctx, "create", "datacenter", dc.ID, dc)`
  - [ ] Update `UpdateDatacenter()` to call `auditLog(ctx, "update", "datacenter", dc.ID, dc)`
  - [ ] Update `DeleteDatacenter()` to call `auditLog(ctx, "delete", "datacenter", id, nil)`
  - [ ] Update `CreateNetwork()` to call `auditLog(ctx, "create", "network", network.ID, network)`
  - [ ] Update `UpdateNetwork()` to call `auditLog(ctx, "update", "network", network.ID, network)`
  - [ ] Update `DeleteNetwork()` to call `auditLog(ctx, "delete", "network", id, nil)`
  - [ ] Update `CreateNetworkPool()` to call `auditLog(ctx, "create", "pool", pool.ID, pool)`
  - [ ] Update `UpdateNetworkPool()` to call `auditLog(ctx, "update", "pool", pool.ID, pool)`
  - [ ] Update `DeleteNetworkPool()` to call `auditLog(ctx, "delete", "pool", id, nil)`
  - [ ] Update `AddRelationship()` to call `auditLog(ctx, "add", "relationship", parentID+":"+childID, nil)`
  - [ ] Update `RemoveRelationship()` to call `auditLog(ctx, "remove", "relationship", parentID+":"+childID, nil)`
  - [ ] Update `UpdateRelationshipNotes()` to call `auditLog(ctx, "update", "relationship", parentID+":"+childID, nil)`

### Files to Create/Modify
- `internal/audit/context.go` (create)
- `internal/storage/storage.go` (modify)
- `internal/storage/sqlite.go` (modify)

---

## Phase 2: Storage Layer - Discovery Methods (1-2 hours)

### Tasks
- [ ] Update `internal/storage/discovery_sqlite.go`:
  - [ ] Update `CreateDiscoveredDevice()` to call `auditLog(ctx, "create", "discovered_device", device.ID, device)`
  - [ ] Update `UpdateDiscoveredDevice()` to call `auditLog(ctx, "update", "discovered_device", device.ID, device)`
  - [ ] Update `DeleteDiscoveredDevice()` to call `auditLog(ctx, "delete", "discovered_device", id, nil)`
  - [ ] Update `DeleteDiscoveredDevicesByNetwork()` to call `auditLog(ctx, "delete", "discovered_device", "network:"+networkID, nil)`
  - [ ] Update `PromoteDiscoveredDevice()` to call `auditLog(ctx, "promote", "discovered_device", discoveredID, nil)`
  - [ ] Update `CreateDiscoveryScan()` to call `auditLog(ctx, "create", "discovery_scan", scan.ID, scan)`
  - [ ] Update `UpdateDiscoveryScan()` to call `auditLog(ctx, "update", "discovery_scan", scan.ID, scan)`
  - [ ] Update `DeleteDiscoveryScan()` to call `auditLog(ctx, "delete", "discovery_scan", id, nil)`
  - [ ] Update `SaveDiscoveryRule()` to call `auditLog(ctx, "save", "discovery_rule", rule.ID, rule)`
  - [ ] Update `DeleteDiscoveryRule()` to call `auditLog(ctx, "delete", "discovery_rule", id, nil)`

### Files to Modify
- `internal/storage/discovery_sqlite.go` (modify)

---

## Phase 3: Bulk Operations (1 hour)

### Tasks
- [ ] Update `internal/storage/bulk.go`:
  - [ ] Add context parameter to all bulk method signatures
  - [ ] Update `BulkCreateDevices()` to call `auditLog(ctx, "bulk_create", "device", "", map{"count": len(devices)})`
  - [ ] Update `BulkUpdateDevices()` to call `auditLog(ctx, "bulk_update", "device", "", map{"count": len(devices)})`
  - [ ] Update `BulkDeleteDevices()` to call `auditLog(ctx, "bulk_delete", "device", "", map{"count": len(ids)})`
  - [ ] Update `BulkAddTags()` to call `auditLog(ctx, "bulk_add_tags", "device", "", map{"count": len(deviceIDs)})`
  - [ ] Update `BulkRemoveTags()` to call `auditLog(ctx, "bulk_remove_tags", "device", "", map{"count": len(deviceIDs)})`
  - [ ] Update `BulkCreateNetworks()` to call `auditLog(ctx, "bulk_create", "network", "", map{"count": len(networks)})`
  - [ ] Update `BulkDeleteNetworks()` to call `auditLog(ctx, "bulk_delete", "network", "", map{"count": len(ids)})`

### Files to Modify
- `internal/storage/bulk.go` (modify)

---

## Phase 4: API Handlers (3-4 hours)

### Tasks
- [ ] Update `internal/api/handlers.go`:
  - [ ] Add `auditContext(r *http.Request) context.Context` helper method
  - [ ] Extract user info from API key context
  - [ ] Extract IP address from request
  - [ ] Set source to "api"
- [ ] Update `internal/api/device_handlers.go`:
  - [ ] Update `createDevice()` to call `h.auditContext(r)` and pass to `CreateDevice()`
  - [ ] Update `updateDevice()` to call `h.auditContext(r)` and pass to `UpdateDevice()`
  - [ ] Update `deleteDevice()` to call `h.auditContext(r)` and pass to `DeleteDevice()`
  - [ ] Update bulk operations to pass context
- [ ] Update `internal/api/datacenter_handlers.go`:
  - [ ] Update `createDatacenter()` to pass context
  - [ ] Update `updateDatacenter()` to pass context
  - [ ] Update `deleteDatacenter()` to pass context
- [ ] Update `internal/api/network_handlers.go`:
  - [ ] Update `createNetwork()` to pass context
  - [ ] Update `updateNetwork()` to pass context
  - [ ] Update `deleteNetwork()` to pass context
  - [ ] Update bulk operations to pass context
- [ ] Update `internal/api/pool_handlers.go`:
  - [ ] Update `createNetworkPool()` to pass context
  - [ ] Update `updateNetworkPool()` to pass context
  - [ ] Update `deleteNetworkPool()` to pass context
- [ ] Update `internal/api/relationship_handlers.go`:
  - [ ] Update `addRelationship()` to pass context
  - [ ] Update `removeRelationship()` to pass context
  - [ ] Update `updateRelationship()` to pass context
- [ ] Update `internal/api/discovery_handlers.go`:
  - [ ] Update `startScan()` to pass context
  - [ ] Update `promoteDevice()` to pass context
  - [ ] Update `saveDiscoveryRule()` to pass context
  - [ ] Update `deleteDiscoveryRule()` to pass context

### Files to Modify
- `internal/api/handlers.go` (modify - add helper)
- `internal/api/device_handlers.go` (modify)
- `internal/api/datacenter_handlers.go` (modify)
- `internal/api/network_handlers.go` (modify)
- `internal/api/pool_handlers.go` (modify)
- `internal/api/relationship_handlers.go` (modify)
- `internal/api/discovery_handlers.go` (modify)

---

## Phase 5: MCP Server (1-2 hours)

### Tasks
- [ ] Update `internal/mcp/server.go`:
  - [ ] Add `auditContext() context.Context` helper method
  - [ ] Set source to "mcp"
  - [ ] Update `handleDeviceSave()` to call `s.auditContext()` and pass to `CreateDevice()/UpdateDevice()`
  - [ ] Update `handleDeviceDelete()` to call `s.auditContext()` and pass to `DeleteDevice()`
  - [ ] Update `handleAddRelationship()` to call `s.auditContext()` and pass to `AddRelationship()`
  - [ ] Update `handleDatacenterSave()` to call `s.auditContext()` and pass to `CreateDatacenter()/UpdateDatacenter()`
  - [ ] Update `handleNetworkSave()` to call `s.auditContext()` and pass to `CreateNetwork()/UpdateNetwork()`
  - [ ] Update `handleStartScan()` to call `s.auditContext()` and pass to `CreateDiscoveryScan()`
  - [ ] Update `handlePromoteDevice()` to call `s.auditContext()` and pass to `CreateDevice()` and `PromoteDiscoveredDevice()`
  - [ ] Extract API key info for user attribution

### Files to Modify
- `internal/mcp/server.go` (modify)

---

## Phase 6: CLI Commands (2-3 hours)

### Tasks
- [ ] Create `internal/cli/audit_context.go`:
  - [ ] Add `AuditContext(ctx context.Context) context.Context` helper
  - [ ] Set source to "cli"
- [ ] Update `cmd/device/add.go`:
  - [ ] Add audit context
  - [ ] Pass to `CreateDevice()`
- [ ] Update `cmd/device/update.go`:
  - [ ] Add audit context
  - [ ] Pass to `UpdateDevice()`
- [ ] Update `cmd/device/delete.go`:
  - [ ] Add audit context
  - [ ] Pass to `DeleteDevice()`
- [ ] Update `cmd/datacenter/add.go`:
  - [ ] Add audit context
  - [ ] Pass to `CreateDatacenter()`
- [ ] Update `cmd/datacenter/update.go`:
  - [ ] Add audit context
  - [ ] Pass to `UpdateDatacenter()`
- [ ] Update `cmd/datacenter/delete.go`:
  - [ ] Add audit context
  - [ ] Pass to `DeleteDatacenter()`
- [ ] Update `cmd/network/add.go`:
  - [ ] Add audit context
  - [ ] Pass to `CreateNetwork()`
- [ ] Update `cmd/network/update.go`:
  - [ ] Add audit context
  - [ ] Pass to `UpdateNetwork()`
- [ ] Update `cmd/network/delete.go`:
  - [ ] Add audit context
  - [ ] Pass to `DeleteNetwork()`
- [ ] Update `cmd/discovery/scan.go`:
  - [ ] Add audit context
  - [ ] Pass to `CreateDiscoveryScan()`
- [ ] Update `cmd/discovery/promote.go`:
  - [ ] Add audit context
  - [ ] Pass to `PromoteDiscoveredDevice()`
- [ ] Update `cmd/import/import.go`:
  - [ ] Add audit context
  - [ ] Pass to bulk operations

### Files to Create/Modify
- `internal/cli/audit_context.go` (create)
- `cmd/device/add.go` (modify)
- `cmd/device/update.go` (modify)
- `cmd/device/delete.go` (modify)
- `cmd/datacenter/add.go` (modify)
- `cmd/datacenter/update.go` (modify)
- `cmd/datacenter/delete.go` (modify)
- `cmd/network/add.go` (modify)
- `cmd/network/update.go` (modify)
- `cmd/network/delete.go` (modify)
- `cmd/discovery/scan.go` (modify)
- `cmd/discovery/promote.go` (modify)
- `cmd/import/import.go` (modify)

---

## Phase 7: Discovery & Worker (1-2 hours)

### Tasks
- [ ] Update `internal/discovery/scanner.go`:
  - [ ] Add context parameter to `Scan()` method
  - [ ] Pass context to `CreateDiscoveryScan()`
  - [ ] Pass context to `UpdateDiscoveryScan()`
  - [ ] Pass context to `CreateDiscoveredDevice()`
  - [ ] Pass context to `UpdateDiscoveredDevice()`
- [ ] Update `internal/discovery/advanced.go`:
  - [ ] Add context parameter to `ScanAdvanced()` method
  - [ ] Pass context to all storage calls
- [ ] Update `internal/worker/scheduled.go`:
  - [ ] Create audit context for scheduled operations
  - [ ] Set source to "scheduler"
  - [ ] Pass context to storage calls
- [ ] Update API discovery handlers to pass context to scanner

### Files to Modify
- `internal/discovery/scanner.go` (modify)
- `internal/discovery/advanced.go` (modify)
- `internal/worker/scheduled.go` (modify)
- `internal/api/discovery_handlers.go` (modify - pass context to scanner)

---

## Phase 8: Tests (2-3 hours)

### Tasks
- [ ] Update `internal/storage/*_test.go`:
  - [ ] Pass `context.Background()` to all storage method calls
  - [ ] Add test for `auditLog()` helper
  - [ ] Add test for audit context propagation
  - [ ] Test that operations without audit context don't fail
- [ ] Update `internal/api/*_test.go`:
  - [ ] Update handlers to work with context parameters
  - [ ] Test that audit context is created correctly
  - [ ] Test that user info is extracted from API key
- [ ] Update `internal/mcp/server_test.go`:
  - [ ] Update handlers to work with context parameters
  - [ ] Test that audit context is created for MCP
  - [ ] Test that source is "mcp"
- [ ] Update `cmd/*/*_test.go`:
  - [ ] Pass context to storage calls
  - [ ] Test that source is "cli"

### Files to Modify
- All test files in `internal/storage/`, `internal/api/`, `internal/mcp/`, `cmd/`

---

## Phase 9: Cleanup (30 min)

### Tasks
- [ ] Remove `internal/api/audit_middleware.go`
- [ ] Update `internal/server/server.go`:
  - [ ] Remove `AuditMiddleware` registration
- [ ] Update `docs/audit.md`:
  - [ ] Document storage-level audit architecture
  - [ ] Add examples of different sources
  - [ ] Update API documentation
  - [ ] Update CLI documentation
- [ ] Remove or update `implementation/audit-refactor-completion.md`

### Files to Modify
- `internal/api/audit_middleware.go` (delete)
- `internal/server/server.go` (modify)
- `docs/audit.md` (modify)
- `implementation/audit-refactor-completion.md` (delete or update)

---

## Phase 10: Verification (1 hour)

### Tasks
- [ ] Test API mutations:
  - [ ] Create device via API
  - [ ] Verify audit log has source="api"
  - [ ] Verify user info is correct
  - [ ] Verify IP address is correct
- [ ] Test MCP mutations:
  - [ ] Create device via MCP
  - [ ] Verify audit log has source="mcp"
  - [ ] Verify user info is correct
- [ ] Test CLI mutations:
  - [ ] Create device via CLI
  - [ ] Verify audit log has source="cli"
- [ ] Test discovery mutations:
  - [ ] Run discovery scan
  - [ ] Verify audit logs for discovery_scan
  - [ ] Verify audit logs for discovered_device
  - [ ] Verify source="discovery"
- [ ] Test scheduled mutations:
  - [ ] Create scheduled scan
  - [ ] Wait for execution
  - [ ] Verify audit logs have source="scheduler"
- [ ] Test bulk operations:
  - [ ] Run bulk create
  - [ ] Verify audit log has count information
- [ ] Run all existing tests:
  - [ ] `go test ./internal/storage/...`
  - [ ] `go test ./internal/api/...`
  - [ ] `go test ./internal/mcp/...`
  - [ ] `go test ./cmd/...`
- [ ] Run integration tests:
  - [ ] `go test ./internal/api/integration_test.go`
- [ ] Manual testing with running server

---

## Rollback Plan

If issues arise during implementation:

### Quick Rollback (Stop after any phase)
- Keep old middleware active
- Storage-level audit will be additional, not replacement
- Can disable storage-level audit temporarily

### Full Rollback
- Remove context parameters from storage interfaces
- Remove `auditLog()` calls from storage methods
- Revert handler changes
- Keep old middleware
- Remove migration (or mark as rollback needed)

---

## Success Criteria

- [ ] All entry points (API, MCP, CLI, discovery, scheduler) create audit logs
- [ ] Audit logs correctly identify source (api/mcp/cli/discovery/scheduler)
- [ ] User info is correctly captured for authenticated operations
- [ ] IP address is captured for API operations
- [ ] All existing tests pass
- [ ] No performance regression (audit writes remain async)
- [ ] Documentation updated

---

## Notes

### Performance Considerations
- Audit writes should remain asynchronous (goroutine)
- Don't block on audit log creation
- Failed audit writes should be logged but not fail operations
- Consider batching for bulk operations

### Transaction Handling
- Audit should be logged even if transaction rolls back
- Write audit after transaction commit, not before
- Use separate goroutine for audit writes

### Context Propagation
- Always use `audit.WithContext()` to wrap context
- Always use `audit.FromContext()` to extract context
- Handle missing audit context gracefully (no crash, no audit)
- Use `audit.MustFromContext()` only when context is guaranteed

### Testing Strategy
- Test each phase independently
- Commit after each phase for easy rollback
- Run tests frequently during implementation
- Use integration tests to verify end-to-end flow

---

## Commands

### Find all methods that need updating:
```bash
grep -rn "^func (s \*SQLiteStorage) \(Create\|Update\|Delete\|Add\|Remove\|Save\|Promote\)" internal/storage/
```

### Find all handler methods:
```bash
grep -rn "^func (h \*Handler).*\(create\|update\|delete\|add\|remove\|save\|promote\)" internal/api/*_handlers.go
```

### Find all CLI commands that mutate:
```bash
find cmd -name "*.go" -exec grep -l "store\.\(Create\|Update\|Delete\)" {} \;
```

### Run specific test packages:
```bash
go test ./internal/storage/... -v
go test ./internal/api/... -v
go test ./internal/mcp/... -v
go test ./cmd/... -v
```

### Build to catch signature mismatches:
```bash
go build ./...
```

---

## Progress Log

### Phase 0: Database Migration
- Started: [date]
- Completed: [date]
- Notes:

### Phase 1: Foundation
- Started: [date]
- Completed: [date]
- Notes:

### Phase 2: Storage Layer - Discovery Methods
- Started: [date]
- Completed: [date]
- Notes:

### Phase 3: Bulk Operations
- Started: [date]
- Completed: [date]
- Notes:

### Phase 4: API Handlers
- Started: [date]
- Completed: [date]
- Notes:

### Phase 5: MCP Server
- Started: [date]
- Completed: [date]
- Notes:

### Phase 6: CLI Commands
- Started: [date]
- Completed: [date]
- Notes:

### Phase 7: Discovery & Worker
- Started: [date]
- Completed: [date]
- Notes:

### Phase 8: Tests
- Started: [date]
- Completed: [date]
- Notes:

### Phase 9: Cleanup
- Started: [date]
- Completed: [date]
- Notes:

### Phase 10: Verification
- Started: [date]
- Completed: [date]
- Notes:

---

## References

- `docs/audit.md` - Current audit documentation
- `implementation/audit-refactor-completion.md` - Original completion guide
- `internal/api/audit_middleware.go` - Current middleware implementation
- `internal/storage/audit_sqlite.go` - Current audit storage implementation
- `internal/model/audit.go` - Audit data models
