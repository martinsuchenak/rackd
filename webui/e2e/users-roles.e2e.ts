import { expect, test } from '@playwright/test';

import { login, logout, openUserMenu } from './auth';
import { uniqueName } from './inventory-helpers';
import { createRole } from './roles-helpers';
import { createUser, grantUserRole } from './users-helpers';

test('@users user menu supports profile edit flow', async ({ page }) => {
  await login(page);

  await openUserMenu(page);
  await page.getByRole('menuitem', { name: 'Edit Profile' }).click();
  const dialog = page.getByRole('dialog', { name: 'Edit Profile' });
  await expect(dialog).toBeVisible();

  const updatedFullName = `Admin ${Date.now()}`;
  await dialog.getByLabel('Full Name').fill(updatedFullName);
  await dialog.getByRole('button', { name: 'Save Changes' }).click();
  await expect(dialog).toBeHidden();

  await openUserMenu(page);
  await page.getByRole('menuitem', { name: 'Edit Profile' }).click();
  const reopenedDialog = page.getByRole('dialog', { name: 'Edit Profile' });
  await expect(reopenedDialog).toBeVisible();
  await expect(reopenedDialog.getByLabel('Full Name')).toHaveValue(updatedFullName);
});

test('@users admin can reset another users password', async ({ page }) => {
  await login(page);

  const username = uniqueName('e2e-reset-user');
  const originalPassword = 'reset-old-pass';
  const resetPassword = 'reset-new-pass';

  await createUser(page, {
    username,
    email: `${username}@test.local`,
    fullName: 'Reset User',
    password: originalPassword,
  });

  const row = page.locator('tbody tr').filter({ has: page.getByText(username, { exact: true }) });
  await row.getByRole('button', { name: `Reset password for ${username}` }).click();

  const dialog = page.getByRole('dialog', { name: 'Reset Password' });
  await expect(dialog).toBeVisible();
  await dialog.getByLabel(/^New Password/).fill(resetPassword);
  await dialog.getByLabel(/^Confirm New Password/).fill(resetPassword);
  await dialog.getByRole('button', { name: 'Reset Password' }).click();
  await expect(dialog).toBeHidden();

  await logout(page);
  await login(page, username, resetPassword);
});

test('@rbac custom role permissions take effect after assignment', async ({ page }) => {
  await login(page);

  const roleName = uniqueName('e2e-devices-reader');
  const username = uniqueName('e2e-role-user');
  const password = 'role-user-pass';

  await createRole(page, {
    name: roleName,
    description: 'Device read-only role',
    permissions: ['device:list'],
  });

  await createUser(page, {
    username,
    email: `${username}@test.local`,
    fullName: 'Role User',
    password,
  });
  await grantUserRole(page, username, roleName);

  await logout(page);
  await login(page, username, password);

  await page.goto('/devices');
  await expect(page.getByRole('heading', { name: 'Devices' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Add Device' })).not.toBeVisible();

  await page.goto('/dns/providers');
  await expect(page.getByText("You don't have permission to access this page.")).toBeVisible();
});
