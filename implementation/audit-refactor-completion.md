# Storage-Level Audit Refactor - COMPLETION GUIDE

## Current Status

✅ **Phase 1 Complete**: Interfaces and helpers
- Created `internal/audit/context.go` with context helpers
- Updated all storage interfaces to accept `context.Context`
- Added `auditLog()` helper method to SQLiteStorage
- Updated `CreateDevice()`, `UpdateDevice()`, `DeleteDevice()` as examples

## Completion Steps

### Step 1: Complete Storage Implementation (4-6 hours)

Update remaining methods in `internal/storage/sqlite.go`:

**Pattern to follow**:
```go
func (s *SQLiteStorage) CreateXXX(ctx context.Context, xxx *model.XXX) error {
    // ... existing logic ...
    
    if err := tx.Commit(); err != nil {
        return err
    }
    
    // Add audit logging
    s.auditLog(ctx, "create", "resource_name", xxx.ID, nil)
    return nil
}
```

**Methods to update** (copy this checklist):
```
Datacenters:
[ ] CreateDatacenter - Add: s.auditLog(ctx, "create", "datacenter", dc.ID, nil)
[ ] UpdateDatacenter - Add: s.auditLog(ctx, "update", "datacenter", dc.ID, nil)
[ ] DeleteDatacenter - Add: s.auditLog(ctx, "delete", "datacenter", id, nil)

Networks:
[ ] CreateNetwork - Add: s.auditLog(ctx, "create", "network", network.ID, nil)
[ ] UpdateNetwork - Add: s.auditLog(ctx, "update", "network", network.ID, nil)
[ ] DeleteNetwork - Add: s.auditLog(ctx, "delete", "network", id, nil)

Pools:
[ ] CreateNetworkPool - Add: s.auditLog(ctx, "create", "pool", pool.ID, nil)
[ ] UpdateNetworkPool - Add: s.auditLog(ctx, "update", "pool", pool.ID, nil)
[ ] DeleteNetworkPool - Add: s.auditLog(ctx, "delete", "pool", id, nil)

Relationships:
[ ] AddRelationship - Add: s.auditLog(ctx, "add", "relationship", parentID+":"+childID, nil)
[ ] RemoveRelationship - Add: s.auditLog(ctx, "remove", "relationship", parentID+":"+childID, nil)
[ ] UpdateRelationshipNotes - Add: s.auditLog(ctx, "update", "relationship", parentID+":"+childID, nil)

Discovery:
[ ] CreateDiscoveredDevice - Add: s.auditLog(ctx, "create", "discovered_device", device.ID, nil)
[ ] UpdateDiscoveredDevice - Add: s.auditLog(ctx, "update", "discovered_device", device.ID, nil)
[ ] DeleteDiscoveredDevice - Add: s.auditLog(ctx, "delete", "discovered_device", id, nil)
[ ] PromoteDiscoveredDevice - Add: s.auditLog(ctx, "promote", "discovered_device", discoveredID, nil)
[ ] CreateDiscoveryScan - Add: s.auditLog(ctx, "create", "discovery_scan", scan.ID, nil)
[ ] UpdateDiscoveryScan - Add: s.auditLog(ctx, "update", "discovery_scan", scan.ID, nil)
[ ] DeleteDiscoveryScan - Add: s.auditLog(ctx, "delete", "discovery_scan", id, nil)
[ ] SaveDiscoveryRule - Add: s.auditLog(ctx, "save", "discovery_rule", rule.ID, nil)
[ ] DeleteDiscoveryRule - Add: s.auditLog(ctx, "delete", "discovery_rule", id, nil)
```

**Also update** `internal/storage/bulk.go`:
```
[ ] BulkCreateDevices
[ ] BulkUpdateDevices
[ ] BulkDeleteDevices
[ ] BulkAddTags
[ ] BulkRemoveTags
[ ] BulkCreateNetworks
[ ] BulkDeleteNetworks
```

### Step 2: Update API Handlers (3-4 hours)

Update all handlers in `internal/api/*_handlers.go` to create and pass audit context.

**Pattern to follow**:
```go
func (h *Handler) createDevice(w http.ResponseWriter, r *http.Request) {
    var device model.Device
    if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
        h.writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
        return
    }
    
    // Create audit context
    ctx := h.auditContext(r)
    
    // Pass context to storage
    if err := h.store.CreateDevice(ctx, &device); err != nil {
        h.internalError(w, err)
        return
    }
    
    h.writeJSON(w, http.StatusCreated, device)
}
```

**Add helper method to Handler**:
```go
func (h *Handler) auditContext(r *http.Request) context.Context {
    auditCtx := &audit.Context{
        IPAddress: getClientIP(r),
        Source:    "api",
    }
    
    // Extract user from API key context
    if apiKey, ok := r.Context().Value(APIKeyContextKey).(*model.APIKey); ok {
        auditCtx.UserID = apiKey.ID
        auditCtx.Username = apiKey.Name
    }
    
    return audit.WithContext(r.Context(), auditCtx)
}
```

**Files to update**:
```
[ ] internal/api/device_handlers.go (create, update, delete, bulk operations)
[ ] internal/api/datacenter_handlers.go (create, update, delete)
[ ] internal/api/network_handlers.go (create, update, delete, bulk operations)
[ ] internal/api/pool_handlers.go (create, update, delete)
[ ] internal/api/relationship_handlers.go (add, remove, update)
[ ] internal/api/discovery_handlers.go (all mutating operations)
```

### Step 3: Update MCP Server (1-2 hours)

Update `internal/mcp/server.go` to pass audit context.

**Add helper**:
```go
func (s *Server) auditContext() context.Context {
    return audit.WithContext(context.Background(), &audit.Context{
        Source: "mcp",
    })
}
```

**Update all tool handlers**:
```go
func (s *Server) handleCreateDevice(params json.RawMessage) (interface{}, error) {
    var device model.Device
    if err := json.Unmarshal(params, &device); err != nil {
        return nil, err
    }
    
    ctx := s.auditContext()
    return nil, s.store.CreateDevice(ctx, &device)
}
```

### Step 4: Update CLI Commands (2-3 hours)

Update all CLI commands in `cmd/*/` to pass context.

**Pattern**:
```go
func addDevice(ctx context.Context, cmd *cli.Command) error {
    store, err := storage.NewExtendedStorage(dataDir)
    if err != nil {
        return err
    }
    defer store.Close()
    
    device := &model.Device{
        Name: cmd.GetString("name"),
        // ... other fields ...
    }
    
    // Create audit context for CLI
    auditCtx := audit.WithContext(ctx, &audit.Context{
        Source: "cli",
    })
    
    return store.CreateDevice(auditCtx, device)
}
```

**Files to update**:
```
[ ] cmd/device/add.go
[ ] cmd/device/update.go
[ ] cmd/device/delete.go
[ ] cmd/datacenter/add.go
[ ] cmd/datacenter/update.go
[ ] cmd/datacenter/delete.go
[ ] cmd/network/add.go
[ ] cmd/network/update.go
[ ] cmd/network/delete.go
[ ] cmd/discovery/*.go
[ ] cmd/import/import.go (uses bulk operations)
```

### Step 5: Update Tests (2-3 hours)

Update all tests to pass `context.Background()` or `context.TODO()`.

**Pattern**:
```go
func TestCreateDevice(t *testing.T) {
    store := newTestStorage(t)
    defer store.Close()
    
    device := &model.Device{Name: "test"}
    
    // Pass context
    err := store.CreateDevice(context.Background(), device)
    if err != nil {
        t.Fatalf("Failed to create device: %v", err)
    }
}
```

**Files to update**:
```
[ ] internal/storage/*_test.go
[ ] internal/api/*_test.go
[ ] cmd/*/*_test.go
```

### Step 6: Remove Old Middleware (30 min)

Once everything is working:

1. Remove `internal/api/audit_middleware.go`
2. Remove middleware registration from `internal/server/server.go`:
   ```go
   // Remove these lines:
   if cfg.AuditEnabled {
       httpHandler = api.AuditMiddleware(store)(httpHandler)
   }
   ```
3. Update documentation to reflect storage-level auditing

### Step 7: Verification (1 hour)

Test all three access methods:

```bash
# Test API
curl -X POST http://localhost:8080/api/devices -d '{"name":"test"}'
curl http://localhost:8080/api/audit | jq '.[] | select(.source=="api")'

# Test MCP
# (use MCP client to create device)
curl http://localhost:8080/api/audit | jq '.[] | select(.source=="mcp")'

# Test CLI
rackd device add --name test
curl http://localhost:8080/api/audit | jq '.[] | select(.source=="cli")'
```

## Quick Reference Commands

### Find all methods that need updating:
```bash
grep -n "^func (s \*SQLiteStorage) \(Create\|Update\|Delete\|Add\|Remove\|Save\|Promote\)" internal/storage/sqlite.go
```

### Find all handler methods:
```bash
grep -n "^func (h \*Handler)" internal/api/*_handlers.go | grep -E "(create|update|delete|add|remove)"
```

### Find all CLI commands:
```bash
find cmd -name "*.go" -exec grep -l "store\.\(Create\|Update\|Delete\)" {} \;
```

### Run tests:
```bash
go test ./internal/storage/... -v
go test ./internal/api/... -v
go test ./cmd/... -v
```

## Estimated Time

- Storage implementation: 4-6 hours
- API handlers: 3-4 hours
- MCP server: 1-2 hours
- CLI commands: 2-3 hours
- Tests: 2-3 hours
- Cleanup & verification: 1-2 hours

**Total: 13-20 hours**

## Tips

1. **Work in order**: Storage → API → MCP → CLI → Tests
2. **Test incrementally**: After each file, run its tests
3. **Use search/replace**: Many changes are mechanical
4. **Keep old middleware**: Until everything is migrated
5. **Git commits**: Commit after each major section

## Need Help?

If you get stuck:
1. Check the example implementations (CreateDevice, UpdateDevice, DeleteDevice)
2. Run `go build` frequently to catch signature mismatches
3. Use `grep` to find all call sites that need updating
4. Tests will fail if signatures don't match - use that as a checklist

Good luck! The refactor is straightforward but requires attention to detail.
