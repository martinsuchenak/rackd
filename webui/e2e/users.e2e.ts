import { expect, test } from '@playwright/test';

import { login } from './auth';
import { createUser } from './users-helpers';

test('@users create user modal closes with escape', async ({ page }) => {
  await login(page);
  await page.goto('/users');
  await expect(page.getByRole('heading', { name: 'Users' })).toBeVisible();

  await page.getByRole('button', { name: 'Add User' }).click();
  const dialog = page.getByRole('dialog', { name: 'Create User' });
  await expect(dialog).toBeVisible();

  await page.keyboard.press('Escape');
  await expect(dialog).toBeHidden();
});

test('@users creates a user from the users page modal', async ({ page }) => {
  await login(page);

  const username = `e2e-user-${Date.now()}`;
  const email = `${username}@test.local`;

  await createUser(page, {
    username,
    email,
    fullName: 'E2E User',
    password: 'e2e-user-pass',
  });

  const row = page.locator('tbody tr').filter({ has: page.getByText(username, { exact: true }) });
  await expect(row).toContainText(email);
  await expect(row).toContainText('E2E User');
});
