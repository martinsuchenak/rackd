// Rackd Web UI - Main Application Entry Point

import Alpine from 'alpinejs';
import focus from '@alpinejs/focus';
import collapse from '@alpinejs/collapse';
import type { UIConfig, Permission, Role } from './core/types';
import { api, RackdAPI } from './core/api';

// Components
import { nav } from './components/nav';
import { globalSearch } from './components/search';
import { deviceList, deviceDetail, deviceForm } from './components/devices';
import { deviceGraph } from './components/graph';
import { networkList, networkDetail, networkForm } from './components/networks';
import { poolDetail, poolForm } from './components/pools';
import { datacenterList, datacenterDetail, datacenterForm } from './components/datacenters';
import { discoveryList, scanForm, scanDetail, promoteForm } from './components/discovery';
import { credentialsList, credentialForm, credentialsPageTemplate } from './components/credentials';
import { profileList, profileForm, profilesPageTemplate } from './components/profiles';
import { scheduledScansList, scheduledScanForm, scheduledScansPageTemplate } from './components/scheduled-scans';
import { scanProfilesList } from './components/scan-profiles';
import { login } from './components/login';
import { usersList } from './components/users';
import { userMenu } from './components/user-menu';
import { toastComponent } from './components/toast';
import { oauthConsent } from './components/oauth-consent';
import { oauthClients } from './components/oauth-clients';
import { conflictList } from './components/conflicts';
import { webhookComponent } from './components/webhooks';
import { customFieldComponent } from './components/custom-fields';
import { dashboardComponent } from './components/dashboard';
import { circuitComponent } from './components/circuits';
import { natComponent } from './components/nat';
import { dnsProvidersComponent, dnsZonesComponent, dnsRecordsComponent } from './components/dns';

// Update page title based on route
function updatePageTitle(route: string) {
  const titles: Record<string, string> = {
    '/': 'Dashboard',
    '/devices': 'Devices',
    '/devices/detail': 'Device Details',
    '/devices/graph': 'Device Relationships Graph',
    '/networks': 'Networks',
    '/networks/detail': 'Network Details',
    '/pools/detail': 'Pool Details',
    '/datacenters': 'Datacenters',
    '/datacenters/detail': 'Datacenter Details',
    '/discovery': 'Discovery',
    '/scan-profiles': 'Scan Profiles',
    '/conflicts': 'IP Conflicts',
    '/circuits': 'Circuits',
    '/nat': 'NAT Mappings',
    '/dns/providers': 'DNS Providers',
    '/dns/zones': 'DNS Zones',
    '/dns/records': 'DNS Records',
  };
  const path = route.split('?')[0];
  document.title = `${titles[path] || 'Page'} - Rackd`;
}

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
    rackdRegisterPage: (path: string, render: () => string) => void;
    rackdExtensionPages: ExtensionPage[];
    rackdRegisterScanType: (type: ScanType) => void;
    rackdScanTypes: ScanType[];
  }
}

// Extension API - for plugins
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

// Route permission requirements - maps route prefixes to required permissions
const routePermissions: { prefix: string; resource: string; action: string }[] = [
  { prefix: '/users', resource: 'users', action: 'list' },
  { prefix: '/roles', resource: 'roles', action: 'list' },
  { prefix: '/devices', resource: 'devices', action: 'list' },
  { prefix: '/networks', resource: 'networks', action: 'list' },
  { prefix: '/pools', resource: 'networks', action: 'list' },
  { prefix: '/datacenters', resource: 'datacenters', action: 'list' },
  { prefix: '/discovery', resource: 'discovery', action: 'list' },
];

function checkRoutePermission(path: string): boolean {
  const cleanPath = path.split('?')[0];
  const userPermissions = (window.rackdConfig?.user?.permissions ?? []) as any[];
  for (const rule of routePermissions) {
    if (cleanPath === rule.prefix || cleanPath.startsWith(rule.prefix + '/') || cleanPath.startsWith(rule.prefix + '?')) {
      return userPermissions.some(
        (p: any) => p.resource === rule.resource && p.action === rule.action
      );
    }
  }
  // No permission rule = allow (dashboard, login, etc.)
  return true;
}

// Router component for SPA navigation
function router() {
  return {
    route: window.location.pathname + window.location.search,
    sidebarOpen: false,
    accessDenied: false,
    activeConflictCount: 0,

    // Nav items from config, filtered by user permissions
    get navItems() {
      const base = [
        { label: 'Devices', path: '/devices', order: 10 },
        { label: 'Networks', path: '/networks', order: 20 },
        { label: 'Datacenters', path: '/datacenters', order: 30 },
        { label: 'Discovery', path: '/discovery', order: 40 },
        { label: 'Conflicts', path: '/conflicts', order: 50, badge: () => this.activeConflictCount },
        { label: 'Circuits', path: '/circuits', order: 55 },
        { label: 'NAT', path: '/nat', order: 56 },
      ];
      const dynamic = window.rackdConfig?.nav_items ?? [];
      const allItems = [...base, ...dynamic].sort((a, b) => a.order - b.order);
      const userPermissions = window.rackdConfig?.user?.permissions ?? [];
      return allItems.filter((item: any) => {
        if (!item.required_permissions || item.required_permissions.length === 0) {
          return true;
        }
        return item.required_permissions.every((req: any) =>
          userPermissions.some((perm: any) =>
            perm.resource === req.resource && perm.action === req.action
          )
        );
      });
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
      this.accessDenied = !checkRoutePermission(this.route);
      updatePageTitle(this.route);
      // Load conflict count on init (only if authenticated)
      if (window.rackdConfig?.user) {
        this.updateConflictCount();
      }
      window.addEventListener('popstate', () => {
        this.route = window.location.pathname + window.location.search;
        this.accessDenied = !checkRoutePermission(this.route);
        updatePageTitle(this.route);
      });
    },

    navigate(path: string) {
      if (path !== this.route) {
        history.pushState({}, '', path);
        this.route = path;
        this.accessDenied = !checkRoutePermission(path);
        updatePageTitle(path);
        this.sidebarOpen = false;
      }
    },

    async updateConflictCount() {
      try {
        const summary = await api.getConflictSummary();
        if (summary) {
          this.activeConflictCount = (summary.duplicate_ips || 0) + (summary.overlapping_subnets || 0);
        }
      } catch {
        // Non-critical, keep default value
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

// Permissions store for checking user permissions (accessible as $store.permissions in all components)
function initPermissionsStore() {
  const userPermissions: Permission[] = (window.rackdConfig?.user?.permissions ?? []) as any;
  const userRoles: Role[] = (window.rackdConfig?.user?.roles ?? []) as any;

  const store = {
    permissions: userPermissions,
    roles: userRoles,
    loaded: true,

    can(resource: string, action: string): boolean {
      return store.permissions.some((p: Permission) =>
        p.resource === resource && p.action === action
      );
    },

    canList(resource: string): boolean {
      return store.can(resource, 'list');
    },

    canRead(resource: string): boolean {
      return store.can(resource, 'read');
    },

    canCreate(resource: string): boolean {
      return store.can(resource, 'create');
    },

    canUpdate(resource: string): boolean {
      return store.can(resource, 'update');
    },

    canDelete(resource: string): boolean {
      return store.can(resource, 'delete');
    },

    hasAnyPermission(resource: string, ...actions: string[]): boolean {
      return actions.some((action) => store.can(resource, action));
    },

    hasAllPermissions(resource: string, ...actions: string[]): boolean {
      return actions.every((action) => store.can(resource, action));
    },
  };

  Alpine.store('permissions', store);
}


// Initialize application
async function init(): Promise<void> {
  window.rackdAPI = api;

  // Fetch config (session cookie is sent automatically)
  try {
    window.rackdConfig = await api.getConfig();
  } catch (error) {
    console.error('Failed to load config:', error);
    window.rackdConfig = { edition: 'oss', features: [], nav_items: [] };
    // Show error toast if loading config fails
    setTimeout(() => {
      window.dispatchEvent(new CustomEvent('toast:error', {
        detail: { message: 'Failed to load application configuration. Some features may not work correctly.' }
      }));
    }, 100);
  }

  // Auth guard: redirect to login if not authenticated (and not already on login page or OAuth consent)
  const isPublicRoute = window.location.pathname === '/login' || window.location.pathname.startsWith('/mcp-oauth/authorize');
  if (!window.rackdConfig?.user && !isPublicRoute) {
    window.location.href = '/login';
    return;
  }

  // Register Alpine components
  Alpine.data('router', router);
  Alpine.data('nav', nav);
  Alpine.data('globalSearch', globalSearch);
  Alpine.data('themeToggle', themeToggle);
  Alpine.data('toast', toastComponent); // Register toast as data component for x-data="toast"
  Alpine.data('deviceList', deviceList);
  Alpine.data('deviceDetail', deviceDetail);
  Alpine.data('deviceForm', deviceForm);
  Alpine.data('deviceGraph', deviceGraph);
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

  // Auth & user management
  Alpine.data('login', login);
  Alpine.data('usersList', usersList);
  Alpine.data('userMenu', userMenu);

  // OAuth
  Alpine.data('oauthConsent', oauthConsent);
  Alpine.data('oauthClients', oauthClients);

  // Permissions store (accessible as $store.permissions in all components)
  initPermissionsStore();

  // Add method to refresh permissions (for role changes)
  Alpine.effect(() => {
    window.addEventListener('permissions:refresh', async () => {
      try {
        const config = await api.getConfig();
        window.rackdConfig = config;
        // Reinitialize permissions store with updated data
        const permissionsStore = Alpine.store('permissions');
        if (permissionsStore) {
          const userPermissions: Permission[] = (config.user?.permissions ?? []) as any;
          const userRoles: Role[] = (config.user?.roles ?? []) as any;
          permissionsStore.permissions = userPermissions;
          permissionsStore.roles = userRoles;
          Alpine.store('toast')?.success('Permissions refreshed successfully');
        }
      } catch (error) {
        console.error('Failed to refresh permissions:', error);
        Alpine.store('toast')?.error('Failed to refresh permissions. Please reload the page.');
      }
    });
  });

  // Toast store for notifications (accessible as $store.toast in all components)
  const toast = toastComponent();
  Alpine.store('toast', toast);

  // Listen for permission denied events from API client
  window.addEventListener('toast:permission-denied', (event: any) => {
    toast.error(event.detail.message);
  });

  // Listen for general error events
  window.addEventListener('toast:error', (event: any) => {
    toast.error(event.detail.message);
  });

  // Listen for success events
  window.addEventListener('toast:success', (event: any) => {
    toast.success(event.detail.message);
  });

  // Register pages for credentials, scheduled scans (scan-profiles uses direct HTML include)
  window.rackdRegisterPage('/credentials', credentialsPageTemplate);
  window.rackdRegisterPage('/scheduled-scans', scheduledScansPageTemplate);

  // Register scan profiles component (page uses direct HTML include)
  Alpine.data('scanProfilesList', scanProfilesList);
  // Conflicts component
  Alpine.data('conflictList', conflictList);
  Alpine.data('webhookComponent', webhookComponent);
  Alpine.data('customFieldComponent', customFieldComponent);
  Alpine.data('dashboardComponent', dashboardComponent);
  Alpine.data('circuitComponent', circuitComponent);
  Alpine.data('natComponent', natComponent);
  Alpine.data('dnsProvidersComponent', dnsProvidersComponent);
  Alpine.data('dnsZonesComponent', dnsZonesComponent);
  Alpine.data('dnsRecordsComponent', dnsRecordsComponent);

  // Register deep scan type
  window.rackdRegisterScanType({
    value: 'deep',
    label: 'Deep',
    description: 'Comprehensive scan with SNMP/SSH',
  });

  // Expose Alpine globally
  window.Alpine = Alpine;

  // Register plugins
  Alpine.plugin(focus);
  Alpine.plugin(collapse);

  // Start Alpine
  Alpine.start();
}

init();
