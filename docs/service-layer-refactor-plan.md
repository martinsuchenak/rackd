# Service Layer Refactor Plan

## Problem

The MCP server (14 tools) directly accesses storage without RBAC checks. Business logic (validation, multi-step orchestration, RBAC enforcement) currently lives in API handlers and would need to be duplicated in MCP tools. The audit CLI also directly accesses storage, bypassing all auth/RBAC. A shared service layer centralizes business logic so all entrypoints (API, MCP, CLI, workers) share the same rules.

## Architecture

```
Transport Layer              Service Layer                Storage Layer
─────────────────           ─────────────────            ─────────────────
API handlers      ──┐       DeviceService        ──┐     DeviceStorage
MCP tools         ──┼──►    UserService          ──┼──►  UserStorage
CLI (audit)       ──┤       RoleService          ──┤     RBACStorage
Workers           ──┘       DiscoveryService     ──┘     DiscoveryStorage
                            NetworkService               NetworkStorage
                            AuthService                  SessionManager
                            ... (per-resource)           ...
```

Transport adapters (API handlers, MCP tools, CLI commands) become thin:
- Parse input (HTTP request, MCP tool params, CLI flags)
- Build a `Caller` identity and inject it into context
- Call the appropriate service method
- Format the response (JSON, MCP response, terminal output)

The service layer owns:
- Input validation
- RBAC permission checks
- Multi-step orchestration (e.g., create user + assign role)
- Audit context enrichment
- Delegating to storage for persistence

## Package Structure

```
internal/service/
    caller.go          - Caller type, context helpers, SystemContext()
    errors.go          - Service error types (ErrForbidden, ErrNotFound, ValidationErrors, etc.)
    rbac.go            - requirePermission() helper
    audit.go           - enrichAuditCtx() bridges Caller -> audit.Context
    services.go        - Services registry struct, NewServices() constructor
    device.go          - DeviceService (CRUD + search)
    datacenter.go      - DatacenterService
    network.go         - NetworkService
    pool.go            - PoolService
    relationship.go    - RelationshipService
    discovery.go       - DiscoveryService (incl. promote orchestration)
    user.go            - UserService (validation, password hashing, role assignment)
    role.go            - RoleService (system role protection, permission management)
    auth.go            - AuthService (login credential verification, session creation, getCurrentUser)
    audit_svc.go       - AuditService (list, export)
    apikey.go          - APIKeyService
    bulk.go            - BulkService
```

## Key Design Decisions

### 1. Caller Identity (context-based)

The caller's identity is carried in `context.Context`, not as an explicit parameter. This keeps service method signatures clean and aligns with the existing `audit.Context` pattern.

```go
// internal/service/caller.go
package service

type CallerType int

const (
    CallerTypeAnonymous CallerType = iota
    CallerTypeUser       // Session-based user (from API web UI)
    CallerTypeAPIKey     // API key (from API or MCP)
    CallerTypeSystem     // Workers, bootstrap, tests
)

type Caller struct {
    Type      CallerType
    UserID    string     // empty for system/anonymous
    Username  string
    IPAddress string     // empty for non-HTTP callers
    Source    string     // "api", "mcp", "cli", "worker", "test"
}

func (c *Caller) IsSystem() bool {
    return c.Type == CallerTypeSystem
}

// Context helpers
func WithCaller(ctx context.Context, c *Caller) context.Context
func CallerFrom(ctx context.Context) *Caller
func SystemContext(ctx context.Context, source string) context.Context
```

**How callers are created:**
- **API session auth**: Middleware extracts session from cookie, creates `Caller{Type: CallerTypeUser, UserID: session.UserID, Username: session.Username, Source: "api"}`
- **API key auth**: Middleware validates Bearer token, creates `Caller{Type: CallerTypeAPIKey, UserID: key.ID, Username: key.Name, Source: "api"}`
- **MCP auth**: HandleRequest validates Bearer token, creates `Caller{Type: CallerTypeAPIKey, ..., Source: "mcp"}`
- **Workers/bootstrap**: Use `SystemContext(ctx, "worker")` which bypasses RBAC
- **Tests**: Use `SystemContext(ctx, "test")` for setup, specific `Caller` for RBAC tests

### 2. RBAC Enforcement in Service Layer

A shared helper function checks RBAC at the top of each service method. System callers bypass the check.

```go
// internal/service/rbac.go
package service

var (
    ErrForbidden       = errors.New("forbidden")
    ErrUnauthenticated = errors.New("unauthenticated")
)

type PermissionChecker interface {
    HasPermission(ctx context.Context, userID, resource, action string) (bool, error)
}

func requirePermission(ctx context.Context, checker PermissionChecker, resource, action string) error {
    caller := CallerFrom(ctx)
    if caller.IsSystem() {
        return nil  // system callers bypass RBAC
    }
    if caller.UserID == "" {
        return ErrUnauthenticated
    }
    has, err := checker.HasPermission(ctx, caller.UserID, resource, action)
    if err != nil {
        return fmt.Errorf("checking permission %s:%s: %w", resource, action, err)
    }
    if !has {
        return ErrForbidden
    }
    return nil
}
```

This replaces the `RequirePermission` middleware in `rbac_middleware.go`. During the incremental migration, both can coexist: `wrapPerm` stays on non-migrated endpoints; migrated endpoints switch to `wrapAuth` (auth only) since the service handles RBAC.

### 3. Audit Context Bridge

A single function converts the `Caller` from context into the existing `audit.Context`, replacing `h.auditContext(r)` in API handlers and `s.auditContext(ctx)` in the MCP server.

```go
// internal/service/audit.go
package service

func enrichAuditCtx(ctx context.Context) context.Context {
    caller := CallerFrom(ctx)
    return audit.WithContext(ctx, &audit.Context{
        UserID:    caller.UserID,
        Username:  caller.Username,
        IPAddress: caller.IPAddress,
        Source:    caller.Source,
    })
}
```

### 4. Service Error Types

Transport adapters map service errors to their native error format (HTTP status codes, MCP errors, etc.).

```go
// internal/service/errors.go
package service

var (
    ErrNotFound        = errors.New("not found")
    ErrAlreadyExists   = errors.New("already exists")
    ErrValidation      = errors.New("validation error")
    ErrForbidden       = errors.New("forbidden")
    ErrUnauthenticated = errors.New("unauthenticated")
    ErrSystemRole      = errors.New("cannot modify system role")
    ErrSelfDelete      = errors.New("cannot delete own account")
)

type ValidationError struct {
    Field   string
    Message string
}

type ValidationErrors []ValidationError  // implements error interface
```

**HTTP error mapping** (in `handleServiceError`):
| Service Error | HTTP Status | Code |
|---|---|---|
| `ErrNotFound` | 404 | `NOT_FOUND` |
| `ErrForbidden` | 403 | `FORBIDDEN` |
| `ErrUnauthenticated` | 401 | `UNAUTHORIZED` |
| `ErrAlreadyExists` | 409 | `ALREADY_EXISTS` |
| `ErrValidation` / `ValidationErrors` | 400 | `VALIDATION_ERROR` |
| `ErrSelfDelete` | 400 | `CANNOT_DELETE_SELF` |
| `ErrSystemRole` | 400 | `SYSTEM_ROLE` |
| other | 500 | `INTERNAL_ERROR` |

### 5. Services Registry

A convenience struct holding all services, created once at server startup:

```go
// internal/service/services.go
package service

type Services struct {
    Devices       *DeviceService
    Datacenters   *DatacenterService
    Networks      *NetworkService
    Pools         *PoolService
    Relationships *RelationshipService
    Discovery     *DiscoveryService
    Users         *UserService
    Roles         *RoleService
    Auth          *AuthService
    Audit         *AuditService
    APIKeys       *APIKeyService
    Bulk          *BulkService
}

func NewServices(store storage.ExtendedStorage, sessionManager *auth.SessionManager, scanner discovery.Scanner) *Services {
    return &Services{
        Devices:       NewDeviceService(store),
        Datacenters:   NewDatacenterService(store),
        Networks:      NewNetworkService(store),
        Pools:         NewPoolService(store),
        Relationships: NewRelationshipService(store),
        Discovery:     NewDiscoveryService(store, scanner),
        Users:         NewUserService(store, sessionManager),
        Roles:         NewRoleService(store),
        Auth:          NewAuthService(store, sessionManager),
        Audit:         NewAuditService(store),
        APIKeys:       NewAPIKeyService(store),
        Bulk:          NewBulkService(store),
    }
}
```

### 6. Per-Resource Service Pattern

Each service follows the same pattern:

```go
// Example: DeviceService
type DeviceService struct {
    store storage.ExtendedStorage
}

func NewDeviceService(store storage.ExtendedStorage) *DeviceService {
    return &DeviceService{store: store}
}

func (s *DeviceService) List(ctx context.Context, filter *model.DeviceFilter) ([]model.Device, error) {
    if err := requirePermission(ctx, s.store, "devices", "list"); err != nil {
        return nil, err
    }
    return s.store.ListDevices(filter)
}

func (s *DeviceService) Create(ctx context.Context, device *model.Device) error {
    if err := requirePermission(ctx, s.store, "devices", "create"); err != nil {
        return err
    }
    // Validation (moved from API handler)
    if device.Name == "" {
        return ValidationErrors{{Field: "name", Message: "Name is required"}}
    }
    return s.store.CreateDevice(enrichAuditCtx(ctx), device)
}

// Get, Update, Delete, Search follow the same pattern
```

### 7. Complex Service Example: UserService

```go
type UserService struct {
    store    storage.ExtendedStorage
    sessions SessionInvalidator  // interface for InvalidateUserSessions
}

func (s *UserService) Create(ctx context.Context, req *model.CreateUserRequest) (*model.UserResponse, error) {
    if err := requirePermission(ctx, s.store, "users", "create"); err != nil {
        return nil, err
    }

    // Validation
    var errs ValidationErrors
    if req.Username == "" { errs = append(errs, ValidationError{"username", "Username is required"}) }
    if len(req.Password) < 8 { errs = append(errs, ValidationError{"password", "Password must be at least 8 characters"}) }
    if req.Email == "" { errs = append(errs, ValidationError{"email", "Email is required"}) }
    if len(errs) > 0 { return nil, errs }

    // Uniqueness checks
    if existing, _ := s.store.GetUserByUsername(req.Username); existing != nil {
        return nil, fmt.Errorf("username: %w", ErrAlreadyExists)
    }

    // Hash password
    passwordHash, err := auth.HashPassword(req.Password)
    if err != nil { return nil, err }

    // Create user
    user := &model.User{Username: req.Username, Email: req.Email, FullName: req.FullName,
        PasswordHash: passwordHash, IsActive: true, IsAdmin: req.IsAdmin}
    if err := s.store.CreateUser(enrichAuditCtx(ctx), user); err != nil { return nil, err }

    // Assign role if specified
    if req.RoleID != "" {
        _ = s.store.AssignRoleToUser(ctx, user.ID, req.RoleID)
    }

    resp := user.ToResponse()
    return &resp, nil
}

func (s *UserService) Delete(ctx context.Context, id string) error {
    if err := requirePermission(ctx, s.store, "users", "delete"); err != nil { return err }

    // Prevent self-deletion
    caller := CallerFrom(ctx)
    if caller.UserID == id { return ErrSelfDelete }

    if err := s.store.DeleteUser(enrichAuditCtx(ctx), id); err != nil { return ErrNotFound }

    s.sessions.InvalidateUserSessions(id)
    return nil
}
```

### 8. AuthService (Login/Logout -- Special Case)

Login and logout are inherently transport-coupled (cookies, rate limiting). The service handles credential verification and session creation. Cookie management stays in the API handler.

```go
type AuthService struct {
    store          storage.ExtendedStorage
    sessionManager *auth.SessionManager
}

type LoginResult struct {
    User    model.UserResponse
    Session *auth.Session
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
    // No RBAC check -- login is pre-auth
    user, err := s.store.GetUserByUsername(username)
    if err != nil { return nil, ErrUnauthenticated }
    if !user.IsActive { return nil, ErrUnauthenticated }
    if err := auth.VerifyPassword(user.PasswordHash, password); err != nil { return nil, ErrUnauthenticated }

    isAdmin, _ := s.store.HasPermission(ctx, user.ID, "users", "create")
    session, err := s.sessionManager.CreateSession(user.ID, user.Username, isAdmin)
    if err != nil { return nil, err }

    _ = s.store.UpdateUserLastLogin(user.ID, time.Now())

    resp := user.ToResponse()
    if roles, err := s.store.GetUserRoles(ctx, user.ID); err == nil {
        resp.Roles = roles
    }
    return &LoginResult{User: resp, Session: session}, nil
}

func (s *AuthService) GetCurrentUser(ctx context.Context) (*model.UserResponse, error) {
    caller := CallerFrom(ctx)
    if caller.UserID == "" { return nil, ErrUnauthenticated }
    user, err := s.store.GetUser(caller.UserID)
    if err != nil { return nil, ErrNotFound }
    resp := user.ToResponse()
    if roles, err := s.store.GetUserRoles(ctx, caller.UserID); err == nil {
        resp.Roles = roles
    }
    return &resp, nil
}
```

### 9. Transport Adapter Changes

**API Handler** becomes thin:

```go
// Handler struct gains svc field
type Handler struct {
    svc              *service.Services
    store            storage.ExtendedStorage  // kept temporarily during migration
    sessionManager   *auth.SessionManager
    loginRateLimiter *RateLimiter
    cookieSecure     bool
    sessionTTL       time.Duration
    trustProxy       bool
}

// Handler methods become thin:
func (h *Handler) createDevice(w http.ResponseWriter, r *http.Request) {
    var device model.Device
    if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
        h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
        return
    }
    if err := h.svc.Devices.Create(r.Context(), &device); err != nil {
        h.handleServiceError(w, err)
        return
    }
    h.writeJSON(w, http.StatusCreated, device)
}

// Generic error mapper:
func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
    switch {
    case errors.Is(err, service.ErrNotFound):
        h.writeError(w, 404, "NOT_FOUND", err.Error())
    case errors.Is(err, service.ErrForbidden):
        h.writeError(w, 403, "FORBIDDEN", "Forbidden")
    case errors.Is(err, service.ErrUnauthenticated):
        h.writeError(w, 401, "UNAUTHORIZED", "Unauthorized")
    // ... etc
    default:
        h.internalError(w, err)
    }
}
```

**MCP Server** becomes thin:

```go
type Server struct {
    mcpServer   *mcp.Server
    svc         *service.Services
    requireAuth bool
}

// In HandleRequest, after API key auth:
caller := &service.Caller{Type: service.CallerTypeAPIKey, UserID: key.ID, Username: key.Name, Source: "mcp"}
ctx = service.WithCaller(ctx, caller)

// Tool handlers become thin:
func (s *Server) handleDeviceSave(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
    device := &model.Device{ /* parse from req */ }
    if device.ID == "" {
        if err := s.svc.Devices.Create(ctx, device); err != nil { return nil, err }
    } else {
        if err := s.svc.Devices.Update(ctx, device); err != nil { return nil, err }
    }
    return mcp.NewToolResponseJSON(device), nil
}
```

## Implementation Phases

### Phase 1: Foundation (no behavior change)
1. Create `internal/service/caller.go` - Caller type, WithCaller, CallerFrom, SystemContext
2. Create `internal/service/errors.go` - error types, ValidationErrors
3. Create `internal/service/rbac.go` - requirePermission helper
4. Create `internal/service/audit.go` - enrichAuditCtx helper
5. Create `internal/service/services.go` - Services registry
6. Modify `internal/api/middleware.go` - inject `Caller` into context after successful session/API key auth
7. Modify `internal/mcp/server.go` HandleRequest - inject `Caller` into context after API key auth
8. Add `handleServiceError()` to `internal/api/handlers.go`
9. Add `svc *service.Services` field to Handler, wire up in `internal/server/server.go`

**Verify**: `go build ./...` and `go test ./...` pass. No behavior change yet.

### Phase 2: Simple CRUD Resources
Migrate one resource at a time. For each resource:
- Create service file with RBAC + validation + storage delegation
- Refactor API handler to call service (remove business logic)
- Refactor MCP tool to call service (remove direct storage access)
- Switch route registration from `wrapPerm` to `wrapAuth` (RBAC now in service)

Order:
1. **DeviceService** - thin CRUD, proves the whole pattern works end-to-end
2. **DatacenterService** - same pattern
3. **NetworkService** + **PoolService**
4. **RelationshipService**
5. **BulkService**

**Verify after each resource**: `go build ./...`, `go test ./...`, manual API + MCP test.

### Phase 3: Discovery (multi-step orchestration)
6. **DiscoveryService** - includes promote (get discovered -> create device -> mark promoted), scan start, scan listing

### Phase 4: User/Role Management (complex business logic)
7. **UserService** - validation, password hashing, role assignment, session invalidation, self-delete prevention, enrich response with roles
8. **RoleService** - system role protection, permission management, grant/revoke

### Phase 5: Auth & Remaining
9. **AuthService** - login (credential verification + session creation), logout, getCurrentUser. Cookie management stays in HTTP handler.
10. **AuditService** - list/export. Also used by `cmd/audit` CLI (replaces direct storage access)
11. **APIKeyService**

### Phase 6: Cleanup
12. Remove `wrapPerm` entirely - all RBAC is now in service layer
13. Remove `Handler.store` field - all access goes through `Handler.svc`
14. Remove `h.auditContext()` from API handlers
15. Remove `s.auditContext()` from MCP server
16. Remove `rbac_middleware.go` (no longer needed)
17. Migrate `cmd/audit` to use AuditService with SystemContext instead of direct storage access

## Files Modified

| File | Change |
|------|--------|
| `internal/service/*.go` | **New** - all service files |
| `internal/api/handlers.go` | Add `svc *service.Services` to Handler, add `handleServiceError`, update `NewHandler` |
| `internal/api/middleware.go` | Inject `Caller` into context after successful auth |
| `internal/api/device_handlers.go` | Refactor to call `svc.Devices.*` |
| `internal/api/datacenter_handlers.go` | Refactor to call `svc.Datacenters.*` |
| `internal/api/network_handlers.go` | Refactor to call `svc.Networks.*` / `svc.Pools.*` |
| `internal/api/relationship_handlers.go` | Refactor to call `svc.Relationships.*` |
| `internal/api/discovery_handlers.go` | Refactor to call `svc.Discovery.*` |
| `internal/api/user_handlers.go` | Refactor to call `svc.Users.*` |
| `internal/api/role_handlers.go` | Refactor to call `svc.Roles.*` |
| `internal/api/auth_handlers.go` | Refactor to call `svc.Auth.*` (keep cookie logic) |
| `internal/api/audit_handlers.go` | Refactor to call `svc.Audit.*` |
| `internal/api/apikey_handlers.go` | Refactor to call `svc.APIKeys.*` |
| `internal/api/bulk_handlers.go` | Refactor to call `svc.Bulk.*` |
| `internal/api/rbac_middleware.go` | Eventually removed |
| `internal/mcp/server.go` | Replace `storage` with `svc`, inject Caller in auth, refactor all 14 tools |
| `internal/server/server.go` | Create Services registry, pass to Handler and MCP Server |
| `cmd/audit/audit.go` | Use AuditService with SystemContext |

## Testing Strategy

### Service Unit Tests
Each service gets its own `_test.go` using in-memory SQLite (existing pattern):

```go
func TestDeviceService_Create_Forbidden(t *testing.T) {
    store := newTestStorage(t)
    svc := service.NewDeviceService(store)

    // User without devices:create permission
    ctx := service.WithCaller(context.Background(), &service.Caller{
        Type: service.CallerTypeUser, UserID: "user-no-perms", Source: "test",
    })

    err := svc.Create(ctx, &model.Device{Name: "test"})
    if !errors.Is(err, service.ErrForbidden) {
        t.Fatalf("expected ErrForbidden, got %v", err)
    }
}

func TestDeviceService_Create_SystemBypass(t *testing.T) {
    store := newTestStorage(t)
    svc := service.NewDeviceService(store)

    ctx := service.SystemContext(context.Background(), "test")
    err := svc.Create(ctx, &model.Device{Name: "test"})
    if err != nil {
        t.Fatalf("system caller should bypass RBAC: %v", err)
    }
}
```

### Existing Tests
Continue working during migration. API handler tests that use real storage still function since handlers now delegate to services which delegate to storage.

### Manual Verification
1. **API**: All CRUD operations with authenticated user (session-based)
2. **MCP**: Device/network operations via MCP with API key - verify RBAC is enforced (user with `viewer` role should get 403 on create/update/delete)
3. **CLI**: `rackd audit list` works after migrating to AuditService
4. **Build**: `go build ./...` and `go test ./...` pass after each phase

## Migration Progress

### Phase 1: Foundation — DONE

All foundation files created and wired up:

- [x] `internal/service/caller.go` — Caller type, WithCaller, CallerFrom, SystemContext
- [x] `internal/service/errors.go` — error types, ValidationErrors (with `Unwrap()`)
- [x] `internal/service/rbac.go` — requirePermission helper (API keys bypass RBAC)
- [x] `internal/service/audit.go` — enrichAuditCtx helper
- [x] `internal/service/services.go` — Services registry, NewServices constructor
- [x] `internal/api/middleware.go` — Caller injected in both session and API key auth paths
- [x] `internal/mcp/server.go` — Caller injected (API key auth) or SystemContext (no-auth mode)
- [x] `internal/api/handlers.go` — `svc` field, `SetServices()`, `handleServiceError()`, `toValidationErrors()`
- [x] `internal/server/server.go` — creates Services, passes to Handler and MCP Server

### Phase 2: Simple CRUD Resources

| # | Resource | Service file | API handlers | Routes | MCP tools | Tests | Status |
|---|----------|-------------|--------------|--------|-----------|-------|--------|
| 1 | **Devices** | `service/device.go` | `device_handlers.go` → `svc.Devices.*` | `wrapAuth` | `svc.Devices.*` | `authReq()` | **DONE** |
| 2 | **Datacenters** | `service/datacenter.go` | `datacenter_handlers.go` → `svc.Datacenters.*` | `wrapAuth` | `svc.Datacenters.*` | `authReq()` | **DONE** |
| 3 | **Networks** | `service/network.go` | `network_handlers.go` → `svc.Networks.*` | `wrapAuth` | `svc.Networks.*` | `authReq()` | **DONE** |
| 4 | **Pools** | `service/pool.go` | `network_handlers.go` → `svc.Pools.*` | `wrapAuth` | `svc.Pools.GetNextIP()` | `authReq()` | **DONE** |
| 5 | **Relationships** | `service/relationship.go` | `relationship_handlers.go` → `svc.Relationships.*` | `wrapAuth` | `svc.Relationships.*` | `authReq()` | **DONE** |
| 6 | **Bulk** | `service/bulk.go` | `device_handlers.go` → `svc.Bulk.*`, `network_handlers.go` → `svc.Bulk.*` | `wrapAuth` | N/A | `authReq()` | **DONE** |

### Phase 3: Discovery — DONE

| # | Resource | Service file | API handlers | Routes | MCP tools | Tests | Status |
|---|----------|-------------|--------------|--------|-----------|-------|--------|
| 7 | **Discovery** | `service/discovery.go` | `discovery_handlers.go` → `svc.Discovery.*` | `wrapAuth` | `svc.Discovery.*` | `authReq()` | **DONE** |

### Phase 4: User/Role Management — DONE

| # | Resource | Service file | API handlers | Routes | MCP tools | Tests | Status |
|---|----------|-------------|--------------|--------|-----------|-------|--------|
| 8 | **Users** | `service/user.go` | `user_handlers.go` → `svc.Users.*` | `wrapAuth` | N/A | N/A | **DONE** |
| 9 | **Roles** | `service/role.go` | `role_handlers.go` → `svc.Roles.*` | `wrapAuth` | N/A | N/A | **DONE** |

### Phase 5: Auth & Remaining — TODO

| # | Resource | Service file | API handlers | Routes | MCP tools | Tests | Status |
|---|----------|-------------|--------------|--------|-----------|-------|--------|
| 10 | **Auth** | `service/auth.go` | `auth_handlers.go` | mixed | N/A | N/A | TODO |
| 11 | **Audit** | `service/audit_svc.go` | `audit_handlers.go` | `wrapPerm` | N/A | no auth | TODO |
| 12 | **API Keys** | `service/apikey.go` | `apikey_handlers.go` | `wrapPerm` | N/A | no auth | TODO |

### Phase 6: Cleanup — TODO

- [ ] Remove `wrapPerm` entirely (all RBAC in service layer)
- [ ] Remove `Handler.store` field (all access via `Handler.svc`)
- [ ] Remove `h.auditContext()` from API handlers
- [ ] Remove `s.auditContext()` from MCP server
- [ ] Remove `rbac_middleware.go`
- [ ] Migrate `cmd/audit` to use AuditService with SystemContext

## Lessons Learned (Phase 1 & 2 Review)

These pitfalls were discovered during Phase 1/2 implementation. Apply these to all subsequent phases.

### 1. Caller Injection: Cover ALL Auth Paths

Both `AuthMiddleware` (API-key-only) and `AuthMiddlewareWithSessions` (session + API key) have **two separate auth paths** (session cookie, Bearer token). Each path must inject `Caller` into the context. It's easy to add Caller injection for the session path and forget the API key path (or vice versa).

**Checklist for each auth middleware path:**

- After successful session auth → inject `Caller{Type: CallerTypeUser, UserID: session.UserID, ...}`
- After successful API key auth → inject `Caller{Type: CallerTypeAPIKey, UserID: key.ID, ...}`
- Both must call `r = r.WithContext(service.WithCaller(r.Context(), caller))` before calling `next`

### 2. API Key Callers Bypass RBAC

API keys (`model.APIKey`) have no `UserID` field that maps to a user in the `users` table. This means `HasPermission(ctx, key.ID, ...)` would always return false. The current design bypasses RBAC for `CallerTypeAPIKey` callers in `requirePermission()`.

**Do not** attempt to do RBAC lookups for API key callers until the `APIKey` model is extended with a `UserID` field.

### 3. Route Wrapper: `wrapAuth` not `wrap` for Migrated Endpoints

When migrating an endpoint from `wrapPerm` (middleware RBAC) to service-layer RBAC, the route must switch to `wrapAuth` (**always** requires auth), NOT `wrap` (auth only when `cfg.requireAuth` is set). Using `wrap` means unauthenticated requests bypass auth entirely when auth is not globally configured, resulting in no `Caller` in context and `ErrUnauthenticated` from the service layer.

**Rule**: Every migrated endpoint uses `wrapAuth`. `wrap` is only for truly optional-auth endpoints (e.g., `/healthz`, `/metrics`).

### 4. ValidationErrors Must Implement `Unwrap()`

`service.ValidationErrors` must have an `Unwrap() error` method returning `service.ErrValidation` so that `errors.Is(err, service.ErrValidation)` works in `handleServiceError`. Without this, the switch/case in `handleServiceError` falls through to the `default` branch and returns 500 instead of 400.

```go
func (e ValidationErrors) Unwrap() error {
    return ErrValidation
}
```

### 5. Two ValidationErrors Types Need Conversion

`api.ValidationErrors` (used by `writeValidationErrors`) and `service.ValidationErrors` are different types. The `toValidationErrors()` function must handle both:

1. Direct `api.ValidationErrors` (from API-layer validation)
2. `service.ValidationErrors` (from service layer) → convert field-by-field to `api.ValidationErrors`

Use `errors.As(err, &svcErrs)` for the service type since it may be wrapped.

### 6. MCP No-Auth Path Needs SystemContext

When `requireAuth=false` in the MCP server, no API key auth runs, so no `Caller` is injected. Service methods called without a Caller get `ErrUnauthenticated`. The fix: inject `SystemContext("mcp")` when auth is not required, so service calls succeed with system-level access.

```go
if s.requireAuth {
    // ... API key auth, inject CallerTypeAPIKey ...
} else {
    ctx = service.SystemContext(ctx, "mcp")
}
```

### 7. Handler Reads Must Also Go Through Service

When an update handler first reads the current entity (e.g., `updateDevice` fetches the device before applying patches), that read must also go through the service layer. Using `h.store.GetDevice(id)` directly bypasses RBAC and leaks entity existence to unauthorized callers.

**Pattern for update handlers:**

```go
// Use service for the initial read too
device, err := h.svc.Devices.Get(r.Context(), id)
if err != nil {
    h.handleServiceError(w, err)
    return
}
// ... apply updates from request body ...
if err := h.svc.Devices.Update(r.Context(), device); err != nil {
    h.handleServiceError(w, err)
    return
}
```

### 8. Tests Must Provide Auth for `wrapAuth` Endpoints

After switching routes from `wrap`/`wrapPerm` to `wrapAuth`, existing tests that don't provide authentication will get 401. Update test setup:

1. `setupTestHandler` must create a test API key in the store and set up services on the handler
2. Add an `authReq(req)` helper that adds `Authorization: Bearer <test-key>` to requests
3. All device/bulk endpoint test requests must use `authReq()`
4. Integration tests using `http.Post`/`http.Get` must switch to `authPost`/`authGet` helpers
5. Consider adding a `CreateDevice_Unauthenticated` test that verifies 401 without auth

### 9. Incremental Migration Checklist (Per Resource)

Use this checklist when migrating each resource in Phases 2-5:

- [ ] Create `internal/service/<resource>.go` with RBAC + validation + storage delegation
- [ ] Each method calls `requirePermission(ctx, s.store, "<resource>", "<action>")` first
- [ ] Each mutation method calls `enrichAuditCtx(ctx)` before passing to storage
- [ ] `ValidationErrors` returned from service have `Unwrap()` → `ErrValidation`
- [ ] Switch API route from `wrapPerm(h.<handler>, ...)` to `wrapAuth(h.<handler>)`
- [ ] Refactor API handler to call `h.svc.<Resource>.<Method>()` and map errors via `handleServiceError`
- [ ] Update handler reads (GET before PUT/PATCH) to also go through service
- [ ] Refactor MCP tool to call `s.svc.<Resource>.<Method>()` instead of `s.store.<Method>()`
- [ ] Update API handler tests: use `authReq()` for all requests to this resource
- [ ] Update integration tests: use `authPost`/`authGet`/`authDo` for requests to this resource
- [ ] Run `go build ./...` and `go test ./...` - all green
- [ ] Manual test: API CRUD, MCP tool, verify 403 for unauthorized user
