# RBAC UI Enforcement Plan

## Status: IN_PROGRESS (Phase 1: COMPLETED, Phase 2: COMPLETED, Phase 3: COMPLETED, Phase 4: COMPLETED, Phase 5: COMPLETED, Phase 6: COMPLETED)

## Problem

The UI currently exposes all features, buttons, forms, and navigation items to all users regardless of their RBAC permissions. A user with only `read` permissions can still see and click "Add", "Edit", "Delete" buttons, and navigation items to resources like "Users" that they don't have access to. This leads to:

1. **Poor UX**: Users are shown options they can't use
2. **API errors**: UI makes calls that return 403 Forbidden
3. **Security confusion**: Admins think users have more access than they do
4. **No feedback**: Users don't know why actions are blocked (except generic error messages)

## Goals

1. **Navigation**: Show/hide nav items based on user permissions
2. **Buttons/Actions**: Enable/disable or show/hide action buttons based on permissions
3. **Forms**: Prevent form submission for resources user can't create/edit
4. **Resource Access**: Block access to pages user doesn't have permission for
5. **Graceful Degradation**: Show helpful messages when access is denied
6. **Permission API**: Provide endpoint to get user's permissions for UI decisions

## High-Level Approach

1. **Backend**: Enhance `/api/auth/me` endpoint to return user's effective permissions and roles
2. **UI State**: Store permissions in Alpine.js reactive state for easy access
3. **Navigation**: Filter nav items based on required permissions
4. **Component Guards**: Add helper functions to check permissions before rendering
5. **Route Protection**: Check permissions on page load, redirect if denied
6. **Form/Action Guards**: Check permissions before showing buttons or enabling forms

## Phases

### Phase 1: Enhanced `/api/auth/me` Response - **COMPLETED**

**Backend Changes:**

`internal/model/user.go`:
- ✅ Add `CurrentUserResponse` struct that extends `User` with permissions and roles:
```go
type CurrentUserResponse struct {
    ID           string              `json:"id"`
    Username     string              `json:"username"`
    Email        string              `json:"email,omitempty"`
    FullName     string              `json:"full_name,omitempty"`
    IsActive     bool                `json:"is_active"`
    IsAdmin      bool                `json:"is_admin"`
    CreatedAt    time.Time           `json:"created_at"`
    UpdatedAt    time.Time           `json:"updated_at"`
    LastLoginAt  *time.Time          `json:"last_login_at,omitempty"`
    Permissions  []model.Permission  `json:"permissions"`
    Roles        []model.Role        `json:"roles"`
}
```

`internal/service/auth.go`:
- ✅ Add `GetCurrentUserWithPermissions()` and `GetCurrentUserWithPermissionsByID()` methods
- ✅ Fetches current user, their roles, and all permissions from those roles
- ✅ Returns consolidated response

`internal/api/auth_handlers.go`:
- ✅ Update `login()` handler to return enhanced user object with permissions
- ✅ Update `getCurrentUser()` handler to return `CurrentUserResponse` instead of plain `User`

**API Response:**
```json
{
  "id": "user-123",
  "username": "john",
  "email": "john@example.com",
  "full_name": "John Doe",
  "is_active": true,
  "is_admin": false,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z",
  "last_login_at": "2024-02-09T12:00:00Z",
  "permissions": [
    {"id": "perm-1", "name": "device:list", "resource": "devices", "action": "list"},
    {"id": "perm-2", "name": "device:read", "resource": "devices", "action": "read"},
    {"id": "perm-3", "name": "networks:list", "resource": "networks", "action": "list"}
  ],
  "roles": [
    {"id": "role-1", "name": "viewer", "is_system": true}
  ]
}
```

**Verification:**
- ✅ Test `/api/auth/me` returns permissions and roles for each role type
- ✅ Test login returns enhanced user object with permissions
- ✅ Test unauthorized access returns 401

### Phase 2: UI Permission State - **COMPLETED**

**Frontend Changes:**

`webui/src/core/types.ts`:
- ✅ Add `Permission` interface
- ✅ Add `CurrentUser` type that extends `User` with `permissions: Permission[]`

`webui/src/app.ts`:
- ✅ Fetch permissions on init after fetching current user
- ✅ Store in Alpine data as `permissions` reactive array
- ✅ Add helper: `can(resource, action)` → bool
- ✅ Add helper: `hasAnyPermission(resource, ...actions)` → bool
- ✅ Add helper: `hasAllPermissions(resource, ...actions)` → bool
- ✅ Add helper: `canList(resource)` → bool
- ✅ Add helper: `canRead(resource)` → bool
- ✅ Add helper: `canCreate(resource)` → bool
- ✅ Add helper: `canUpdate(resource)` → bool
- ✅ Add helper: `canDelete(resource)` → bool

`webui/src/core/api.ts`:
- ✅ Update `getCurrentUser()` to return `CurrentUser` instead of `User`

**Example Alpine data:**
```typescript
Alpine.data('permissions', () => ({
   permissions: [],
   roles: [],
   loaded: false,

   init() {
     this.load();
   },

   async load() {
     try {
       const user = await api.getCurrentUser();
       this.permissions = user.permissions;
       this.roles = user.roles || [];
       this.loaded = true;
     } catch {
       this.permissions = [];
       this.roles = [];
       this.loaded = true;
     }
   },

   can(resource: string, action: string): boolean {
     return this.permissions.some(p =>
       p.resource === resource && p.action === action
     );
   },

   canList(resource: string): boolean {
     return this.can(resource, 'list');
   },

   canRead(resource: string): boolean {
     return this.can(resource, 'read');
   },

   canCreate(resource: string): boolean {
     return this.can(resource, 'create');
   },

   canUpdate(resource: string): boolean {
     return this.can(resource, 'update');
   },

   canDelete(resource: string): boolean {
     return this.can(resource, 'delete');
   }
}));
```

**Verification:**
- ✅ Test permissions load on app start
- ✅ Test helper functions return correct booleans
- ✅ Test empty permissions state

### Phase 3: Navigation Filtering - **COMPLETED**

**Backend Changes:**

`internal/api/config_handlers.go`:
- ✅ Update `NavItem` struct to include `RequiredPermissions: []PermissionCheck`
- ✅ Update `UserInfo` struct to include `Permissions []model.Permission` and `Roles []model.Role`
- ✅ Update `HandlerWithSession(sessionManager, store)` to accept storage parameter
- ✅ Fetch user roles and permissions from storage
- ✅ Return filtered nav items based on user permissions in UserInfo

`internal/api/handlers.go`:
- ✅ Update `getConfig()` to add `required_permissions` to Users and Roles nav items

`internal/server/server.go`:
- ✅ Update Users nav item to require `users:list` permission
- ✅ Update Roles nav item to require `roles:list` permission
- ✅ Update `HandlerWithSession()` calls to pass `store` parameter

`internal/api/integration_test.go`:
- ✅ Update test to pass sessionManager and store to Handler()

`internal/api/config_handlers_test.go`:
- ✅ Update tests to use new Handler signature
- ✅ Update UserInfo to use model.Role and model.Permission types

**Frontend Changes:**

`webui/src/core/types.ts`:
- ✅ Update `NavItem` interface to include `required_permissions?: {resource: string, action: string}[]`
- ✅ Update `UserInfo` interface to include `permissions?: Permission[]`

`webui/src/components/nav.ts`:
- ✅ Add `filteredItems` getter that filters nav items based on `required_permissions`
- ✅ Update to fetch user permissions from config
- ✅ Filter items using permissions: check if user has all required permissions

### Phase 4: Action Button Guards - **COMPLETED**

**Implementation Notes:**

- Converted `permissions` from `Alpine.data` to `Alpine.store` so it's accessible as `$store.permissions` in all components
- Permissions are initialized from `window.rackdConfig.user.permissions` (loaded from `/api/config`)
- All action buttons now use `x-show="$store.permissions.canCreate/canUpdate/canDelete('resource')"` guards

**Frontend Changes:**

`webui/src/app.ts`:
- ✅ Converted `permissions()` function from `Alpine.data` to `Alpine.store('permissions', ...)` via `initPermissionsStore()`
- ✅ Store initialized from `window.rackdConfig.user.permissions` (no extra API call needed)

`webui/src/partials/pages/devices.html`:
- ✅ "Add Device" button: `x-show="$store.permissions.canCreate('devices')"`
- ✅ "Edit" button: `x-show="$store.permissions.canUpdate('devices')"`
- ✅ "Delete" button: `x-show="$store.permissions.canDelete('devices')"`

`webui/src/partials/pages/networks.html`:
- ✅ "Add Network" button: `x-show="$store.permissions.canCreate('networks')"`
- ✅ "Edit" button: `x-show="$store.permissions.canUpdate('networks')"`
- ✅ "Delete" button: `x-show="$store.permissions.canDelete('networks')"`

`webui/src/partials/pages/datacenters.html`:
- ✅ "Add Datacenter" button: `x-show="$store.permissions.canCreate('datacenters')"`
- ✅ "Edit" button: `x-show="$store.permissions.canUpdate('datacenters')"`
- ✅ "Delete" button: `x-show="$store.permissions.canDelete('datacenters')"`

`webui/src/partials/pages/discovery.html`:
- ✅ "New Scan" button: `x-show="$store.permissions.canCreate('discovery')"`
- ✅ "Delete Old" button: combined with existing status check
- ✅ "Delete" scan button: combined with existing status check
- ✅ "Delete All" devices button: combined with existing count check
- ✅ "Promote" button: `x-show="$store.permissions.canCreate('devices')"` (promotes to device)
- ✅ "Delete" device button: `x-show="$store.permissions.canDelete('discovery')"`

`webui/src/components/credentials.ts`:
- ✅ "Add Credential" button: `x-show="$store.permissions.canCreate('credentials')"`
- ✅ "Edit" button: `x-show="$store.permissions.canUpdate('credentials')"`
- ✅ "Delete" button: `x-show="$store.permissions.canDelete('credentials')"`

`webui/src/components/profiles.ts`:
- ✅ "Add Profile" button: `x-show="$store.permissions.canCreate('scan_profiles')"`
- ✅ "Edit" button: `x-show="$store.permissions.canUpdate('scan_profiles')"`
- ✅ "Delete" button: `x-show="$store.permissions.canDelete('scan_profiles')"`

`webui/src/components/scheduled-scans.ts`:
- ✅ "Add Schedule" button: `x-show="$store.permissions.canCreate('scheduled_scans')"`
- ✅ "Edit" button: `x-show="$store.permissions.canUpdate('scheduled_scans')"`
- ✅ "Delete" button: `x-show="$store.permissions.canDelete('scheduled_scans')"`

`webui/src/partials/pages/users.html`:
- ✅ "Add User" button: `x-show="$store.permissions.canCreate('users')"`
- ✅ "Edit" button: `x-show="$store.permissions.canUpdate('users')"`
- ✅ "Roles" button: `x-show="$store.permissions.canList('roles')"`
- ✅ "Password" button: `x-show="$store.permissions.canUpdate('users')"`
- ✅ "Delete" button: `x-show="$store.permissions.canDelete('users') && currentUser?.id !== user.id"`
- ✅ Role Manager Grant/Revoke: replaced `currentUser?.is_admin` with `$store.permissions.canUpdate('roles')`

`webui/src/partials/pages/roles.html`:
- ✅ "Edit" button: combined `!role.is_system` with `$store.permissions.canUpdate('roles')`
- Note: "Add Role" button is missing from the page header (pre-existing issue)

**Verification:**
- ✅ All builds pass (Go + WebUI)
- ✅ All tests pass
- Test each page with admin user - all actions visible
- Test each page with operator user - create/update visible, delete hidden
- Test each page with viewer user - only list/read visible
- Test users/roles pages with non-admin user - entire page hidden or redirect

### Phase 5: Route Protection - **COMPLETED**

**Implementation Notes:**

- Added `routePermissions` map in `app.ts` that maps route prefixes to required `resource:action` permissions
- Added `checkRoutePermission(path)` function that checks user permissions against the route map
- Router tracks `accessDenied` state, updated on every navigation (navigate, popstate, init)
- Access denied page shows centered card with icon, message, and "Go to Dashboard" button
- Routes without permission rules (dashboard, login) are always allowed
- Self-service profile update: `UserService.Update` now allows users to edit their own email/full_name without `users:update` permission (privileged fields like `is_active`/`is_admin` still require the permission)

**Frontend Changes:**

`webui/src/app.ts`:
- ✅ Added `routePermissions` array mapping route prefixes to required permissions
- ✅ Added `checkRoutePermission(path)` function
- ✅ Router `accessDenied` state tracked and updated on `init()`, `navigate()`, and `popstate`

Route permission map:
| Route prefix | Required permission |
|---|---|
| `/users` | `users:list` |
| `/roles` | `roles:list` |
| `/devices` | `devices:list` |
| `/networks` | `networks:list` |
| `/pools` | `networks:list` |
| `/datacenters` | `datacenters:list` |
| `/discovery` | `discovery:list` |

`webui/src/index.html`:
- ✅ Added access denied template with icon, message, and dashboard link
- ✅ Wrapped all page includes in `<template x-if="!accessDenied">` to prevent rendering

**Backend Changes:**

`internal/service/user.go`:
- ✅ `Update()` now allows self-updates (email, full_name) without `users:update` permission
- ✅ Privileged fields (`is_active`, `is_admin`) silently ignored for self-updates

**Verification:**
- ✅ Build passes (Go + WebUI)
- Test direct URL access to `/users` with viewer user -> shows access denied page
- Test direct URL access to `/roles` with viewer user -> shows access denied page
- Test browser back/forward navigation respects permission checks
- Test "Go to Dashboard" button navigates correctly

### Phase 6: Error Handling and Feedback - **COMPLETED**

**Implementation Notes:**

- Created `webui/src/components/toast.ts` with a toast notification component that supports success, error, warning, and info messages
- Enhanced `webui/src/core/api.ts` to:
  - Handle 403 Forbidden responses by dispatching a `toast:permission-denied` event
  - Handle 401 Unauthorized responses by redirecting to login
- Toast notifications appear in the top-right corner with smooth enter/leave animations
- Different toast types have distinct colors and icons for better UX
- Toast notifications persist for configurable duration (default: 5s for info/success, 6s for warning, 7s for error)
- Users can manually dismiss toast notifications with the close button
- Toast component registered as Alpine store (`$store.toast`) for easy access across all components

**Frontend Changes:**

`webui/src/components/toast.ts`:
- ✅ Created toast component with `show()`, `success()`, `error()`, `warning()`, `info()` methods
- ✅ Added `remove()` and `clear()` methods for managing notifications
- ✅ Exported `showPermissionDenied()` helper for non-Alpine contexts

`webui/src/core/api.ts`:
- ✅ Enhanced `request()` method to handle 403 Forbidden responses
- ✅ Dispatches `toast:permission-denied` custom event with user-friendly message
- ✅ Handles 401 Unauthorized by redirecting to login (if not already there)

`webui/src/app.ts`:
- ✅ Imported `toastComponent` from `./components/toast`
- ✅ Registered toast as Alpine store: `Alpine.store('toast', toastComponent())`
- ✅ Added event listener for `toast:permission-denied` events to show error toast

`webui/src/index.html`:
- ✅ Added toast notification container with fixed positioning (top-right corner)
- ✅ Toast container uses Alpine.js transitions for smooth animations
- ✅ Different toast types (success, error, warning, info) have distinct colors and icons
- ✅ Close button allows manual dismissal of notifications
- ✅ Accessible with proper ARIA attributes (`role="alert"`, `aria-live`)

**Example Toast Implementation:**
```typescript
// Show permission denied toast
Alpine.store('toast').error("You don't have permission to perform this action");

// Show success toast
Alpine.store('toast').success("Operation completed successfully");

// Show warning toast
Alpine.store('toast').warning("This action may have consequences");

// Show info toast
Alpine.store('toast').info("New feature available");
```

**Verification:**
- ✅ Build passes (Go + WebUI)
- ✅ All tests pass
- Test API 403 errors show user-friendly toast message
- Test 403 page displays correctly (from Phase 5)
- Test navigation back from 403 page works
- Test toast notifications dismiss correctly on timeout
- Test toast notifications dismiss correctly when clicking close button

### Phase 7: Polish and Edge Cases

**Frontend Changes:**

`webui/src/app.ts`:
- Add loading state while permissions are loading
- Prevent rendering app until permissions loaded
- Handle permission refresh (e.g., after role change)

`webui/src/components/user-menu.ts`:
- Show user's current roles in dropdown
- Add "My Permissions" page to view all permissions

`webui/src/components/`:
- Ensure all action buttons use permission helpers
- Add `:disabled` attributes to buttons user can't click (better UX than hiding)
- Consider showing "disabled" state with tooltip explaining why

**Edge Cases:**
1. Permission changes while user is logged in → add refresh permissions on role grant/revoke
2. Expired session → trigger re-auth
3. Network errors → show retry option
4. Permission loading failure → show error but allow basic navigation

**Verification:**
- Test permission refresh after admin grants new role
- Test permission loading with network issues
- Test all role transitions (viewer → operator → admin)

## File Changes Summary

### Backend
| File | Changes |
|------|---------|
| `internal/model/user.go` | Add `CurrentUserResponse` struct with permissions and roles |
| `internal/service/user.go` | Add `GetCurrentUserWithPermissions()` method |
| `internal/api/auth_handlers.go` | Update `login()` and `getCurrentUser()` to return enhanced response |
| `internal/api/api.go` | Update `NavItem` with `RequiredPermissions` |

### Frontend
| File | Changes |
|------|---------|
| `webui/src/core/types.ts` | Add Permission types, update NavItem |
| `webui/src/core/api.ts` | Add 403/401 error handling with toast notifications |
| `webui/src/app.ts` | Add permission state, helpers, route guards, toast store registration |
| `webui/src/components/nav.ts` | Filter nav items by permissions |
| `webui/src/components/toast.ts` | **NEW** - Toast notification component |
| `webui/src/components/devices.ts` | Add action button guards |
| `webui/src/components/networks.ts` | Add action button guards |
| `webui/src/components/datacenters.ts` | Add action button guards |
| `webui/src/components/discovery.ts` | Add action button guards |
| `webui/src/components/credentials.ts` | Add action button guards |
| `webui/src/components/profiles.ts` | Add action button guards |
| `webui/src/components/scheduled-scans.ts` | Add action button guards |
| `webui/src/components/users.ts` | Add component-level and action guards |
| `webui/src/components/roles.ts` | Add component-level and action guards |
| `webui/src/index.html` | Add toast notification container, access denied template |

## Testing Strategy

### Unit Tests
- Test `GetCurrentUserWithPermissions()` returns correct permissions for each role
- Test `/api/auth/me` endpoint with different users returns enhanced response
- Test permission helper functions

### Integration Tests
- Test `/api/auth/me` returns correct permissions/roles for different roles
- Test nav filtering with different roles
- Test action button visibility with different roles
- Test route protection with different roles
- Test API calls return 403 for unauthorized actions

### Manual Testing Matrix
| Role | Devices | Networks | Datacenters | Discovery | Users | Roles |
|------|---------|----------|-------------|-----------|-------|-------|
| Admin | R/W/D | R/W/D | R/W/D | R/W/D | R/W/D | R/W/D |
| Operator | R/W/D | R/W/D | R/W/D | R/W/D | Hidden | Hidden |
| Viewer | R only | R only | R only | R only | Hidden | Hidden |

Legend: R=Read, W=Write, D=Delete, Hidden=Not in nav

### Browser Testing
- Test in Chrome, Firefox, Safari
- Test mobile responsive view
- Test keyboard navigation with disabled buttons
- Test screen reader announcements for hidden/missing elements

## Rollout Plan

1. **Phase 1**: Enhanced `/api/auth/me` response (backend only, no UI changes) - ✅ COMPLETED
2. **Phase 2**: UI permission state (no visible changes) - ✅ COMPLETED
3. **Phase 3**: Navigation filtering (first visible change) - ✅ COMPLETED
4. **Phase 4**: Action button guards (biggest UX change) - ✅ COMPLETED
5. **Phase 5**: Form/route guards (security hardening) - ✅ COMPLETED
6. **Phase 6**: Error handling (UX polish) - ✅ COMPLETED
7. **Phase 7**: Polish (final touches) - IN PROGRESS

Each phase should be tested thoroughly before proceeding to the next.

## Migration Considerations

- Existing users: No action needed, permissions already in DB
- New users: Automatically get permissions from assigned roles
- Role changes: Require UI refresh or automatic permission reload

## Future Enhancements

- Permission groups (e.g., "manage_devices" = device:list + device:read + device:create + device:update + device:delete)
- Resource-level permissions (e.g., "edit this specific device")
- Time-based permissions (e.g., "operator during business hours only")
- Audit log of permission changes
- Permission request workflow for users
