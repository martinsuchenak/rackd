# RBAC Remediation Plan

## Status: DONE

## Problems

1. **Metrics endpoint** (`/metrics`) uses `wrap()` — no auth when `requireAuth` is false
2. **Credentials, scan profiles, scheduled scans** use `wrap()` — data-mutating endpoints without guaranteed auth
3. **API keys are global** — no `user_id` column, no user association, bypass all RBAC checks

## Phase 1: Switch wrap() routes to wrapAuth

Simply change credentials, scan profiles, scheduled scans, and metrics routes from `wrap()` to `wrapAuth()`.

### Status: DONE

### Files Modified
- `internal/api/handlers.go` — Lines 164-188 and 250: change `wrap(` to `wrapAuth(` for:
  - `GET/POST /api/credentials`, `GET/PUT/DELETE /api/credentials/{id}`
  - `GET/POST /api/scan-profiles`, `GET/PUT/DELETE /api/scan-profiles/{id}`
  - `GET/POST /api/scheduled-scans`, `GET/PUT/DELETE /api/scheduled-scans/{id}`
  - `GET /metrics`

No service layer needed for these yet — they don't have RBAC permissions but at least require a valid session or API key.

## Phase 2: Per-User API Keys

### 2a: Model & DB Migration

**Status: DONE**

`internal/model/apikey.go`:
- Add `UserID string` field to `APIKey` struct (json: `"user_id"`)
- Add `UserID string` to `APIKeyResponse`
- Update `ToResponse()` to copy UserID
- Add `UserID string` to `APIKeyFilter`

`internal/storage/migrations.go`:
- New migration: `add_apikey_user_id`
  - `ALTER TABLE api_keys ADD COLUMN user_id TEXT REFERENCES users(id)`
  - `CREATE INDEX idx_api_keys_user_id ON api_keys(user_id)`
  - Existing keys get `user_id = NULL` (orphaned/system keys — keep working)

### 2b: Storage Layer

**Status: DONE**

`internal/storage/storage.go`:
- `ListAPIKeys` signature already takes `*model.APIKeyFilter` — add UserID filtering in SQL

SQLite implementation (likely `internal/storage/sqlite_apikey.go` or similar):
- `ListAPIKeys`: filter by `user_id` when `filter.UserID != ""`
- `CreateAPIKey`: persist `user_id` column
- `GetAPIKey`, `GetAPIKeyByKey`: return `user_id` in result

### 2c: Service Layer

**Status: DONE**

`internal/service/apikey.go`:
- `Create()`: set `key.UserID` from `CallerFrom(ctx).UserID` (the creating user owns the key)
- `List()`: if caller is not admin, filter to `filter.UserID = caller.UserID` (users see only their own keys)
- `Delete()`: verify caller owns the key or is admin
- `Get()`: verify caller owns the key or is admin

### 2d: Auth Middleware — Resolve API Key Owner

**Status: DONE**

`internal/api/middleware.go`:
- Change `AuthMiddleware` and `AuthMiddlewareWithSessions` to accept `storage.ExtendedStorage` instead of `storage.APIKeyStorage` (needed to call `GetUser` for owner lookup). Update call sites in `RegisterRoutes`.
- In both functions, when API key auth succeeds:
  - If `key.UserID != ""`: look up the user, create Caller with `CallerTypeUser` (inherits user's RBAC roles)
  - If `key.UserID == ""`: legacy key, keep `CallerTypeAPIKey` (bypass RBAC as before)

```go
if key.UserID != "" {
    user, err := store.GetUser(key.UserID)  // needs UserStorage in middleware
    if err == nil && user.IsActive {
        caller = &service.Caller{
            Type:      service.CallerTypeUser,
            UserID:    user.ID,
            Username:  user.Username,
            IPAddress: getClientIP(r, false),
            Source:    "apikey",
        }
    }
}
```

This means API key requests inherit the owner's RBAC permissions — no special bypass needed.

`internal/mcp/server.go` — Same change in MCP auth handler (~line 202).

### 2e: Remove API Key RBAC Bypass

**Status: DONE**

`internal/service/rbac.go`:
- Remove the `CallerTypeAPIKey` bypass block (lines 20-26)
- API keys with a `UserID` now go through normal RBAC (as `CallerTypeUser`)
- Legacy keys (no `UserID`) still need a path — keep bypass ONLY for `key.UserID == ""`

Updated logic:
```go
func requirePermission(ctx, checker, resource, action) error {
    caller := CallerFrom(ctx)
    if caller != nil && caller.IsSystem() { return nil }
    if caller == nil || caller.UserID == "" { return ErrUnauthenticated }
    // CallerTypeAPIKey with no UserID is now impossible — middleware resolves to User or keeps APIKey
    // Legacy API keys (no user) get CallerTypeAPIKey with UserID="" -> unauthenticated
    has, err := checker.HasPermission(ctx, caller.UserID, resource, action)
    if !has { return ErrForbidden }
    return nil
}
```

### 2f: API Key Handlers

**Status: DONE**

`internal/api/apikey_handlers.go`:
- `createAPIKey`: service now sets UserID from caller context
- `listAPIKeys`: service filters by caller's UserID (non-admin sees own keys only)
- `deleteAPIKey`: service verifies ownership

### 2g: UI Updates

**Status: DONE**

API key management UI — show UserID/username in key list, filter by current user.

## Phase 3: Service Layer for Credentials, Profiles, Scheduled Scans

### Status: DONE

Move credentials, scan profiles, and scheduled scans from direct storage access to the service layer pattern with RBAC enforcement.

### New Service Files

`internal/service/credential.go`:
- Create `CredentialService` with `List()`, `Create()`, `Get()`, `Update()`, `Delete()` methods
- Service constructor accepts `credentials.Storage` and `PermissionChecker` (for RBAC)
- All methods use `requirePermission(ctx, s.rbac, "credentials", "...")` for RBAC checks
- Storage errors converted to service errors (`ErrNotFound`, `ValidationErrors`)

`internal/service/scanprofile.go`:
- Create `ScanProfileService` with `List()`, `Create()`, `Get()`, `Update()`, `Delete()` methods
- Service constructor accepts `storage.ProfileStorage` and `PermissionChecker` (for RBAC)
- All methods use `requirePermission(ctx, s.rbac, "scan-profiles", "...")` for RBAC checks
- Storage errors converted to service errors

`internal/service/scheduledscan.go`:
- Create `ScheduledScanService` with `List()`, `Create()`, `Get()`, `Update()`, `Delete()` methods
- Service constructor accepts `storage.ScheduledScanStorage` and `PermissionChecker` (for RBAC)
- All methods use `requirePermission(ctx, s.rbac, "scheduled-scans", "...")` for RBAC checks
- Storage errors converted to service errors

### Service Registry Updates

`internal/service/services.go`:
- Add `Credentials *CredentialService`, `ScanProfiles *ScanProfileService`, `ScheduledScans *ScheduledScanService` to `Services` struct
- Add setter methods `SetCredentialsStorage()`, `SetProfileStorage()`, `SetScheduledScanStorage()` for optional storage dependencies
- Setter methods pass `s.Users.store` (which implements `PermissionChecker`) to service constructors for RBAC checks

### Handler Updates

`internal/api/credentials_handlers.go`:
- Update all handlers to call `h.svc.Credentials.*()` instead of `h.credStore.*()`
- Use `h.handleServiceError()` for consistent error handling

`internal/api/profiles_handlers.go`:
- Update all handlers to call `h.svc.ScanProfiles.*()` instead of `h.profileStore.*()`
- Use `h.handleServiceError()` for consistent error handling

`internal/api/scheduled_handlers.go`:
- Update all handlers to call `h.svc.ScheduledScans.*()` instead of `h.scheduledStore.*()`
- Use `h.handleServiceError()` for consistent error handling

### Server Initialization Updates

`internal/server/server.go`:
- Call `services.SetCredentialsStorage(credStore)` after creating services
- Call `services.SetProfileStorage(profileStore)` after creating services
- Call `services.SetScheduledScanStorage(scheduledStore)` after creating services

### RBAC Permissions

Permissions were already added in `migrateAddRBACMissingPermissionsUp`:
- `credential:*` (list, create, read, update, delete)
- `scan-profile:*` (list, create, read, update, delete)
- `scheduled-scan:*` (list, create, read, update, delete)

These are granted to:
- `admin` role: all permissions
- `operator` role: list, create, read, update (no delete)
- `viewer` role: list, read only

## Implementation Order

1. **Phase 1** (quick win): Switch `wrap()` to `wrapAuth()` for credentials, profiles, scheduled scans, metrics
2. **Phase 2a-b**: API key model + migration + storage (add `user_id`)
3. **Phase 2c**: APIKeyService changes (ownership logic)
4. **Phase 2d**: Auth middleware resolves API key owner to User caller
5. **Phase 2e**: Remove RBAC bypass for API keys
6. **Phase 2f-g**: Handler and UI updates
7. **Phase 3**: Service layers for credentials, scan profiles, scheduled scans with RBAC enforcement

## Files Modified Summary

| File | Change |
|------|--------|
| `internal/api/handlers.go` | `wrap()` → `wrapAuth()` for credentials/profiles/scheduled/metrics routes |
| `internal/model/apikey.go` | Add `UserID` field to `APIKey`, `APIKeyResponse`, `APIKeyFilter` |
| `internal/storage/migrations.go` | New migration adding `user_id` column + index; new permissions for credentials/profiles/scheduled scans |
| `internal/storage/` (SQLite impl) | Persist/filter `user_id` in API key CRUD |
| `internal/storage/storage.go` | Update `APIKeyStorage` interface if needed |
| `internal/service/apikey.go` | Ownership logic: set UserID on create, filter on list, verify on delete |
| `internal/service/rbac.go` | Remove `CallerTypeAPIKey` bypass |
| `internal/api/middleware.go` | Resolve API key UserID → User caller with RBAC |
| `internal/mcp/server.go` | Same API key owner resolution |
| `internal/api/apikey_handlers.go` | Adapt to service ownership logic |
| `internal/service/credential.go` | New `CredentialService` with RBAC enforcement |
| `internal/service/scanprofile.go` | New `ScanProfileService` with RBAC enforcement |
| `internal/service/scheduledscan.go` | New `ScheduledScanService` with RBAC enforcement |
| `internal/service/services.go` | Add new services and setter methods |
| `internal/api/credentials_handlers.go` | Use service layer instead of direct storage access |
| `internal/api/profiles_handlers.go` | Use service layer instead of direct storage access |
| `internal/api/scheduled_handlers.go` | Use service layer instead of direct storage access |
| `internal/server/server.go` | Call service setters after creating services |

## Verification

1. `go build ./...` passes after each phase
2. `go test ./...` passes after each phase
3. **Phase 1 test**: Unauthenticated requests to `/api/credentials`, `/api/scan-profiles`, `/api/scheduled-scans`, `/metrics` return 401
4. **Phase 2 test**: Create API key as user A → key has `user_id` = A. Use key to call API → RBAC checks run against user A's roles. User B cannot list/delete user A's keys (unless admin).
5. **Legacy keys test**: Existing keys without `user_id` continue to work (bypass RBAC as before, log deprecation warning)
6. **Phase 3 test**: User with only viewer role can list/read credentials/profiles/scheduled scans but cannot create/update/delete. User with operator role can create/update but not delete. Admin has full access.
