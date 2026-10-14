import { afterEach, describe, expect, test } from 'bun:test';

import { findExtensionPage, registerPage, resetExtensionRegistry } from '../src/core/extensions';
import { canAccessRoute, getPageTitle } from '../src/core/features';
import type { PageInfo, Permission } from '../src/core/types';

const pages: PageInfo[] = [
  { path: '/notes', title: 'Notes', required_permission: { resource: 'notes', action: 'list' } },
  { path: '/notes/archive', title: 'Archived Notes' },
  { path: '/reports', title: 'Reports' },
];

const notesList: Permission = {
  id: 'perm-notes-list',
  name: 'notes.list',
  resource: 'notes',
  action: 'list',
  created_at: '2026-01-01T00:00:00Z',
};

afterEach(() => resetExtensionRegistry());

describe('core/extensions', () => {
  test('findExtensionPage picks the page that owns a route', () => {
    expect(findExtensionPage('/notes', pages)?.title).toBe('Notes');
    expect(findExtensionPage('/notes/42?tab=history', pages)?.title).toBe('Notes');
    expect(findExtensionPage('/notes/archive/7', pages)?.title).toBe('Archived Notes');
    // A shared prefix is not ownership.
    expect(findExtensionPage('/notesy', pages)).toBeUndefined();
    expect(findExtensionPage('/devices', pages)).toBeUndefined();
  });

  test('extension pages get their titles and permissions', () => {
    expect(getPageTitle('/notes/42', pages)).toBe('Notes');
    expect(canAccessRoute('/notes', [], pages)).toBe(false);
    expect(canAccessRoute('/notes', [notesList], pages)).toBe(true);
    expect(canAccessRoute('/reports', [], pages)).toBe(true);
  });

  test("core's routes win over an extension page with the same path", () => {
    const clash: PageInfo[] = [{ path: '/users', title: 'Not Users' }];
    expect(getPageTitle('/users', clash)).toBe('User Management');
    expect(canAccessRoute('/users', [], clash)).toBe(false);
  });

  test('registerPage rejects relative and duplicate paths', () => {
    registerPage('/notes', () => {});
    expect(() => registerPage('/notes', () => {})).toThrow('already registered');
    expect(() => registerPage('notes', () => {})).toThrow('must start with /');
  });
});
