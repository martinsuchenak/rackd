import { afterEach, beforeEach, describe, expect, mock, test } from 'bun:test';

import { RackdAPI, RackdAPIError } from '../src/core/api';

const api = new RackdAPI();

const originalFetch = globalThis.fetch;
const originalWindow = globalThis.window;

function setWindow(pathname = '/oauth-clients'): void {
  (globalThis as typeof globalThis & {
    window: Window & { dispatchEvent: (event: Event) => boolean };
  }).window = {
    location: { pathname, href: pathname } as Window['location'],
    dispatchEvent: mock(() => true),
  } as unknown as Window & { dispatchEvent: (event: Event) => boolean };
}

beforeEach(() => {
  setWindow();
});

afterEach(() => {
  globalThis.fetch = originalFetch;
  (globalThis as typeof globalThis & { window?: Window }).window = originalWindow;
  mock.restore();
});

describe('RackdAPI OAuth client handling', () => {
  test('listOAuthClients returns an empty list when the route responds with HTML', async () => {
    globalThis.fetch = mock(async () =>
      new Response('<html>oauth disabled</html>', {
        status: 200,
        headers: { 'content-type': 'text/html' },
      }),
    ) as typeof fetch;

    await expect(api.listOAuthClients()).resolves.toEqual([]);
  });

  test('listOAuthClients returns parsed JSON when the route is enabled', async () => {
    const payload = [{ client_id: 'client-1', client_name: 'Client 1', redirect_uris: [], grant_types: [], created_at: '2026-01-01T00:00:00Z' }];
    globalThis.fetch = mock(async () =>
      new Response(JSON.stringify(payload), {
        status: 200,
        headers: { 'content-type': 'application/json' },
      }),
    ) as typeof fetch;

    await expect(api.listOAuthClients()).resolves.toEqual(payload);
  });

  test('requestOptionalJSON preserves permission-denied behavior on 403 JSON responses', async () => {
    globalThis.fetch = mock(async () =>
      new Response(JSON.stringify({ code: 'FORBIDDEN', message: 'forbidden' }), {
        status: 403,
        headers: { 'content-type': 'application/json' },
      }),
    ) as typeof fetch;

    await expect(api.listOAuthClients()).rejects.toBeInstanceOf(RackdAPIError);
    expect((window.dispatchEvent as unknown as ReturnType<typeof mock>).mock.calls).toHaveLength(1);
  });
});
