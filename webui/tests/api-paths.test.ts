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
