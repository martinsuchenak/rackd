# RBAC UI Enforcement Plan

## Status: IN_PROGRESS (Phase 2: COMPLETED, Phase 3: IN_PROGRESS)

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

### Phase 1: Enhanced `/api/auth/me` Response

**Backend Changes:**

`internal/model/user.go`:
- Add `CurrentUserResponse` struct that extends `User` with permissions and roles:
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

`internal/service/user.go`:
- Add `GetCurrentUserWithPermissions(ctx context.Context) (*model.CurrentUserResponse, error)` method
- Fetches current user, their roles, and all permissions from those roles
- Returns consolidated response

`internal/api/auth_handlers.go`:
- Update `login()` handler to return enhanced user object with permissions
- Update `getCurrentUser()` handler to return `CurrentUserResponse` instead of plain `User`

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
- Test `/api/auth/me` returns permissions and roles for each role type
- Test login returns enhanced user object with permissions
- Test unauthorized access returns 401

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

### Phase 3: Navigation Filtering - **IN_PROGRESS**

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

### Phase 4: Action Button Guards

**Frontend Changes:**

For each component, add `x-show` and `x-disabled` directives based on permissions:

`webui/src/components/devices.ts`:
- "Add Device" button: `x-show="permissions.canCreate('devices')"`
- "Edit" button: `x-show="permissions.canUpdate('devices')"`
- "Delete" button: `x-show="permissions.canDelete('devices')"`

`webui/src/components/networks.ts`:
- "Add Network" button: `x-show="permissions.canCreate('networks')"`
- "Edit" button: `x-show="permissions.canUpdate('networks')"`
- "Delete" button: `x-show="permissions.canDelete('networks')"`

`webui/src/components/datacenters.ts`:
- Similar guards for datacenter actions

`webui/src/components/discovery.ts`:
- "Start Scan" button: `x-show="permissions.canCreate('discovery')"`

`webui/src/components/credentials.ts`:
- Guard credential CRUD operations

`webui/src/components/profiles.ts`:
- Guard profile CRUD operations

`webui/src/components/scheduled-scans.ts`:
- Guard scheduled scan CRUD operations

`webui/src/components/users.ts`:
- Entire component: `x-show="permissions.canList('users')"`
- "Add User" button: `x-show="permissions.canCreate('users')"`
- "Edit User" button: `x-show="permissions.canUpdate('users')"`

`webui/src/components/roles.ts`:
- Entire component: `x-show="permissions.canList('roles')"`
- "Add Role" button: `x-show="permissions.canCreate('roles')"`
- Guard role management operations

**Verification:**
- Test each page with admin user - all actions visible
- Test each page with operator user - create/update visible, delete hidden
- Test each page with viewer user - only list/read visible
- Test users/roles pages with non-admin user - entire page hidden or redirect

### Phase 5: Form Guards and Route Protection

**Frontend Changes:**

`webui/src/app.ts`:
- Add route guard function that checks permissions before allowing navigation
- On route change, check if user has permission for that resource
- Redirect to home or show 403 if denied

`webui/src/index.html`:
- Add 403 page template for permission denied

`webui/src/components/*.ts`:
- For detail pages (e.g., `/devices/detail`): check `canRead('devices')` on load
- For create forms: check `canCreate('resource')` on load
- For edit forms: check `canUpdate('resource')` on load
- Show "Access Denied" message if permission check fails

**Route protection example:**
```typescript
function checkRoutePermission(path: string): boolean {
  const resource = getRouteResource(path);
  const action = getRouteAction(path);

  if (resource === 'users' && !permissions.canList('users')) {
    return false;
  }
  // ... other routes
  return true;
}

// In Alpine data
beforeEnterRoute(path: string) {
  if (!checkRoutePermission(path)) {
    window.location.href = '/403';
  }
}
```

**Form guard example:**
```typescript
// In devices.ts
init() {
  if (this.isEditMode && !permissions.canUpdate('devices')) {
    this.error = 'You do not have permission to edit devices';
    this.loading = false;
    return;
  }
  if (!this.isEditMode && !permissions.canCreate('devices')) {
    this.error = 'You do not have permission to create devices';
    this.loading = false;
    return;
  }
  // ... continue loading
}
```

**Verification:**
- Test direct URL access to `/users` with non-admin user -> redirect/403
- Test direct URL access to `/devices/add` with read-only user -> redirect/403
- Test direct URL access to `/devices/edit?id=X` with read-only user -> redirect/403

### Phase 6: Error Handling and Feedback

**Frontend Changes:**

`webui/src/core/api-client.ts`:
- Enhance error handling for 403 responses
- Show user-friendly message: "You don't have permission to perform this action"

`webui/src/core/toast.ts` (or create):
- Add toast/notification component for error messages
- Show permission denied messages in toast

`webui/src/index.html`:
- Add 403 Forbidden page template with:
  - Clear message about insufficient permissions
  - Link to contact admin or go back
  - Suggestion to check with admin for access

**Example 403 page:**
```html
<template x-for="403">
  <div class="error-page">
    <h1>Access Denied</h1>
    <p>You don't have permission to access this resource.</p>
    <p>Required permission: <span x-text="requiredPermission"></span></p>
    <a href="/">Go Home</a>
  </div>
</template>
```

**Verification:**
- Test API 403 errors show user-friendly toast message
- Test 403 page displays correctly
- Test navigation back from 403 page works

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
| `webui/src/api/client.ts` | Add `getUserPermissions()` method |
| `webui/src/app.ts` | Add permission state, helpers, route guards |
| `webui/src/components/nav.ts` | Filter nav items by permissions |
| `webui/src/components/devices.ts` | Add action button guards |
| `webui/src/components/networks.ts` | Add action button guards |
| `webui/src/components/datacenters.ts` | Add action button guards |
| `webui/src/components/discovery.ts` | Add action button guards |
| `webui/src/components/credentials.ts` | Add action button guards |
| `webui/src/components/profiles.ts` | Add action button guards |
| `webui/src/components/scheduled-scans.ts` | Add action button guards |
| `webui/src/components/users.ts` | Add component-level and action guards |
| `webui/src/components/roles.ts` | Add component-level and action guards |
| `webui/src/index.html` | Add 403 page template |

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

1. **Phase 1**: Enhanced `/api/auth/me` response (backend only, no UI changes)
2. **Phase 2**: UI permission state (no visible changes)
3. **Phase 3**: Navigation filtering (first visible change)
4. **Phase 4**: Action button guards (biggest UX change)
5. **Phase 5**: Form/route guards (security hardening)
6. **Phase 6**: Error handling (UX polish)
7. **Phase 7**: Polish (final touches)

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
