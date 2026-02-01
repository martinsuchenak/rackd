// Navigation Component for Rackd Web UI

import type { NavItem, UIConfig } from '../core/types';

interface NavData {
  config: UIConfig | null;
  loading: boolean;
  items: NavItem[];
  hasFeature(name: string): boolean;
  isEnterprise: boolean;
  init(): Promise<void>;
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
    items: [],

    hasFeature(name: string): boolean {
      return this.config?.features.includes(name) ?? false;
    },

    get isEnterprise(): boolean {
      return this.config?.edition === 'enterprise';
    },

    async init(): Promise<void> {
      try {
        const response = await fetch('/api/config');
        if (response.ok) {
          this.config = await response.json();
          const dynamicItems = this.config?.nav_items ?? [];
          this.items = [...baseNavItems, ...dynamicItems].sort((a, b) => a.order - b.order);
        } else {
          this.items = baseNavItems;
        }
      } catch {
        this.items = baseNavItems;
      } finally {
        this.loading = false;
      }
    },
  };
}
