import { describe, expect, test } from 'bun:test';

import { activeSectionId, canAccessRoute, filterNavItems, getPageTitle, groupNavItems, isActiveNavItem, mergeNavItems, missingSetting } from '../src/core/features';
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

    expect(merged.some((item) => item.label === 'Users' && item.section === 'administration')).toBe(true);
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

  test('groupNavItems puts core items in their sections, in menu order', () => {
    const sections = groupNavItems(mergeNavItems([], { mcp_oauth: true }));
    expect(sections.map((s) => s.label)).toEqual(['', 'Organization', 'Inventory', 'IPAM', 'DNS', 'Discovery', 'Integrations', 'Administration']);
    expect(sections[0].items.map((i) => i.label)).toEqual(['Dashboard']);
    const ipam = sections.find((s) => s.id === 'ipam')!;
    expect(ipam.items.map((i) => i.label)).toEqual(['Networks', 'NAT', 'Conflicts']);
    const dns = sections.find((s) => s.id === 'dns')!;
    expect(dns.items.map((i) => i.label)).toEqual(['Zones', 'Providers']);
  });

  test('extension items join a core section, their own, or Extensions, before Administration', () => {
    const dynamic: NavItem[] = [
      { label: 'Racks', path: '/ext/racks', order: 5, section: 'inventory' },
      { label: 'Reports', path: '/ext/reports', order: 1, section: 'Reporting' },
      { label: 'Hello', path: '/ext/hello', order: 1 },
    ];
    const sections = groupNavItems(mergeNavItems(dynamic, {}));
    const labels = sections.map((s) => s.label);
    expect(sections.find((s) => s.id === 'inventory')!.items.map((i) => i.label)).toEqual(['Racks', 'Devices', 'Circuits']);
    expect(labels.indexOf('Reporting')).toBeGreaterThan(labels.indexOf('Integrations'));
    expect(labels.indexOf('Extensions')).toBeGreaterThan(labels.indexOf('Integrations'));
    expect(labels[labels.length - 1]).toBe('Administration');
    // A label that matches a core item's is fine on another path.
    const providers = groupNavItems(mergeNavItems([{ label: 'Providers', path: '/ext/billing', order: 1, section: 'Billing' }], {}));
    expect(providers.find((s) => s.label === 'Billing')!.items.map((i) => i.label)).toEqual(['Providers']);
    // An extension item without a section never lands above the sections.
    expect(sections[0].items.map((i) => i.label)).toEqual(['Dashboard']);
  });

  test('sections without visible items are dropped, and the active section is found', () => {
    const viewer = filterNavItems(mergeNavItems([], {}), permissions); // users:list and dns-zone:list only
    const sections = groupNavItems(viewer);
    expect(sections.map((s) => s.label)).toEqual(['', 'DNS', 'Administration']);
    expect(activeSectionId(sections, '/dns/zones/zone-1')).toBe('dns');
    expect(activeSectionId(sections, '/')).toBe('');
    expect(isActiveNavItem({ label: 'Networks', path: '/networks', order: 1 }, '/pools/detail?id=1')).toBe(true);
    expect(isActiveNavItem({ label: 'Devices', path: '/devices', order: 1 }, '/devicesx')).toBe(false);
  });
});
