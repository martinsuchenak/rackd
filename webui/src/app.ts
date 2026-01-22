// Rackd Web UI - Main Application Entry Point

import Alpine from 'alpinejs';
import type { UIConfig } from './core/types';
import { RackdAPI } from './core/api';

// Components
import { nav } from './components/nav';
import { globalSearch } from './components/search';
import { deviceList, deviceDetail, deviceForm } from './components/devices';
import { networkList, networkDetail, networkForm } from './components/networks';
import { poolDetail, poolForm } from './components/pools';
import { datacenterList, datacenterDetail, datacenterForm } from './components/datacenters';
import { discoveryList, scanForm, scanDetail, promoteForm } from './components/discovery';

declare global {
  interface Window {
    Alpine: typeof Alpine;
    rackdConfig: UIConfig | null;
    rackdEnterprise?: { init(): void };
  }
}

// Theme management
type Theme = 'light' | 'dark' | 'system';

function getStoredTheme(): Theme {
  return (localStorage.getItem('theme') as Theme) || 'system';
}

function applyTheme(theme: Theme): void {
  const isDark = theme === 'dark' || (theme === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches);
  document.documentElement.classList.toggle('dark', isDark);
}

function themeToggle() {
  return {
    theme: getStoredTheme(),
    init() {
      applyTheme(this.theme);
      window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
        if (this.theme === 'system') applyTheme('system');
      });
    },
    setTheme(t: Theme) {
      this.theme = t;
      localStorage.setItem('theme', t);
      applyTheme(t);
    },
  };
}

// Initialize application
async function init(): Promise<void> {
  const api = new RackdAPI();
  
  // Fetch config
  try {
    window.rackdConfig = await api.getConfig();
  } catch {
    window.rackdConfig = { edition: 'oss', features: [], nav_items: [] };
  }

  // Register Alpine components
  Alpine.data('nav', nav);
  Alpine.data('globalSearch', globalSearch);
  Alpine.data('themeToggle', themeToggle);
  Alpine.data('deviceList', deviceList);
  Alpine.data('deviceDetail', deviceDetail);
  Alpine.data('deviceForm', deviceForm);
  Alpine.data('networkList', networkList);
  Alpine.data('networkDetail', networkDetail);
  Alpine.data('networkForm', networkForm);
  Alpine.data('poolDetail', poolDetail);
  Alpine.data('poolForm', poolForm);
  Alpine.data('datacenterList', datacenterList);
  Alpine.data('datacenterDetail', datacenterDetail);
  Alpine.data('datacenterForm', datacenterForm);
  Alpine.data('discoveryList', discoveryList);
  Alpine.data('scanForm', scanForm);
  Alpine.data('scanDetail', scanDetail);
  Alpine.data('promoteForm', promoteForm);

  // Enterprise extension hook
  window.rackdEnterprise?.init();

  // Expose Alpine globally
  window.Alpine = Alpine;

  // Start Alpine
  Alpine.start();
}

init();
