import Alpine from '@alpinejs/csp';
import type { Permission, Role } from './types';

export interface PermissionsStore {
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

export interface ToastStore {
  success: (msg: string) => void;
  error: (msg: string) => void;
  info: (msg: string) => void;
  warning: (msg: string) => void;
}

type AlpineWatchable = Record<string, unknown> & {
  $watch?: (property: string, callback: (value: unknown) => void) => void;
};

interface AlpineDispatchable {
  $dispatch?: (event: string, detail?: unknown) => void;
}

interface AlpineInternals {
  closestDataStack?: (el: HTMLElement) => Array<Record<string, unknown>>;
  mutateDom?: (callback: () => void) => void;
}

export function getPermissionsStore(): PermissionsStore | undefined {
  return Alpine.store('permissions') as PermissionsStore | undefined;
}

export function getToastStore(): ToastStore | undefined {
  return Alpine.store('toast') as ToastStore | undefined;
}

export function dispatchNav(target: AlpineDispatchable | null | undefined, path: string): void {
  if (target?.$dispatch) {
    target.$dispatch('nav', path);
    return;
  }
  window.dispatchEvent(new CustomEvent('nav', { detail: path }));
}

export function watchAlpineProperty(
  target: AlpineWatchable | null | undefined,
  property: string,
  callback: (value: unknown) => void
): void {
  target?.$watch?.(property, callback);
}

export function getClosestDataStack(el: HTMLElement): Array<Record<string, unknown>> | undefined {
  return (Alpine as unknown as AlpineInternals).closestDataStack?.(el);
}

export function mutateDom(callback: () => void): void {
  const internals = Alpine as unknown as AlpineInternals;
  if (internals.mutateDom) {
    internals.mutateDom(callback);
    return;
  }
  callback();
}
