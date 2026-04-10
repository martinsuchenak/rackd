# Role-Based Access Control (RBAC)

Rackd implements role-based access control (RBAC) for fine-grained permission management.

## Overview

RBAC in Rackd follows the standard model:

- **Users** - Accounts that can authenticate
- **Roles** - Collections of permissions
- **Permissions** - Fine-grained `resource:action` access controls

Users are assigned roles, and roles grant permissions. A user's effective permissions are the union of all permissions from their assigned roles.

## Default Roles

Rackd includes three built-in system roles that cannot be modified or deleted:

### Admin

Full administrative access with all permissions. Can manage all resources including users, roles, and system configuration.

### Operator

Operational access for day-to-day management:

| Resource | Actions |
|----------|---------|
| Devices | list, create, read, update |
| Networks | list, create, read, update |
| Datacenters | list, read |
| Discovery | list, create, read |

Cannot delete resources or manage users/roles.

### Viewer

Read-only access to view resources:

| Resource | Actions |
|----------|---------|
| Devices | list, read |
| Networks | list, read |
| Datacenters | list, read |
| Discovery | list, read |

## Permissions Reference

### Devices

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `device:list` | devices | list | List all devices |
| `device:create` | devices | create | Create new devices |
| `device:read` | devices | read | View device details |
| `device:update` | devices | update | Modify devices |
| `device:delete` | devices | delete | Delete devices |

### Networks

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `network:list` | networks | list | List all networks |
| `network:create` | networks | create | Create new networks |
| `network:read` | networks | read | View network details |
| `network:update` | networks | update | Modify networks |
| `network:delete` | networks | delete | Delete networks |

### Datacenters

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `datacenter:list` | datacenters | list | List all datacenters |
| `datacenter:create` | datacenters | create | Create new datacenters |
| `datacenter:read` | datacenters | read | View datacenter details |
| `datacenter:update` | datacenters | update | Modify datacenters |
| `datacenter:delete` | datacenters | delete | Delete datacenters |

### Pools

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `pool:list` | pools | list | List all IP pools |
| `pool:create` | pools | create | Create new pools |
| `pool:read` | pools | read | View pool details |
| `pool:update` | pools | update | Modify pools |
| `pool:delete` | pools | delete | Delete pools |

### NAT

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `nat:list` | nat | list | List all NAT rules |
| `nat:create` | nat | create | Create NAT rules |
| `nat:read` | nat | read | View NAT rule details |
| `nat:update` | nat | update | Modify NAT rules |
| `nat:delete` | nat | delete | Delete NAT rules |

### Circuits

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `circuits:list` | circuits | list | List all circuits |
| `circuits:create` | circuits | create | Create circuits |
| `circuits:read` | circuits | read | View circuit details |
| `circuits:update` | circuits | update | Modify circuits |
| `circuits:delete` | circuits | delete | Delete circuits |

### Discovery

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `discovery:list` | discovery | list | List discovered devices |
| `discovery:create` | discovery | create | Start discovery scans |
| `discovery:read` | discovery | read | View discovery results |
| `discovery:update` | discovery | update | Update discovery settings |
| `discovery:delete` | discovery | delete | Delete discovered devices |

### Credentials

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `credential:list` | credentials | list | List all credentials |
| `credential:create` | credentials | create | Create credentials |
| `credential:read` | credentials | read | View credential details |
| `credential:update` | credentials | update | Modify credentials |
| `credential:delete` | credentials | delete | Delete credentials |

### Scan Profiles

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `scan-profile:list` | scan-profiles | list | List all scan profiles |
| `scan-profile:create` | scan-profiles | create | Create scan profiles |
| `scan-profile:read` | scan-profiles | read | View scan profile details |
| `scan-profile:update` | scan-profiles | update | Modify scan profiles |
| `scan-profile:delete` | scan-profiles | delete | Delete scan profiles |

### Scheduled Scans

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `scheduled-scan:list` | scheduled-scans | list | List all scheduled scans |
| `scheduled-scan:create` | scheduled-scans | create | Create scheduled scans |
| `scheduled-scan:read` | scheduled-scans | read | View scheduled scan details |
| `scheduled-scan:update` | scheduled-scans | update | Modify scheduled scans |
| `scheduled-scan:delete` | scheduled-scans | delete | Delete scheduled scans |

### Relationships

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `relationship:list` | relationships | list | List all relationships |
| `relationship:create` | relationships | create | Create relationships |
| `relationship:read` | relationships | read | View relationship details |
| `relationship:update` | relationships | update | Modify relationships |
| `relationship:delete` | relationships | delete | Delete relationships |

### Conflicts

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `conflict:list` | conflicts | list | List all conflicts |
| `conflict:read` | conflicts | read | View conflict details |
| `conflict:detect` | conflicts | detect | Run conflict detection |
| `conflict:resolve` | conflicts | resolve | Resolve conflicts |
| `conflict:delete` | conflicts | delete | Delete conflict records |

### Reservations

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `reservation:list` | reservations | list | List all reservations |
| `reservation:create` | reservations | create | Create reservations |
| `reservation:read` | reservations | read | View reservation details |
| `reservation:update` | reservations | update | Modify reservations |
| `reservation:delete` | reservations | delete | Delete reservations |

### DNS Providers

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `dns-provider:list` | dns-provider | list | List all DNS providers |
| `dns-provider:create` | dns-provider | create | Create DNS providers |
| `dns-provider:read` | dns-provider | read | View DNS provider details |
| `dns-provider:update` | dns-provider | update | Modify DNS providers |
| `dns-provider:delete` | dns-provider | delete | Delete DNS providers |
| `dns-provider:test` | dns-provider | test | Test DNS provider connection |

### DNS Zones

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `dns-zone:list` | dns-zone | list | List all DNS zones |
| `dns-zone:create` | dns-zone | create | Create DNS zones |
| `dns-zone:read` | dns-zone | read | View DNS zone details |
| `dns-zone:update` | dns-zone | update | Modify DNS zones |
| `dns-zone:delete` | dns-zone | delete | Delete DNS zones |
| `dns-zone:sync` | dns-zone | sync | Sync DNS zone records |
| `dns-zone:import` | dns-zone | import | Import DNS zone records |

### DNS Records

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `dns:list` | dns | list | List all DNS records |
| `dns:create` | dns | create | Create DNS records |
| `dns:read` | dns | read | View DNS record details |
| `dns:update` | dns | update | Modify DNS records |
| `dns:delete` | dns | delete | Delete DNS records |
| `dns:sync` | dns | sync | Sync DNS records |
| `dns:import` | dns | import | Import DNS records |

### Users

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `user:list` | users | list | List all users |
| `user:create` | users | create | Create new users |
| `user:read` | users | read | View user details |
| `user:update` | users | update | Modify users |
| `user:delete` | users | delete | Delete users |

### Roles

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `role:list` | roles | list | List all roles |
| `role:create` | roles | create | Create custom roles |
| `role:read` | roles | read | View role details |
| `role:update` | roles | update | Modify roles |
| `role:delete` | roles | delete | Delete custom roles |

### API Keys

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `apikey:list` | apikeys | list | List API keys |
| `apikey:create` | apikeys | create | Create API keys |
| `apikey:read` | apikeys | read | View API key details |
| `apikey:update` | apikeys | update | Modify API keys |
| `apikey:delete` | apikeys | delete | Delete API keys |

### Webhooks

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `webhooks:list` | webhooks | list | List all webhooks |
| `webhooks:create` | webhooks | create | Create webhooks |
| `webhooks:read` | webhooks | read | View webhook details |
| `webhooks:update` | webhooks | update | Modify webhooks |
| `webhooks:delete` | webhooks | delete | Delete webhooks |

### Custom Fields

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `custom-fields:list` | custom-fields | list | List all custom fields |
| `custom-fields:create` | custom-fields | create | Create custom fields |
| `custom-fields:read` | custom-fields | read | View custom field details |
| `custom-fields:update` | custom-fields | update | Modify custom fields |
| `custom-fields:delete` | custom-fields | delete | Delete custom fields |

### Audit Logs

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `audit:list` | audit | list | View audit logs |
| `audit:read` | audit | read | View individual audit log entries |
| `audit:export` | audit | export | Export audit logs |

### Application Logs

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `logs:list` | logs | list | View recent application logs |
| `logs:read` | logs | read | View individual application log entries |
| `logs:export` | logs | export | Export recent application logs |

### Search

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `search:read` | search | read | Search across resources |

### Dashboard

| Permission | Resource | Action | Description |
|------------|----------|--------|-------------|
| `dashboard:read` | dashboard | read | View dashboard |

## Managing Roles

### CLI Commands

```bash
# List roles
rackd role list

# List available permissions
rackd role permissions

# Filter permissions by resource
rackd role permissions --resource devices

# Create a custom role
rackd role create --name "Network Admin" --description "Full network management"

# Assign role to user
rackd role assign --user-id user-123 --role-id role-456

# Revoke role from user
rackd role revoke --user-id user-123 --role-id role-456

# Delete a custom role
rackd role delete --id role-456
```

### Web UI

1. Navigate to **Settings → Roles**
2. View existing roles and their permissions
3. Create custom roles with specific permission sets
4. Assign roles to users from the user management page

## API Endpoints

### Roles

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/roles` | List roles |
| GET | `/api/roles/{id}` | Get role details |
| POST | `/api/roles` | Create role |
| DELETE | `/api/roles/{id}` | Delete role |

### Permissions

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/permissions` | List permissions |

### User Role Assignment

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/users/grant-role` | Assign role to user |
| POST | `/api/users/revoke-role` | Revoke role from user |

## Permission Checking

### In Code

Permissions are checked at the service layer using the `requirePermission` function:

```go
func (s *DeviceService) Create(ctx context.Context, device *model.Device) error {
    if err := requirePermission(ctx, s.checker, "devices", "create"); err != nil {
        return err
    }
    // ... create device
}
```

### For API Keys

API keys inherit permissions from their owning user. When an API key makes a request:

1. The system resolves the user associated with the API key
2. Checks the user's roles and permissions
3. Allows or denies the action based on permissions

### For OAuth Tokens (MCP)

OAuth tokens can be scoped to specific permissions. The token's scope restricts what actions can be performed, even if the user has broader permissions.

## Best Practices

1. **Principle of Least Privilege** - Assign only the minimum required permissions
2. **Use Built-in Roles** - Start with viewer/operator before creating custom roles
3. **Regular Audits** - Periodically review role assignments
4. **Document Custom Roles** - Maintain clear descriptions of custom role purposes
5. **Separate Duties** - Use different roles for different operational areas

## Related Documentation

- [User Authentication](user-authentication.md) - Authentication methods
- [API Reference](api.md) - REST API endpoints
- [CLI Reference](cli.md) - Role management commands
