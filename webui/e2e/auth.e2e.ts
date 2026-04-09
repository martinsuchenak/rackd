import { expect, test } from '@playwright/test';

import { login } from './helpers';

test('redirects unauthenticated users to login', async ({ page }) => {
  await page.goto('/users');
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole('heading', { name: /Welcome to/i })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible();
});

test('logs in with the bootstrapped admin user', async ({ page }) => {
  await login(page);
  await expect(page.getByRole('button', { name: /Test Admin/ })).toBeVisible();
});
