// The web UI's extension API: what an extension's UI module can use, as
// window.rackd. This file has types only, so an extension can import it for
// type checking without bundling any of core.
//
// A module runs once, before the UI starts, and registers what it draws:
//
//   const rackd = window.rackd;
//   rackd.registerPage('/notes', (el, ctx) => { el.textContent = 'Notes'; });
//   rackd.registerSlot('device-detail', (el, ctx) => { ... });
//
// Changes that would break a module bump `version`.

/** Draws into el. It may return a function that undoes what it set up. */
export type Mount<C> = (el: HTMLElement, ctx: C) => void | (() => void);

/** What a page is drawn for. */
export interface PageContext {
  /** The route without its query, such as "/notes/42". */
  path: string;
  query: URLSearchParams;
}

/** What a slot is drawn for. */
export interface SlotContext {
  slot: string;
  /** The object the slot is about: the device on "device-detail", and so on. */
  data: unknown;
}

/**
 * The places in core's pages where extensions can add content, and what each
 * passes as SlotContext.data.
 */
export type SlotName =
  | 'dashboard' // end of the dashboard; no data
  | 'device-detail' // end of a device's page; the device
  | 'network-detail' // end of a network's page; the network
  | 'datacenter-detail'; // end of a datacenter's page; the datacenter

export interface RackdExtensionAPI {
  readonly version: 1;
  /**
   * Draws the page at path, which the extension's manifest declares in
   * UI.Pages. The page owns path and every route below it. Core checks the
   * page's permission before drawing it.
   */
  registerPage(path: string, mount: Mount<PageContext>): void;
  /** Adds content to one of core's slots, after any added before it. */
  registerSlot(slot: SlotName, mount: Mount<SlotContext>): void;
  /**
   * Registers an Alpine component for markup the extension draws. The UI uses
   * Alpine's CSP build, so expressions can only read and call the component's
   * properties. Prefix names with the extension's name: a name that is
   * already registered, by core or another extension, throws.
   */
  registerComponent(name: string, factory: (...args: never[]) => object): void;
  /** Whether the signed-in user has the permission. */
  can(resource: string, action: string): boolean;
  /** Opens a route in the UI without reloading the page. */
  navigate(path: string): void;
  /**
   * Calls Rackd's API as the signed-in user, sending and parsing JSON. path is
   * absolute, such as "/api/v1/devices". Errors reject with an Error whose
   * message is the API's.
   */
  request<T = unknown>(method: string, path: string, body?: unknown): Promise<T>;
  readonly toast: {
    success(message: string): void;
    error(message: string): void;
    info(message: string): void;
    warning(message: string): void;
  };
}

declare global {
  interface Window {
    rackd: RackdExtensionAPI;
  }
}
