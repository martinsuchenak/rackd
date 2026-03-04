// Navigation Component for Rackd Web UI

import type { NavItem, UIConfig, Permission } from '../core/types';

interface NavData {
  config: UIConfig | null;
  loading: boolean;
  items: NavItem[];
  hasFeature(name: string): boolean;
  init(): Promise<void>;
  get filteredItems(): NavItem[];
}

const baseNavItems: NavItem[] = [
  { label: 'Dashboard', path: '/', icon: 'home', order: 0 },
  { label: 'Devices', path: '/devices', icon: 'server', order: 10 },
  { label: 'Networks', path: '/networks', icon: 'network', order: 20 },
  { label: 'Datacenters', path: '/datacenters', icon: 'building', order: 30 },
  { label: 'Discovery', path: '/discovery', icon: 'search', order: 40 },
  { label: 'Credentials', path: '/credentials', icon: 'key', order: 42, required_permissions: [{ resource: 'credentials', action: 'list' }] },
  { label: 'Scan Profiles', path: '/scan-profiles', icon: 'settings', order: 44, required_permissions: [{ resource: 'discovery', action: 'list' }] },
  { label: 'Scheduled Scans', path: '/scheduled-scans', icon: 'clock', order: 46, required_permissions: [{ resource: 'discovery', action: 'list' }] },
  { label: 'Conflicts', path: '/conflicts', icon: 'warning', order: 50 },
  { label: 'Webhooks', path: '/webhooks', icon: 'zap', order: 51, required_permissions: [{ resource: 'webhooks', action: 'list' }] },
  { label: 'Custom Fields', path: '/custom-fields', icon: 'tag', order: 52, required_permissions: [{ resource: 'custom-fields', action: 'list' }] },
  { label: 'Circuits', path: '/circuits', icon: 'shuffle', order: 55 },
  { label: 'NAT', path: '/nat', icon: 'git-branch', order: 56 },
  { label: 'DNS Providers', path: '/dns/providers', icon: 'server', order: 57, required_permissions: [{ resource: 'dns', action: 'list' }] },
  { label: 'DNS Zones', path: '/dns/zones', icon: 'globe', order: 58, required_permissions: [{ resource: 'dns', action: 'list' }] },
  { label: 'Users', path: '/users', icon: 'user', order: 90, required_permissions: [{ resource: 'users', action: 'list' }] },
  { label: 'Roles', path: '/roles', icon: 'shield', order: 91, required_permissions: [{ resource: 'roles', action: 'list' }] },
];

export function nav(): NavData {
  return {
    config: null,
    loading: true,
    items: baseNavItems,

    get filteredItems(): NavItem[] {
      if (!this.config) {
        return this.items;
      }
      const dynamic = (this.config.nav_items ?? []).filter(
        (item: NavItem) => !baseNavItems.some((b) => (b.path === item.path || (b.path === '/' && item.path === '')) || b.label === item.label)
      );
      const allItems = [...baseNavItems, ...dynamic].sort((a, b) => a.order - b.order);
      const userPermissions = this.config.user?.permissions ?? [];

      return allItems.filter((item: NavItem) => {
        if (!item.required_permissions || item.required_permissions.length === 0) {
          return true;
        }
        return item.required_permissions.every((req) =>
          userPermissions.some((perm: Permission) =>
            perm.resource === req.resource && perm.action === req.action
          )
        );
      });
    },

    hasFeature(name: string): boolean {
      return this.config?.features.includes(name) ?? false;
    },

    async init(): Promise<void> {
      try {
        const response = await fetch('/api/config');
        if (response.ok) {
          this.config = await response.json();
        }
      } catch {
        this.config = null;
      } finally {
        this.loading = false;
      }
    },
  };
}
