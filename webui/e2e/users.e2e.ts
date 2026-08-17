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

  // Regression: a broken self-closed <input/> used to render raw
  // 'class="w-full ...' attribute text inside the open modal.
  await expect(page.evaluate(() => {
    const d = document.querySelector<HTMLElement>('[role="dialog"]');
    const text = d?.innerText ?? '';
    return !/class="/.test(text) && !/:class/.test(text) && !/x-model/.test(text);
  })).resolves.toBe(true);

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

test('@users edit user active status round-trips a boolean', async ({ page }) => {
  await login(page);

  const username = `e2e-active-${Date.now()}`;
  await createUser(page, {
    username,
    email: `${username}@test.local`,
    password: 'e2e-user-pass',
  });

  const row = page.locator('tbody tr').filter({ has: page.getByText(username, { exact: true }) });
  await row.getByRole('button', { name: `Edit ${username}` }).click();
  const dialog = page.getByRole('dialog', { name: 'Edit User' });
  await expect(dialog).toBeVisible();

  // Regression: <option :value="true"> made the select submit the literal
  // label string, so is_active="Active" failed server-side JSON decoding.
  await dialog.locator('select').first().selectOption('false');
  await dialog.getByRole('button', { name: 'Save Changes' }).click();
  await expect(dialog).toBeHidden();
  await expect(row).toContainText('Inactive');

  await row.getByRole('button', { name: `Edit ${username}` }).click();
  await expect(dialog).toBeVisible();
  await dialog.locator('select').first().selectOption('true');
  await dialog.getByRole('button', { name: 'Save Changes' }).click();
  await expect(dialog).toBeHidden();
  await expect(row).toContainText('Active');
});
