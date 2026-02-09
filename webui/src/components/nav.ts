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
  { label: 'Devices', path: '/devices', icon: 'server', order: 10 },
  { label: 'Networks', path: '/networks', icon: 'network', order: 20 },
  { label: 'Datacenters', path: '/datacenters', icon: 'building', order: 30 },
  { label: 'Discovery', path: '/discovery', icon: 'search', order: 40 },
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
      const allItems = [...baseNavItems, ...(this.config.nav_items ?? [])];
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
