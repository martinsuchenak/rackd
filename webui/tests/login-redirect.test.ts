import { describe, expect, test } from 'bun:test';

import { loginRedirectTarget } from '../src/components/login';

const origin = 'https://rackd.example';

describe('loginRedirectTarget', () => {
  test('keeps same-origin paths', () => {
    expect(loginRedirectTarget('/devices?id=1#x', origin)).toBe('https://rackd.example/devices?id=1#x');
    expect(loginRedirectTarget(null, origin)).toBe('/');
  });

  test('never leaves the origin', () => {
    for (const redirect of [
      'https://evil.example/',
      '//evil.example',
      '/\\\\evil.example',
      'javascript:alert(1)',
      // Same origin, but the path alone would be protocol-relative.
      '/.//evil.example/login',
      '/a/..//evil.example',
      'https://rackd.example//evil.example',
    ]) {
      const target = loginRedirectTarget(redirect, origin);
      expect(target === '/' || new URL(target).origin === origin).toBe(true);
      expect(new URL(target, origin).origin).toBe(origin);
    }
  });
});
