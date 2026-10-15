import { afterEach, describe, expect, mock, test } from 'bun:test';

import { RackdAPI, hasDotSegment } from '../src/core/api';

const originalFetch = globalThis.fetch;
const originalWindow = globalThis.window;

afterEach(() => {
  globalThis.fetch = originalFetch;
  (globalThis as { window?: Window }).window = originalWindow;
});

// Records the URL of each request instead of sending it.
function recordFetches(): string[] {
  const urls: string[] = [];
  (globalThis as { window?: unknown }).window = { location: { pathname: '/', href: '/' }, dispatchEvent: () => true };
  globalThis.fetch = mock(async (url: string | URL | Request) => {
    urls.push(String(url));
    return new Response(null, { status: 204 });
  }) as unknown as typeof fetch;
  return urls;
}

describe('API paths', () => {
  test('ids are escaped, so they stay in their own path segment', async () => {
    const urls = recordFetches();
    const api = new RackdAPI();
    await api.deleteDevice('abc?force=1');
    await api.deleteDevice('a/b');
    await api.updateDevice('x#frag', {});
    expect(urls).toEqual(['/api/v1/devices/abc%3Fforce%3D1', '/api/v1/devices/a%2Fb', '/api/v1/devices/x%23frag']);
  });

  test('a request that would leave its route is refused, not sent', async () => {
    const urls = recordFetches();
    const api = new RackdAPI();
    // encodeURIComponent leaves ".." alone, and the browser would resolve
    // /api/v1/devices/.. to /api/v1/.
    await expect(api.deleteDevice('..')).rejects.toThrow('path segment');
    expect(urls).toEqual([]);
  });

  test('hasDotSegment', () => {
    expect(hasDotSegment('/api/v1/devices/..')).toBe(true);
    expect(hasDotSegment('/api/v1/devices/%2e%2E/users')).toBe(true);
    expect(hasDotSegment('/api/v1/./devices')).toBe(true);
    expect(hasDotSegment('/api/v1/devices/a..b')).toBe(false);
    expect(hasDotSegment('/api/v1/devices?x=../y')).toBe(false);
  });
});

describe('listDevices', () => {
  test('fetches every page, not just the first', async () => {
    const urls: string[] = [];
    (globalThis as { window?: unknown }).window = { location: { pathname: '/', href: '/' }, dispatchEvent: () => true };
    globalThis.fetch = mock(async (url: string | URL | Request) => {
      urls.push(String(url));
      const offset = Number(new URL(String(url), 'http://x').searchParams.get('offset'));
      const count = offset === 0 ? 1000 : 5;
      const page = Array.from({ length: count }, (_, i) => ({ id: `d${offset + i}`, name: 'd' }));
      return new Response(JSON.stringify(page), { status: 200, headers: { 'Content-Type': 'application/json' } });
    }) as unknown as typeof fetch;

    const devices = await new RackdAPI().listDevices({ tags: ['a', 'b'] });
    expect(devices.length).toBe(1005);
    expect(urls.length).toBe(2);
    expect(urls[1]).toContain('offset=1000');
    expect(urls[0]).toContain('tags=a&tags=b');
  });
});
