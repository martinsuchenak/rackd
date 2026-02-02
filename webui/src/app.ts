// Rackd Web UI - Main Application Entry Point

import Alpine from 'alpinejs';
import type { UIConfig } from './core/types';
import { api, RackdAPI } from './core/api';

// Components
import { nav } from './components/nav';
import { globalSearch } from './components/search';
import { deviceList, deviceDetail, deviceForm } from './components/devices';
import { networkList, networkDetail, networkForm } from './components/networks';
import { poolDetail, poolForm } from './components/pools';
import { datacenterList, datacenterDetail, datacenterForm } from './components/datacenters';
import { discoveryList, scanForm, scanDetail, promoteForm } from './components/discovery';
import { credentialsList, credentialForm, credentialsPageTemplate } from './components/credentials';
import { profileList, profileForm, profilesPageTemplate } from './components/profiles';
import { scheduledScansList, scheduledScanForm, scheduledScansPageTemplate } from './components/scheduled-scans';

// Page registry for extensions
interface ExtensionPage {
  path: string;
  render: () => string; // Returns HTML template with x-data binding
}

const extensionPages: ExtensionPage[] = [];

// Scan type registry for extensions
interface ScanType {
  value: string;
  label: string;
  description?: string;
}

const baseScanTypes: ScanType[] = [
  { value: 'quick', label: 'Quick', description: 'ICMP ping' },
  { value: 'full', label: 'Full', description: 'TCP port scan' },
];

const extensionScanTypes: ScanType[] = [];

declare global {
  interface Window {
    Alpine: typeof Alpine;
    rackdAPI: RackdAPI;
    rackdConfig: UIConfig | null;
    rackdEnterprise?: { init(): void };
    rackdRegisterPage: (path: string, render: () => string) => void;
    rackdExtensionPages: ExtensionPage[];
    rackdRegisterScanType: (type: ScanType) => void;
    rackdScanTypes: ScanType[];
  }
}

// Extension API - called by enterprise/plugins
window.rackdRegisterPage = (path: string, render: () => string) => {
  extensionPages.push({ path, render });
};

window.rackdRegisterScanType = (type: ScanType) => {
  extensionScanTypes.push(type);
  // Update the exposed array
  window.rackdScanTypes = [...baseScanTypes, ...extensionScanTypes];
};

// Expose for components
window.rackdExtensionPages = extensionPages;
window.rackdScanTypes = [...baseScanTypes];

// Router component for SPA navigation
function router() {
  return {
    route: window.location.pathname + window.location.search,
    sidebarOpen: false,

    // Nav items from config (core + enterprise)
    get navItems() {
      const base = [
        { label: 'Devices', path: '/devices', order: 10 },
        { label: 'Networks', path: '/networks', order: 20 },
        { label: 'Datacenters', path: '/datacenters', order: 30 },
        { label: 'Discovery', path: '/discovery', order: 40 },
      ];
      const dynamic = window.rackdConfig?.nav_items ?? [];
      return [...base, ...dynamic].sort((a, b) => a.order - b.order);
    },

    // Check if current route is an extension page
    get extensionPage() {
      return window.rackdExtensionPages?.find(
        (p) => this.route === p.path || this.route.startsWith(p.path + '?')
      );
    },

    // Get rendered content for extension page
    get extensionContent() {
      return this.extensionPage?.render() || '';
    },

    init() {
      window.addEventListener('popstate', () => {
        this.route = window.location.pathname + window.location.search;
      });
    },

    navigate(path: string) {
      if (path !== this.route) {
        history.pushState({}, '', path);
        this.route = path;
        this.sidebarOpen = false;
      }
    },
  };
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
  window.rackdAPI = api;

  // Fetch config
  try {
    window.rackdConfig = await api.getConfig();
  } catch {
    window.rackdConfig = { edition: 'oss', features: [], nav_items: [] };
  }

  // Register Alpine components
  Alpine.data('router', router);
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

  // Credentials, Profiles, Scheduled Scans components
  Alpine.data('credentialsList', credentialsList);
  Alpine.data('credentialForm', credentialForm);
  Alpine.data('profileList', profileList);
  Alpine.data('profileForm', profileForm);
  Alpine.data('scheduledScansList', scheduledScansList);
  Alpine.data('scheduledScanForm', scheduledScanForm);

  // Register pages for credentials, profiles, scheduled scans
  window.rackdRegisterPage('/credentials', credentialsPageTemplate);
  window.rackdRegisterPage('/scan-profiles', profilesPageTemplate);
  window.rackdRegisterPage('/scheduled-scans', scheduledScansPageTemplate);

  // Register deep scan type
  window.rackdRegisterScanType({
    value: 'deep',
    label: 'Deep',
    description: 'Comprehensive scan with SNMP/SSH',
  });

  // Expose Alpine globally (before enterprise init)
  window.Alpine = Alpine;

  // Enterprise extension hook (preserved for future extensibility)
  window.rackdEnterprise?.init();

  // Start Alpine
  Alpine.start();
}

init();
