import { expect, test } from '@playwright/test';

import { login, logout, openUserMenu } from './auth';
import { uniqueName } from './inventory-helpers';
import { createUser } from './users-helpers';

test('@auth @smoke shows an error for invalid credentials', async ({ page }) => {
  await page.goto('/login');
  await page.getByLabel('Username').fill('admin');
  await page.getByLabel('Password').fill('wrong-password');
  await page.getByRole('button', { name: 'Sign in' }).click();

  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole('alert')).toContainText(/invalid|failed/i);
});

test('@auth can sign out from the user menu', async ({ page }) => {
  await login(page);
  await logout(page);
  await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible();
});

test('@auth @users user can change their password and sign in with the new password', async ({ page }) => {
  await login(page);

  const username = uniqueName('e2e-password-user');
  const originalPassword = 'original-pass-123';
  const newPassword = 'updated-pass-123';

  await createUser(page, {
    username,
    email: `${username}@test.local`,
    fullName: 'Password User',
    password: originalPassword,
  });

  await logout(page);
  await login(page, username, originalPassword);

  await openUserMenu(page);
  await page.getByRole('menuitem', { name: 'Change Password' }).click();
  const dialog = page.getByRole('dialog', { name: 'Change Password' });
  await expect(dialog).toBeVisible();

  await dialog.getByLabel(/^Current Password/).fill(originalPassword);
  await dialog.getByLabel(/^New Password/).fill(newPassword);
  await dialog.getByLabel(/^Confirm New Password/).fill(newPassword);
  await dialog.getByRole('button', { name: 'Change Password' }).click();
  await expect(dialog).toBeHidden();

  await logout(page);

  await page.goto('/login');
  await page.getByLabel('Username').fill(username);
  await page.getByLabel('Password').fill(originalPassword);
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(page.getByRole('alert')).toContainText(/invalid|failed/i);

  await login(page, username, newPassword);
});
