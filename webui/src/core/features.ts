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
    /** The menu section; none puts it above the sections (Dashboard). */
    section?: NavSectionId;
    order: number;
    badgeKey?: FeatureBadgeKey;
  };
  permission?: PermissionRequirement;
  /** A server setting that must be on for the page to exist. */
  requires?: 'mcp_oauth';
}

/** Core's menu sections, in menu order. */
export const navSections = [
  { id: 'organization', label: 'Organization' },
  { id: 'inventory', label: 'Inventory' },
  { id: 'ipam', label: 'IPAM' },
  { id: 'dns', label: 'DNS' },
  { id: 'discovery', label: 'Discovery' },
  { id: 'integrations', label: 'Integrations' },
  { id: 'administration', label: 'Administration' },
] as const;

export type NavSectionId = (typeof navSections)[number]['id'];

export interface NavSection {
  /** '' for the items above the sections, such as Dashboard. */
  id: string;
  label: string;
  items: FeatureNavItem[];
}

const EXTENSIONS_SECTION = 'Extensions';

/**
 * Groups menu items into sections, in menu order, each sorted by order and
 * without empty sections. An extension item with a core section's id joins
 * it; one with another section name gets a section of that name, and one
 * with none goes in Extensions. Extension sections come before
 * Administration, which stays last.
 */
export function groupNavItems(items: FeatureNavItem[]): NavSection[] {
  const core = new Map<string, string>(navSections.map((s) => [s.id, s.label]));
  const top: NavSection = { id: '', label: '', items: [] };
  const sections = new Map<string, NavSection>(navSections.map((s) => [s.id, { id: s.id, label: s.label, items: [] }]));
  const extra = new Map<string, NavSection>();
  for (const item of items) {
    const section = item.section ?? '';
    if (section === '' && item.extension !== true) {
      top.items.push(item);
    } else if (core.has(section)) {
      sections.get(section)!.items.push(item);
    } else {
      const label = section || EXTENSIONS_SECTION;
      const id = 'ext:' + label;
      if (!extra.has(id)) {
        extra.set(id, { id, label, items: [] });
      }
      extra.get(id)!.items.push(item);
    }
  }
  const admin = sections.get('administration')!;
  sections.delete('administration');
  const ordered = [top, ...sections.values(), ...extra.values(), admin];
  for (const s of ordered) {
    s.items.sort((a, b) => a.order - b.order);
  }
  return ordered.filter((s) => s.items.length > 0);
}

/** The id of the section whose item is the current route, or ''. */
export function activeSectionId(sections: NavSection[], route: string): string {
  for (const s of sections) {
    if (s.items.some((item) => isActiveNavItem(item, route))) {
      return s.id;
    }
  }
  return '';
}

/** Whether a menu item is the current page, or a page under it. */
export function isActiveNavItem(item: NavItem, route: string): boolean {
  const path = route.split('?')[0];
  if (item.path === '/') {
    return path === '/';
  }
  return path === item.path || path.startsWith(item.path + '/') || (item.path === '/networks' && path.startsWith('/pools'));
}

export interface FeatureNavItem extends NavItem {
  badgeKey?: FeatureBadgeKey;
}

const featureDefinitions: FeatureDefinition[] = [
  { path: '/', title: 'Dashboard', nav: { label: 'Dashboard', icon: 'home', order: 0 } },
  { path: '/devices', title: 'Devices', routePrefix: '/devices', nav: { label: 'Devices', icon: 'server', section: 'inventory', order: 10 }, permission: { resource: 'devices', action: 'list' } },
  { path: '/devices/detail', title: 'Device Details', permission: { resource: 'devices', action: 'list' } },
  { path: '/devices/graph', title: 'Device Relationships Graph', permission: { resource: 'devices', action: 'list' } },
  { path: '/networks', title: 'Networks', routePrefix: '/networks', nav: { label: 'Networks', icon: 'network', section: 'ipam', order: 10 }, permission: { resource: 'networks', action: 'list' } },
  { path: '/networks/detail', title: 'Network Details', permission: { resource: 'networks', action: 'list' } },
  { path: '/pools/detail', title: 'Pool Details', routePrefix: '/pools', permission: { resource: 'pools', action: 'list' } },
  { path: '/datacenters', title: 'Datacenters', routePrefix: '/datacenters', nav: { label: 'Datacenters', icon: 'building', section: 'organization', order: 10 }, permission: { resource: 'datacenters', action: 'list' } },
  { path: '/datacenters/detail', title: 'Datacenter Details', permission: { resource: 'datacenters', action: 'list' } },
  { path: '/discovery', title: 'Discovery', routePrefix: '/discovery', nav: { label: 'Scans', icon: 'search', section: 'discovery', order: 10 }, permission: { resource: 'discovery', action: 'list' } },
  { path: '/credentials', title: 'Credentials', nav: { label: 'Credentials', icon: 'key', section: 'discovery', order: 40 }, permission: { resource: 'credentials', action: 'list' } },
  { path: '/scan-profiles', title: 'Scan Profiles', nav: { label: 'Scan Profiles', icon: 'settings', section: 'discovery', order: 30 }, permission: { resource: 'scan-profiles', action: 'list' } },
  { path: '/scheduled-scans', title: 'Scheduled Scans', nav: { label: 'Scheduled Scans', icon: 'clock', section: 'discovery', order: 20 }, permission: { resource: 'scheduled-scans', action: 'list' } },
  { path: '/conflicts', title: 'IP Conflicts', nav: { label: 'Conflicts', icon: 'warning', section: 'ipam', order: 30, badgeKey: 'conflicts' }, permission: { resource: 'conflicts', action: 'list' } },
  { path: '/webhooks', title: 'Webhooks', nav: { label: 'Webhooks', icon: 'zap', section: 'integrations', order: 10 }, permission: { resource: 'webhooks', action: 'list' } },
  { path: '/custom-fields', title: 'Custom Fields', nav: { label: 'Custom Fields', icon: 'tag', section: 'integrations', order: 20 }, permission: { resource: 'custom-fields', action: 'list' } },
  { path: '/circuits', title: 'Circuits', nav: { label: 'Circuits', icon: 'shuffle', section: 'inventory', order: 20 }, permission: { resource: 'circuits', action: 'list' } },
  { path: '/nat', title: 'NAT Mappings', nav: { label: 'NAT', icon: 'git-branch', section: 'ipam', order: 20 }, permission: { resource: 'nat', action: 'list' } },
  { path: '/dns/providers', title: 'DNS Providers', routePrefix: '/dns/providers', nav: { label: 'Providers', icon: 'server', section: 'dns', order: 20 }, permission: { resource: 'dns-provider', action: 'list' } },
  { path: '/dns/zones', title: 'DNS Zones', routePrefix: '/dns/zones', nav: { label: 'Zones', icon: 'globe', section: 'dns', order: 10 }, permission: { resource: 'dns-zone', action: 'list' } },
  { path: '/dns/records', title: 'DNS Records', routePrefix: '/dns/records', permission: { resource: 'dns', action: 'list' } },
  { path: '/users', title: 'User Management', nav: { label: 'Users', icon: 'user', section: 'administration', order: 10 }, permission: { resource: 'users', action: 'list' } },
  { path: '/roles', title: 'Role Management', nav: { label: 'Roles', icon: 'shield', section: 'administration', order: 20 }, permission: { resource: 'roles', action: 'list' } },
  { path: '/oauth-clients', title: 'OAuth Clients', nav: { label: 'OAuth Clients', icon: 'shield', section: 'integrations', order: 40 }, permission: { resource: 'users', action: 'list' }, requires: 'mcp_oauth' },
  { path: '/api-keys', title: 'API Keys', nav: { label: 'API Keys', icon: 'key', section: 'integrations', order: 30 }, permission: { resource: 'apikeys', action: 'list' } },
  { path: '/audit', title: 'Audit Logs', nav: { label: 'Audit', icon: 'shield', section: 'administration', order: 30 }, permission: { resource: 'audit', action: 'list' } },
  { path: '/logs', title: 'Application Logs', nav: { label: 'Logs', icon: 'terminal', section: 'administration', order: 40 }, permission: { resource: 'logs', action: 'list' } },
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
      section: feature.nav!.section,
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
    // By path: core owns its routes. Labels may repeat across sections.
    (item) => !coreItems.some((base) => base.path === item.path || (base.path === '/' && item.path === ''))
  );
  return [...baseItems, ...dynamic.map((item) => ({ ...item, extension: true }))].sort((a, b) => a.order - b.order);
}
