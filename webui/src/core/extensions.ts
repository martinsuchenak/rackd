// Extension UI: loads extensions' modules and draws the pages and slot content
// they register through window.rackd (see extension-api.ts).

import type Alpine from '@alpinejs/csp';
import type { Mount, PageContext, RackdExtensionAPI, SlotContext, SlotName } from './extension-api';
import type { ExtensionInfo, PageInfo } from './types';

type AlpineInstance = typeof Alpine;

interface DirectiveData {
  value: string;
  expression: string;
}

interface DirectiveUtilities {
  effect: (callback: () => void) => void;
  cleanup: (callback: () => void) => void;
  evaluateLater: (expression: string) => (callback: (value: unknown) => void) => void;
}

type Directive = (el: Element, data: DirectiveData, utilities: DirectiveUtilities) => void;

const pages = new Map<string, Mount<PageContext>>();
const slots = new Map<string, Mount<SlotContext>[]>();

export function registerPage(path: string, mount: Mount<PageContext>): void {
  if (!path.startsWith('/')) {
    throw new Error(`rackd.registerPage: path "${path}" must start with /`);
  }
  if (pages.has(path)) {
    throw new Error(`rackd.registerPage: ${path} is already registered`);
  }
  pages.set(path, mount);
}

export function registerSlot(slot: SlotName, mount: Mount<SlotContext>): void {
  const list = slots.get(slot) ?? [];
  list.push(mount);
  slots.set(slot, list);
}

/** Forgets every registration; for tests. */
export function resetExtensionRegistry(): void {
  pages.clear();
  slots.clear();
}

function routePath(route: string): string {
  return route.split('?')[0];
}

/**
 * Returns the declared page that owns route: the one whose path is the route
 * or the longest one the route is below.
 */
export function findExtensionPage(route: string, declared: PageInfo[]): PageInfo | undefined {
  const path = routePath(route);
  let best: PageInfo | undefined;
  for (const page of declared) {
    const owns = path === page.path || path.startsWith(page.path.endsWith('/') ? page.path : page.path + '/');
    if (owns && (!best || page.path.length > best.path.length)) {
      best = page;
    }
  }
  return best;
}

function runCleanup(cleanup: unknown, what: string): void {
  if (typeof cleanup !== 'function') return;
  try {
    cleanup();
  } catch (error) {
    console.error(`Extension ${what} cleanup failed:`, error);
  }
}

function pageUnavailable(el: HTMLElement): void {
  const message = document.createElement('p');
  message.className = 'text-gray-600 dark:text-gray-400';
  message.textContent = 'This page could not be loaded.';
  el.replaceChildren(message);
}

/**
 * Registers x-ext-page="route", which draws the extension page that owns the
 * route, and x-ext-slot:<name>="data", which draws the content registered for
 * a slot. Mounts run outside Alpine's effect, so what they read is not
 * tracked, and receive raw objects rather than reactive proxies.
 */
export function registerExtensionDirectives(
  alpine: AlpineInstance,
  declaredPages: () => PageInfo[],
  isCoreRoute: (route: string) => boolean,
): void {
  const extPage: Directive = (el, { expression }, { effect, evaluateLater, cleanup }) => {
    const target = el as HTMLElement;
    const getRoute = evaluateLater(expression);
    let undo: unknown;
    let shown = '';
    let removed = false;

    const clear = () => {
      const pending = undo;
      undo = undefined;
      runCleanup(pending, 'page');
    };

    const draw = (route: string) => {
      if (removed || route === shown) return;
      shown = route;
      clear();
      target.replaceChildren();

      // Core's routes take precedence, as they do for titles and permissions.
      const page = isCoreRoute(route) ? undefined : findExtensionPage(route, declaredPages());
      if (!page) return;
      const mount = pages.get(page.path);
      if (!mount) {
        pageUnavailable(target);
        return;
      }
      const [path, query = ''] = route.split('?');
      try {
        undo = mount(target, { path, query: new URLSearchParams(query) });
      } catch (error) {
        console.error(`Extension page ${page.path} failed:`, error);
        pageUnavailable(target);
      }
    };

    effect(() => {
      getRoute((route) => {
        const value = String(route ?? '');
        queueMicrotask(() => draw(value));
      });
    });
    cleanup(() => {
      removed = true;
      clear();
    });
  };

  const extSlot: Directive = (el, { value, expression }, { effect, evaluateLater, cleanup }) => {
    const target = el as HTMLElement;
    const slot = value;
    let undos: unknown[] = [];
    let removed = false;

    const clear = () => {
      const pending = undos;
      undos = [];
      pending.forEach((undo) => runCleanup(undo, `slot ${slot}`));
    };

    const draw = (data: unknown) => {
      if (removed) return;
      clear();
      target.replaceChildren();
      // A slot about an object stays empty until the page has loaded it.
      if (expression && data == null) return;
      for (const mount of slots.get(slot) ?? []) {
        const child = document.createElement('div');
        target.append(child);
        try {
          undos.push(mount(child, { slot, data }));
        } catch (error) {
          console.error(`Extension slot ${slot} failed:`, error);
          child.remove();
        }
      }
    };

    if (!slots.has(slot)) return;
    if (!expression) {
      queueMicrotask(() => draw(undefined));
    } else {
      const getData = evaluateLater(expression);
      effect(() => {
        getData((data) => {
          const raw = data && typeof data === 'object' ? alpine.raw(data) : data;
          queueMicrotask(() => draw(raw));
        });
      });
    }
    cleanup(() => {
      removed = true;
      clear();
    });
  };

  alpine.directive('ext-page', extPage);
  alpine.directive('ext-slot', extSlot);
}

// How long the UI waits for a module before starting without it.
const moduleTimeoutMs = 10_000;

function importWithTimeout(src: string): Promise<unknown> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  const timeout = new Promise<never>((_, reject) => {
    timer = setTimeout(() => reject(new Error(`timed out after ${moduleTimeoutMs / 1000}s`)), moduleTimeoutMs);
  });
  return Promise.race([import(src), timeout]).finally(() => clearTimeout(timer));
}

/**
 * Adds the extensions' stylesheets and imports their modules. It returns the
 * names of the extensions whose modules failed or took too long, so the UI can
 * say so; the rest of the UI works without them. A module that finishes after
 * the timeout still registers, and its pages work from the next navigation.
 */
export async function loadExtensionUI(extensions: ExtensionInfo[]): Promise<string[]> {
  for (const ext of extensions) {
    for (const href of ext.styles ?? []) {
      const link = document.createElement('link');
      link.rel = 'stylesheet';
      link.href = href;
      document.head.append(link);
    }
  }

  const loads = extensions.flatMap((ext) =>
    (ext.scripts ?? []).map(async (src) => {
      try {
        await importWithTimeout(src);
        return null;
      } catch (error) {
        console.error(`Extension ${ext.name}: ${src} failed to load:`, error);
        return ext.name;
      }
    })
  );
  const failed = (await Promise.all(loads)).filter((name): name is string => name !== null);
  return [...new Set(failed)];
}

export type ExtensionAPIDeps = Omit<RackdExtensionAPI, 'version' | 'registerPage' | 'registerSlot'>;

export function createExtensionAPI(deps: ExtensionAPIDeps): RackdExtensionAPI {
  return { version: 1, registerPage, registerSlot, ...deps };
}
