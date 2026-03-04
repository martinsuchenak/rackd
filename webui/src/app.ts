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
import { credentialsList } from './components/credentials';
import { scanProfilesList } from './components/scan-profiles';
import { scheduledScansList } from './components/scheduled-scans';
import { login } from './components/login';
import { usersList } from './components/users';
import { rolesList } from './components/roles';
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
    '/credentials': 'Credentials',
    '/scan-profiles': 'Scan Profiles',
    '/scheduled-scans': 'Scheduled Scans',
    '/conflicts': 'IP Conflicts',
    '/webhooks': 'Webhooks',
    '/custom-fields': 'Custom Fields',
    '/circuits': 'Circuits',
    '/nat': 'NAT Mappings',
    '/dns/providers': 'DNS Providers',
    '/dns/zones': 'DNS Zones',
    '/dns/records': 'DNS Records',
    '/users': 'User Management',
    '/roles': 'Role Management',
  };
  const path = route.split('?')[0];
  document.title = `${titles[path] || 'Page'} - Rackd`;
}

declare global {
  interface Window {
    Alpine: typeof Alpine;
    rackdAPI: RackdAPI;
    rackdConfig: UIConfig | null;
  }
}

// Route permission requirements - maps route prefixes to required permissions
const routePermissions: { prefix: string; resource: string; action: string }[] = [
  { prefix: '/users', resource: 'users', action: 'list' },
  { prefix: '/roles', resource: 'roles', action: 'list' },
  { prefix: '/devices', resource: 'devices', action: 'list' },
  { prefix: '/networks', resource: 'networks', action: 'list' },
  { prefix: '/pools', resource: 'networks', action: 'list' },
  { prefix: '/datacenters', resource: 'datacenters', action: 'list' },
  { prefix: '/discovery', resource: 'discovery', action: 'list' },
  { prefix: '/credentials', resource: 'credentials', action: 'list' },
  { prefix: '/scan-profiles', resource: 'discovery', action: 'list' },
  { prefix: '/scheduled-scans', resource: 'discovery', action: 'list' },
  { prefix: '/webhooks', resource: 'webhooks', action: 'list' },
  { prefix: '/custom-fields', resource: 'custom-fields', action: 'list' },
  { prefix: '/dns', resource: 'dns', action: 'list' },
  { prefix: '/conflicts', resource: 'conflicts', action: 'list' },
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
        { label: 'Dashboard', path: '/', order: 0 },
        { label: 'Devices', path: '/devices', order: 10 },
        { label: 'Networks', path: '/networks', order: 20 },
        { label: 'Datacenters', path: '/datacenters', order: 30 },
        { label: 'Discovery', path: '/discovery', order: 40 },
        { label: 'Credentials', path: '/credentials', order: 42, required_permissions: [{ resource: 'credentials', action: 'list' }] },
        { label: 'Scan Profiles', path: '/scan-profiles', order: 44, required_permissions: [{ resource: 'discovery', action: 'list' }] },
        { label: 'Scheduled Scans', path: '/scheduled-scans', order: 46, required_permissions: [{ resource: 'discovery', action: 'list' }] },
        { label: 'Conflicts', path: '/conflicts', order: 50, badge: () => this.activeConflictCount },
        { label: 'Webhooks', path: '/webhooks', order: 51, required_permissions: [{ resource: 'webhooks', action: 'list' }] },
        { label: 'Custom Fields', path: '/custom-fields', order: 52, required_permissions: [{ resource: 'custom-fields', action: 'list' }] },
        { label: 'Circuits', path: '/circuits', order: 55 },
        { label: 'NAT', path: '/nat', order: 56 },
        { label: 'DNS Providers', path: '/dns/providers', order: 57, required_permissions: [{ resource: 'dns', action: 'list' }] },
        { label: 'DNS Zones', path: '/dns/zones', order: 58, required_permissions: [{ resource: 'dns', action: 'list' }] },
        { label: 'Users', path: '/users', order: 90, required_permissions: [{ resource: 'users', action: 'list' }] },
        { label: 'Roles', path: '/roles', order: 91, required_permissions: [{ resource: 'roles', action: 'list' }] },
      ];
      const dynamic = (window.rackdConfig?.nav_items ?? []).filter(
        (item: any) => !base.some((b) => (b.path === item.path || (b.path === '/' && item.path === '')) || b.label === item.label)
      );
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

interface PermissionsStore {
  permissions: Permission[];
  roles: Role[];
  loaded: boolean;
  can(resource: string, action: string): boolean;
  canList(resource: string): boolean;
  canRead(resource: string): boolean;
  canCreate(resource: string): boolean;
  canUpdate(resource: string): boolean;
  canDelete(resource: string): boolean;
  hasAnyPermission(resource: string, ...actions: string[]): boolean;
  hasAllPermissions(resource: string, ...actions: string[]): boolean;
}

interface ToastStore {
  success: (msg: string) => void;
  error: (msg: string) => void;
  info: (msg: string) => void;
  warning: (msg: string) => void;
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
  Alpine.data('scanProfilesList', scanProfilesList);
  Alpine.data('scheduledScansList', scheduledScansList);

  // Auth & user management
  Alpine.data('login', login);
  Alpine.data('usersList', usersList);
  Alpine.data('rolesList', rolesList);
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
        const permissionsStore = Alpine.store('permissions') as PermissionsStore;
        if (permissionsStore) {
          const userPermissions: Permission[] = (config.user?.permissions ?? []) as any;
          const userRoles: Role[] = (config.user?.roles ?? []) as any;
          permissionsStore.permissions = userPermissions;
          permissionsStore.roles = userRoles;
          (Alpine.store('toast') as ToastStore)?.success('Permissions refreshed successfully');
        }
      } catch (error) {
        console.error('Failed to refresh permissions:', error);
        (Alpine.store('toast') as ToastStore)?.error('Failed to refresh permissions. Please reload the page.');
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

  // Expose Alpine globally
  window.Alpine = Alpine;

  // Register plugins
  Alpine.plugin(focus);
  Alpine.plugin(collapse);

  // Start Alpine
  Alpine.start();
}

init();
