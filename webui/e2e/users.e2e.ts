import { expect, test } from '@playwright/test';

import { login } from './helpers';

test('create user modal closes with escape', async ({ page }) => {
  await login(page);
  await page.goto('/users');
  await expect(page.getByRole('heading', { name: 'Users' })).toBeVisible();

  await page.getByRole('button', { name: 'Add User' }).click();
  const dialog = page.getByRole('dialog', { name: 'Create User' });
  await expect(dialog).toBeVisible();

  await page.keyboard.press('Escape');
  await expect(dialog).toBeHidden();
});

test('creates a user from the users page modal', async ({ page }) => {
  await login(page);
  await page.goto('/users');
  await expect(page.getByRole('heading', { name: 'Users' })).toBeVisible();

  const username = `e2e-user-${Date.now()}`;
  const email = `${username}@test.local`;
  await page.getByRole('button', { name: 'Add User' }).click();
  const dialog = page.getByRole('dialog', { name: 'Create User' });
  await expect(dialog).toBeVisible();

  await dialog.getByLabel(/^Username/).fill(username);
  await dialog.getByLabel(/^Email/).fill(email);
  await dialog.getByLabel('Full Name').fill('E2E User');
  await dialog.getByLabel(/^Password/).fill('e2e-user-pass');
  await dialog.getByRole('button', { name: 'Create User' }).click();

  await expect(dialog).toBeHidden();
  const row = page.locator('tbody tr').filter({ has: page.getByText(username, { exact: true }) });
  await expect(row).toBeVisible();
  await expect(row).toContainText(email);
  await expect(row).toContainText('E2E User');
});
