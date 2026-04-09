import { expect, test } from '@playwright/test';

import { login } from './auth';
import { createNetwork, rowByExactText, uniqueName } from './inventory-helpers';
import { createScanProfile } from './scan-profile-helpers';

test('@discovery scheduled scans support create, edit, toggle, and delete flows', async ({ page }) => {
  await login(page);

  const networkName = uniqueName('e2e-schedule-network');
  const profileName = uniqueName('e2e-scan-profile');
  const scheduleName = uniqueName('e2e-schedule');
  const updatedScheduleName = `${scheduleName}-updated`;

  await createNetwork(page, {
    name: networkName,
    subnet: '10.92.0.0/24',
  });
  await createScanProfile(page, {
    name: profileName,
    scanType: 'quick',
    description: 'Scheduled scan profile',
  });

  await page.goto('/scheduled-scans');
  await expect(page.getByRole('heading', { name: 'Scheduled Scans' })).toBeVisible();

  await page.getByRole('button', { name: 'Add new scheduled scan' }).click();
  const dialog = page.getByRole('dialog', { name: 'Add Schedule' });
  await expect(dialog).toBeVisible();
  await dialog.getByLabel(/^Name/).fill(scheduleName);
  await dialog.getByLabel(/^Network/).selectOption({ label: `${networkName} (10.92.0.0/24)` });
  await dialog.getByLabel(/Scan Profile/).selectOption({ label: profileName });
  await dialog.getByLabel(/Cron Expression/).fill('0 5 * * *');
  await dialog.getByLabel('Description').fill('Nightly schedule');
  await dialog.getByRole('button', { name: 'Save' }).click();
  await expect(dialog).toBeHidden();

  const row = rowByExactText(page, scheduleName);
  await expect(row).toBeVisible();
  await expect(row).toContainText(profileName);
  await expect(row).toContainText('Enabled');

  await row.getByRole('button', { name: `Edit ${scheduleName}` }).click();
  const editDialog = page.getByRole('dialog', { name: 'Edit Schedule' });
  await expect(editDialog).toBeVisible();
  await editDialog.getByLabel(/^Name/).fill(updatedScheduleName);
  await editDialog.getByLabel(/Cron Expression/).fill('0 6 * * *');
  await editDialog.getByRole('button', { name: 'Save' }).click();
  await expect(editDialog).toBeHidden();

  const updatedRow = rowByExactText(page, updatedScheduleName);
  await expect(updatedRow).toBeVisible();
  await updatedRow.getByRole('button', { name: `Disable schedule: ${updatedScheduleName}` }).click();
  await expect(updatedRow).toContainText('Disabled');

  await updatedRow.getByRole('button', { name: `Delete ${updatedScheduleName}` }).click();
  const deleteDialog = page.getByRole('alertdialog', { name: 'Delete Schedule' });
  await expect(deleteDialog).toBeVisible();
  await deleteDialog.getByRole('button', { name: 'Delete' }).click();
  await expect(updatedRow).toHaveCount(0);
});
