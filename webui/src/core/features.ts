import type { NavItem, Permission } from './types';

export interface PermissionRequirement {
  resource: string;
  action: string;
}

type FeatureBadgeKey = 'conflicts';

interface FeatureDefinition {
  path: string;
  title: string;
  routePrefix?: string;
  nav?: {
    label: string;
    icon: string;
    order: number;
    badgeKey?: FeatureBadgeKey;
  };
  permission?: PermissionRequirement;
}

export interface FeatureNavItem extends NavItem {
  badgeKey?: FeatureBadgeKey;
}

const featureDefinitions: FeatureDefinition[] = [
  { path: '/', title: 'Dashboard', nav: { label: 'Dashboard', icon: 'home', order: 0 } },
  { path: '/devices', title: 'Devices', routePrefix: '/devices', nav: { label: 'Devices', icon: 'server', order: 10 }, permission: { resource: 'devices', action: 'list' } },
  { path: '/devices/detail', title: 'Device Details', permission: { resource: 'devices', action: 'list' } },
  { path: '/devices/graph', title: 'Device Relationships Graph', permission: { resource: 'devices', action: 'list' } },
  { path: '/networks', title: 'Networks', routePrefix: '/networks', nav: { label: 'Networks', icon: 'network', order: 20 }, permission: { resource: 'networks', action: 'list' } },
  { path: '/networks/detail', title: 'Network Details', permission: { resource: 'networks', action: 'list' } },
  { path: '/pools/detail', title: 'Pool Details', routePrefix: '/pools', permission: { resource: 'pools', action: 'list' } },
  { path: '/datacenters', title: 'Datacenters', routePrefix: '/datacenters', nav: { label: 'Datacenters', icon: 'building', order: 30 }, permission: { resource: 'datacenters', action: 'list' } },
  { path: '/datacenters/detail', title: 'Datacenter Details', permission: { resource: 'datacenters', action: 'list' } },
  { path: '/discovery', title: 'Discovery', routePrefix: '/discovery', nav: { label: 'Discovery', icon: 'search', order: 40 }, permission: { resource: 'discovery', action: 'list' } },
  { path: '/credentials', title: 'Credentials', nav: { label: 'Credentials', icon: 'key', order: 42 }, permission: { resource: 'credentials', action: 'list' } },
  { path: '/scan-profiles', title: 'Scan Profiles', nav: { label: 'Scan Profiles', icon: 'settings', order: 44 }, permission: { resource: 'scan-profiles', action: 'list' } },
  { path: '/scheduled-scans', title: 'Scheduled Scans', nav: { label: 'Scheduled Scans', icon: 'clock', order: 46 }, permission: { resource: 'scheduled-scans', action: 'list' } },
  { path: '/conflicts', title: 'IP Conflicts', nav: { label: 'Conflicts', icon: 'warning', order: 50, badgeKey: 'conflicts' }, permission: { resource: 'conflicts', action: 'list' } },
  { path: '/webhooks', title: 'Webhooks', nav: { label: 'Webhooks', icon: 'zap', order: 51 }, permission: { resource: 'webhooks', action: 'list' } },
  { path: '/custom-fields', title: 'Custom Fields', nav: { label: 'Custom Fields', icon: 'tag', order: 52 }, permission: { resource: 'custom-fields', action: 'list' } },
  { path: '/circuits', title: 'Circuits', nav: { label: 'Circuits', icon: 'shuffle', order: 55 } },
  { path: '/nat', title: 'NAT Mappings', nav: { label: 'NAT', icon: 'git-branch', order: 56 } },
  { path: '/dns/providers', title: 'DNS Providers', routePrefix: '/dns/providers', nav: { label: 'DNS Providers', icon: 'server', order: 57 }, permission: { resource: 'dns-provider', action: 'list' } },
  { path: '/dns/zones', title: 'DNS Zones', routePrefix: '/dns/zones', nav: { label: 'DNS Zones', icon: 'globe', order: 58 }, permission: { resource: 'dns-zone', action: 'list' } },
  { path: '/dns/records', title: 'DNS Records', routePrefix: '/dns/records', permission: { resource: 'dns', action: 'list' } },
  { path: '/users', title: 'User Management', nav: { label: 'Users', icon: 'user', order: 90 }, permission: { resource: 'users', action: 'list' } },
  { path: '/roles', title: 'Role Management', nav: { label: 'Roles', icon: 'shield', order: 91 }, permission: { resource: 'roles', action: 'list' } },
  { path: '/oauth-clients', title: 'OAuth Clients', nav: { label: 'OAuth Clients', icon: 'shield', order: 93 }, permission: { resource: 'users', action: 'list' } },
  { path: '/api-keys', title: 'API Keys', nav: { label: 'API Keys', icon: 'key', order: 94 }, permission: { resource: 'apikeys', action: 'list' } },
];

function normalizePath(route: string): string {
  return route.split('?')[0];
}

function matchesFeaturePath(routePath: string, feature: FeatureDefinition): boolean {
  if (routePath === feature.path) {
    return true;
  }
  if (feature.routePrefix) {
    return routePath === feature.routePrefix || routePath.startsWith(feature.routePrefix + '/');
  }
  return false;
}

export function getFeatureNavItems(): FeatureNavItem[] {
  return featureDefinitions
    .filter((feature) => feature.nav)
    .map((feature) => ({
      label: feature.nav!.label,
      path: feature.path,
      icon: feature.nav!.icon,
      order: feature.nav!.order,
      badgeKey: feature.nav!.badgeKey,
      required_permissions: feature.permission ? [feature.permission] : undefined,
    }));
}

export function getPageTitle(route: string): string {
  const routePath = normalizePath(route);
  const feature = featureDefinitions.find((item) => item.path === routePath)
    ?? featureDefinitions.find((item) => matchesFeaturePath(routePath, item));
  return feature?.title || 'Page';
}

export function canAccessRoute(route: string, permissions: Permission[]): boolean {
  const routePath = normalizePath(route);
  const feature = featureDefinitions.find((item) => item.path === routePath)
    ?? featureDefinitions.find((item) => matchesFeaturePath(routePath, item));
  if (!feature?.permission) {
    return true;
  }
  return permissions.some((permission) =>
    permission.resource === feature.permission!.resource && permission.action === feature.permission!.action
  );
}

export function filterNavItems(items: NavItem[], permissions: Permission[]): NavItem[] {
  return items.filter((item) => {
    if (!item.required_permissions || item.required_permissions.length === 0) {
      return true;
    }
    return item.required_permissions.every((requirement) =>
      permissions.some((permission) =>
        permission.resource === requirement.resource && permission.action === requirement.action
      )
    );
  });
}

export function mergeNavItems(dynamicItems: NavItem[]): FeatureNavItem[] {
  const baseItems = getFeatureNavItems();
  const dynamic = dynamicItems.filter(
    (item) => !baseItems.some((base) =>
      (base.path === item.path || (base.path === '/' && item.path === '')) || base.label === item.label
    )
  );
  return [...baseItems, ...dynamic].sort((a, b) => a.order - b.order);
}
