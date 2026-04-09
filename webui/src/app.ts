// Rackd Web UI - Main Application Entry Point

import Alpine from '@alpinejs/csp';
import focus from '@alpinejs/focus';
import collapse from '@alpinejs/collapse';
import type { UIConfig, Permission, Role } from './core/types';
import { api, RackdAPI } from './core/api';
import { getClosestDataStack, getPermissionsStore, getToastStore, mutateDom, type PermissionsStore } from './core/alpine';
import { canAccessRoute, getPageTitle, mergeNavItems } from './core/features';

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
import { apiKeysList } from './components/api-keys';

function parseModelPath(expression: string): string[] | null {
  const trimmed = expression.trim();
  if (!trimmed || /[()\[\]]/.test(trimmed)) return null;
  const parts = trimmed.split('.').map(part => part.trim()).filter(Boolean);
  return parts.length > 0 ? parts : null;
}

function getScopeEntry(el: HTMLElement, rootKey: string): Record<string, unknown> | null {
  const stack = getClosestDataStack(el);
  if (!stack) return null;
  for (const scope of stack) {
    if (scope && Object.prototype.hasOwnProperty.call(scope, rootKey)) {
      return scope;
    }
  }
  return null;
}

function setModelValue(el: HTMLElement, expression: string, value: unknown): void {
  const path = parseModelPath(expression);
  if (!path) return;

  const [rootKey, ...rest] = path;
  const scope = getScopeEntry(el, rootKey);
  if (!scope) return;

  if (rest.length === 0) {
    scope[rootKey] = value;
    return;
  }

  let target = scope[rootKey] as Record<string, unknown> | undefined;
  if (!target || typeof target !== 'object') return;

  for (let i = 0; i < rest.length - 1; i++) {
    const next = target[rest[i]];
    if (!next || typeof next !== 'object') return;
    target = next as Record<string, unknown>;
  }

  target[rest[rest.length - 1]] = value;
}

function getInputValue(el: HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement, currentValue: unknown, modifiers: string[]): unknown {
  if (el instanceof HTMLInputElement && el.type === 'checkbox') {
    if (Array.isArray(currentValue)) {
      const next = [...currentValue];
      const idx = next.findIndex(v => String(v) === el.value);
      if (el.checked && idx === -1) next.push(el.value);
      if (!el.checked && idx !== -1) next.splice(idx, 1);
      return next;
    }
    return el.checked;
  }

  if (el instanceof HTMLInputElement && el.type === 'radio') {
    return el.value;
  }

  if (el instanceof HTMLSelectElement && el.multiple) {
    return Array.from(el.selectedOptions).map(option => option.value);
  }

  let value: unknown = el.value;
  if (modifiers.includes('number')) {
    value = value === '' ? '' : Number(value);
  }
  if (modifiers.includes('trim') && typeof value === 'string') {
    value = value.trim();
  }
  return value;
}

function syncModelValue(el: HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement, value: unknown): void {
  mutateDom(() => {
    if (el instanceof HTMLInputElement && el.type === 'checkbox') {
      el.checked = Array.isArray(value) ? value.map(String).includes(el.value) : !!value;
      return;
    }

    if (el instanceof HTMLInputElement && el.type === 'radio') {
      el.checked = String(value) === el.value;
      return;
    }

    if (el instanceof HTMLSelectElement && el.multiple && Array.isArray(value)) {
      const selected = value.map(String);
      Array.from(el.options).forEach(option => {
        option.selected = selected.includes(option.value);
      });
      return;
    }

    const normalized = value == null ? '' : String(value);
    if (el.value !== normalized) {
      el.value = normalized;
    }
  });
}

function registerCspSafeModelDirective(): void {
  Alpine.directive('model', (
    el: Element,
    { expression, modifiers }: { expression: string; modifiers: string[] },
    {
      effect,
      cleanup,
      evaluateLater,
    }: {
      effect: (callback: () => void) => void;
      cleanup: (callback: () => void) => void;
      evaluateLater: (expression: string) => (callback: (value: unknown) => void) => void;
    }
  ) => {
    const target = el as HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement;
    const evaluateGet = evaluateLater(expression);

    const eventName =
      target instanceof HTMLSelectElement ||
      (target instanceof HTMLInputElement && ['checkbox', 'radio'].includes(target.type))
        ? 'change'
        : 'input';

    const listener = () => {
      evaluateGet((currentValue: unknown) => {
        setModelValue(target, expression, getInputValue(target, currentValue, modifiers));
      });
    };

    target.addEventListener(eventName, listener);
    cleanup(() => target.removeEventListener(eventName, listener));

    effect(() => {
      evaluateGet((value: unknown) => {
        syncModelValue(target, value);
      });
    });
  });
}

// Update page title based on route
function updatePageTitle(route: string) {
  document.title = `${getPageTitle(route)} - Rackd`;
}

declare global {
  interface Window {
    Alpine: typeof Alpine;
    rackdAPI: RackdAPI;
    rackdConfig: UIConfig | null;
  }
}

function checkRoutePermission(path: string): boolean {
  const userPermissions = window.rackdConfig?.user?.permissions ?? [];
  return canAccessRoute(path, userPermissions);
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
      const allItems = mergeNavItems(window.rackdConfig?.nav_items ?? []);
      const userPermissions = window.rackdConfig?.user?.permissions ?? [];
      return allItems.filter((item) => {
        if (!item.required_permissions || item.required_permissions.length === 0) {
          return true;
        }
        return item.required_permissions.every((req) =>
          userPermissions.some((perm) =>
            perm.resource === req.resource && perm.action === req.action
          )
        );
      }).map((item) => ({
        ...item,
        badgeValue: item.badgeKey === 'conflicts' ? this.activeConflictCount : 0,
      }));
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

    closeSidebar() {
      this.sidebarOpen = false;
    },

    toggleSidebar() {
      this.sidebarOpen = !this.sidebarOpen;
    },

    hasBadge(item: any): boolean {
      return item.badgeValue > 0;
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
  const userPermissions: Permission[] = window.rackdConfig?.user?.permissions ?? [];
  const userRoles: Role[] = window.rackdConfig?.user?.roles ?? [];

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
        const permissionsStore = getPermissionsStore();
        if (permissionsStore) {
          const userPermissions: Permission[] = config.user?.permissions ?? [];
          const userRoles: Role[] = config.user?.roles ?? [];
          permissionsStore.permissions = userPermissions;
          permissionsStore.roles = userRoles;
          getToastStore()?.success('Permissions refreshed successfully');
        }
      } catch (error) {
        console.error('Failed to refresh permissions:', error);
        getToastStore()?.error('Failed to refresh permissions. Please reload the page.');
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
  Alpine.data('apiKeysList', apiKeysList);

  // Expose Alpine globally
  window.Alpine = Alpine;

  // Register plugins
  Alpine.plugin(focus);
  Alpine.plugin(collapse);
  registerCspSafeModelDirective();

  // Start Alpine
  Alpine.start();
}

init();
