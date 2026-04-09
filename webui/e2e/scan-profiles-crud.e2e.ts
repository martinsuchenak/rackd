import { expect, test } from '@playwright/test';

import { login } from './auth';
import { rowByExactText, uniqueName } from './inventory-helpers';
import { createScanProfile } from './scan-profile-helpers';

test('@discovery scan profiles support direct create and edit flows', async ({ page }) => {
  await login(page);

  const profileName = uniqueName('e2e-profile');
  const updatedName = `${profileName}-updated`;

  await createScanProfile(page, {
    name: profileName,
    scanType: 'custom',
    ports: '22,80,443',
    enableSSH: true,
    description: 'Initial scan profile',
  });

  const row = rowByExactText(page, profileName);
  await expect(row).toBeVisible();
  await expect(row).toContainText('CUSTOM');

  await row.getByRole('button', { name: `Edit ${profileName}` }).click();
  const editDialog = page.getByRole('dialog', { name: 'Edit Scan Profile' });
  await expect(editDialog).toBeVisible();
  await editDialog.locator('input[type="text"]').first().fill(updatedName);
  await editDialog.locator('textarea').fill('Updated scan profile');
  await editDialog.getByRole('button', { name: 'Update Profile' }).click();
  await expect(editDialog).toBeHidden();

  const updatedRow = rowByExactText(page, updatedName);
  await expect(updatedRow).toBeVisible();
  await expect(updatedRow).toContainText('Updated scan profile');
});
