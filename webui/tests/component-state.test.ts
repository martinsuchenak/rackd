import { describe, expect, mock, test } from 'bun:test';

import { apiKeysList } from '../src/components/api-keys';
import { scanProfilesList } from '../src/components/scan-profiles';
import type { Role, ScanProfile } from '../src/core/types';

describe('component state regressions', () => {
  test('scanProfilesList deleteProfile opens the shared delete modal instead of using confirm', async () => {
    const component = scanProfilesList();
    const profile: ScanProfile = {
      id: 'profile-1',
      name: 'Daily',
      scan_type: 'quick',
      timeout_sec: 5,
      max_workers: 10,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    };

    component.profiles = [profile];
    await component.deleteProfile(profile.id);

    expect(component.showDeleteModal).toBe(true);
    expect(component.selectedProfile).toEqual(profile);
    expect(component.deleteModalName).toBe('Daily');
  });

  test('scanProfilesList save dispatches to create or update based on modalType', async () => {
    const component = scanProfilesList();
    const createSpy = mock(async () => {});
    const updateSpy = mock(async () => {});
    component.createProfile = createSpy;
    component.updateProfile = updateSpy;

    component.modalType = 'create';
    await component.save();
    expect(createSpy).toHaveBeenCalledTimes(1);
    expect(updateSpy).toHaveBeenCalledTimes(0);

    component.modalType = 'edit';
    await component.save();
    expect(updateSpy).toHaveBeenCalledTimes(1);
  });

  test('rolesList modal getters derive from modalType and deleteConfirmed uses the shared delete action', async () => {
    const { rolesList } = await import('../src/components/roles');
    const component = rolesList();
    const deleteSpy = mock(async () => {});
    component.doDeleteRole = deleteSpy;

    const role: Role = {
      id: 'role-1',
      name: 'operators',
      is_system: false,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    };

    component.openDeleteModal(role);
    expect(component.showDeleteModal).toBe(true);
    expect(component.selectedRole).toEqual(role);

    await component.deleteConfirmed();
    expect(deleteSpy).toHaveBeenCalledTimes(1);
  });

  test('apiKeysList modal state resets selected data on close', () => {
    const component = apiKeysList();
    component.newKeyValue = 'secret';
    component.validationErrors = { name: 'required' };
    component.openCreateModal();
    expect(component.showCreateModal).toBe(true);

    component.closeModal();
    expect(component.modalType).toBe('');
    expect(component.selectedKey).toBeNull();
    expect(component.newKeyValue).toBe('');
    expect(component.validationErrors).toEqual({});
  });
});
