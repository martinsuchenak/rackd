import { describe, expect, test } from 'bun:test';

import { canAccessRoute, filterNavItems, getPageTitle, mergeNavItems, missingSetting } from '../src/core/features';
import type { NavItem, Permission } from '../src/core/types';

const permissions: Permission[] = [
  {
    id: 'perm-users-list',
    name: 'users.list',
    resource: 'users',
    action: 'list',
    created_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 'perm-dns-zone-list',
    name: 'dns-zone.list',
    resource: 'dns-zone',
    action: 'list',
    created_at: '2026-01-01T00:00:00Z',
  },
];

describe('core/features', () => {
  test('canAccessRoute uses feature permissions and route prefixes', () => {
    expect(canAccessRoute('/users', permissions)).toBe(true);
    expect(canAccessRoute('/dns/zones/123', permissions)).toBe(true);
    expect(canAccessRoute('/webhooks', permissions)).toBe(false);
    expect(canAccessRoute('/login', permissions)).toBe(true);
  });

  test('getPageTitle resolves feature titles for detail and nested routes', () => {
    expect(getPageTitle('/devices/detail?id=123')).toBe('Device Details');
    expect(getPageTitle('/dns/zones/zone-1')).toBe('DNS Zones');
    expect(getPageTitle('/unknown')).toBe('Page');
  });

  test('filterNavItems removes entries without required permissions', () => {
    const items: NavItem[] = [
      { label: 'Public', path: '/public', order: 1 },
      {
        label: 'Users',
        path: '/users',
        order: 2,
        required_permissions: [{ resource: 'users', action: 'list' }],
      },
      {
        label: 'Webhooks',
        path: '/webhooks',
        order: 3,
        required_permissions: [{ resource: 'webhooks', action: 'list' }],
      },
    ];

    expect(filterNavItems(items, permissions).map((item) => item.label)).toEqual(['Public', 'Users']);
  });

  test('mergeNavItems preserves base items and deduplicates matching dynamic entries', () => {
    const dynamicItems: NavItem[] = [
      { label: 'Users', path: '/users', order: 999 },
      { label: 'Custom', path: '/custom', order: 200 },
    ];

    const merged = mergeNavItems(dynamicItems);

    expect(merged.some((item) => item.label === 'Users' && item.order === 90)).toBe(true);
    expect(merged.filter((item) => item.label === 'Users')).toHaveLength(1);
    expect(merged.some((item) => item.label === 'Custom' && item.path === '/custom')).toBe(true);
  });

  test('the OAuth Clients page exists only when MCP OAuth is on', () => {
    const labels = (settings: { mcp_oauth?: boolean } | null) =>
      mergeNavItems([], settings).map((item) => item.label);

    expect(labels({ mcp_oauth: false })).not.toContain('OAuth Clients');
    expect(labels(null)).not.toContain('OAuth Clients');
    expect(labels({ mcp_oauth: true })).toContain('OAuth Clients');

    // Off is "not on this server", not "access denied".
    expect(missingSetting('/oauth-clients', { mcp_oauth: false })).toBe('mcp_oauth');
    expect(missingSetting('/oauth-clients', { mcp_oauth: true })).toBeNull();
    expect(missingSetting('/users', { mcp_oauth: false })).toBeNull();
    expect(canAccessRoute('/oauth-clients', permissions)).toBe(true);

    // An extension can't take the slot of the hidden core entry.
    const extension: NavItem[] = [{ label: 'OAuth Clients', path: '/ext-oauth', order: 5 }];
    expect(mergeNavItems(extension, { mcp_oauth: false }).map((item) => item.label)).not.toContain('OAuth Clients');
  });
});
