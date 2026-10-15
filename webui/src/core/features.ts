import { findExtensionPage } from './extensions';
import type { NavItem, PageInfo, Permission, UIConfig } from './types';

/** The server settings a core page can depend on, from /api/v1/config. */
export type ServerSettings = Pick<UIConfig, 'mcp_oauth'> | null | undefined;

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
  /** A server setting that must be on for the page to exist. */
  requires?: 'mcp_oauth';
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
  { path: '/circuits', title: 'Circuits', nav: { label: 'Circuits', icon: 'shuffle', order: 55 }, permission: { resource: 'circuits', action: 'list' } },
  { path: '/nat', title: 'NAT Mappings', nav: { label: 'NAT', icon: 'git-branch', order: 56 }, permission: { resource: 'nat', action: 'list' } },
  { path: '/dns/providers', title: 'DNS Providers', routePrefix: '/dns/providers', nav: { label: 'DNS Providers', icon: 'server', order: 57 }, permission: { resource: 'dns-provider', action: 'list' } },
  { path: '/dns/zones', title: 'DNS Zones', routePrefix: '/dns/zones', nav: { label: 'DNS Zones', icon: 'globe', order: 58 }, permission: { resource: 'dns-zone', action: 'list' } },
  { path: '/dns/records', title: 'DNS Records', routePrefix: '/dns/records', permission: { resource: 'dns', action: 'list' } },
  { path: '/users', title: 'User Management', nav: { label: 'Users', icon: 'user', order: 90 }, permission: { resource: 'users', action: 'list' } },
  { path: '/roles', title: 'Role Management', nav: { label: 'Roles', icon: 'shield', order: 91 }, permission: { resource: 'roles', action: 'list' } },
  { path: '/oauth-clients', title: 'OAuth Clients', nav: { label: 'OAuth Clients', icon: 'shield', order: 93 }, permission: { resource: 'users', action: 'list' }, requires: 'mcp_oauth' },
  { path: '/api-keys', title: 'API Keys', nav: { label: 'API Keys', icon: 'key', order: 94 }, permission: { resource: 'apikeys', action: 'list' } },
  { path: '/audit', title: 'Audit Logs', nav: { label: 'Audit', icon: 'shield', order: 95 }, permission: { resource: 'audit', action: 'list' } },
  { path: '/logs', title: 'Application Logs', nav: { label: 'Logs', icon: 'terminal', order: 96 }, permission: { resource: 'logs', action: 'list' } },
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

function enabled(feature: FeatureDefinition, settings: ServerSettings): boolean {
  return !feature.requires || settings?.[feature.requires] === true;
}

/** Core's nav items; pass settings to leave out pages the server has off. */
export function getFeatureNavItems(settings?: ServerSettings, all = false): FeatureNavItem[] {
  return featureDefinitions
    .filter((feature) => feature.nav && (all || enabled(feature, settings)))
    .map((feature) => ({
      label: feature.nav!.label,
      path: feature.path,
      icon: feature.nav!.icon,
      order: feature.nav!.order,
      badgeKey: feature.nav!.badgeKey,
      required_permissions: feature.permission ? [feature.permission] : undefined,
    }));
}

function findCoreFeature(routePath: string): FeatureDefinition | undefined {
  return featureDefinitions.find((item) => item.path === routePath)
    ?? featureDefinitions.find((item) => matchesFeaturePath(routePath, item));
}

/** Whether one of core's pages owns the route. */
export function isCoreRoute(route: string): boolean {
  return findCoreFeature(normalizePath(route)) !== undefined;
}

// Core's routes come first, so an extension page cannot take one over.
export function getPageTitle(route: string, extensionPages: PageInfo[] = []): string {
  const routePath = normalizePath(route);
  const title = findCoreFeature(routePath)?.title ?? findExtensionPage(routePath, extensionPages)?.title;
  return title || 'Page';
}

/**
 * The server setting a core route needs that is off, such as 'mcp_oauth', or
 * null. The page doesn't exist on this server, which isn't the same as the
 * user lacking permission for it.
 */
export function missingSetting(route: string, settings: ServerSettings): string | null {
  const feature = findCoreFeature(normalizePath(route));
  return feature?.requires && !enabled(feature, settings) ? feature.requires : null;
}

export function canAccessRoute(route: string, permissions: Permission[], extensionPages: PageInfo[] = []): boolean {
  const routePath = normalizePath(route);
  const feature = findCoreFeature(routePath);
  const required = feature ? feature.permission : findExtensionPage(routePath, extensionPages)?.required_permission;
  if (!required) {
    return true;
  }
  return permissions.some((permission) =>
    permission.resource === required.resource && permission.action === required.action
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

export function mergeNavItems(dynamicItems: NavItem[], settings?: ServerSettings): FeatureNavItem[] {
  const baseItems = getFeatureNavItems(settings);
  // Dedupe against every core item, shown or not, so an extension can't
  // fill the slot of a core page the server has off.
  const coreItems = getFeatureNavItems(settings, true);
  const dynamic = dynamicItems.filter(
    (item) => !coreItems.some((base) =>
      (base.path === item.path || (base.path === '/' && item.path === '')) || base.label === item.label
    )
  );
  return [...baseItems, ...dynamic].sort((a, b) => a.order - b.order);
}
