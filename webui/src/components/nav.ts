// Navigation Component for Rackd Web UI

import type { NavItem, UIConfig } from '../core/types';
import { filterNavItems, mergeNavItems } from '../core/features';

interface NavData {
  config: UIConfig | null;
  loading: boolean;
  items: NavItem[];
  hasFeature(name: string): boolean;
  init(): Promise<void>;
  get filteredItems(): NavItem[];
}

export function nav(): NavData {
  return {
    config: null,
    loading: true,
    items: mergeNavItems([]),

    get filteredItems(): NavItem[] {
      if (!this.config) {
        return this.items;
      }
      const allItems = mergeNavItems(this.config.nav_items ?? []);
      return filterNavItems(allItems, this.config.user?.permissions ?? []);
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
