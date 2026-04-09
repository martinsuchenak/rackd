import { describe, expect, test } from 'bun:test';

import { canAccessRoute, filterNavItems, getPageTitle, mergeNavItems } from '../src/core/features';
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
});
