import { expect, test } from '@playwright/test';

import { login } from './auth';
import { createDNSProvider } from './dns-helpers';
import { createNetwork, uniqueName } from './inventory-helpers';
import { createScanProfile } from './scan-profile-helpers';
import { createUser } from './users-helpers';

test('@errors duplicate usernames are rejected in the users modal', async ({ page }) => {
  await login(page);

  const username = uniqueName('e2e-duplicate-user');
  await createUser(page, {
    username,
    email: `${username}@test.local`,
    fullName: 'Original User',
    password: 'duplicate-user-pass',
  });

  await page.goto('/users');
  await page.getByRole('button', { name: 'Add User' }).click();
  const dialog = page.getByRole('dialog', { name: 'Create User' });
  await dialog.getByLabel(/^Username/).fill(username);
  await dialog.getByLabel(/^Email/).fill(`${uniqueName('new-email')}@test.local`);
  await dialog.getByLabel(/^Password/).fill('duplicate-user-pass');
  await dialog.getByRole('button', { name: 'Create User' }).click();

  await expect(dialog).toBeVisible();
  await expect(page.locator('tbody tr').filter({ has: page.getByText(username, { exact: true }) })).toHaveCount(1);
});

test('@errors invalid cron expressions are rejected for scheduled scans', async ({ page }) => {
  await login(page);

  const networkName = uniqueName('e2e-invalid-cron-network');
  const profileName = uniqueName('e2e-invalid-cron-profile');

  await createNetwork(page, {
    name: networkName,
    subnet: '10.93.0.0/24',
  });
  await createScanProfile(page, {
    name: profileName,
    description: 'Invalid cron profile',
  });

  await page.goto('/scheduled-scans');
  await page.getByRole('button', { name: 'Add new scheduled scan' }).click();
  const dialog = page.getByRole('dialog', { name: 'Add Schedule' });
  const scheduleName = uniqueName('e2e-invalid-cron');
  await dialog.getByLabel(/^Name/).fill(scheduleName);
  await dialog.getByLabel(/^Network/).selectOption({ label: `${networkName} (10.93.0.0/24)` });
  await dialog.getByLabel(/Scan Profile/).selectOption({ label: profileName });
  await dialog.getByLabel(/Cron Expression/).fill('not-a-cron');
  await dialog.getByRole('button', { name: 'Save' }).click();

  await expect(dialog).toBeVisible();
  await expect(page.locator('tbody tr').filter({ has: page.getByText(scheduleName, { exact: true }) })).toHaveCount(0);
});

test('@errors dns provider connection failures are surfaced in the test modal', async ({ page }) => {
  await login(page);

  const providerName = uniqueName('e2e-test-failure-provider');
  await createDNSProvider(page, {
    name: providerName,
    endpoint: 'http://127.0.0.1:9',
    token: 'test-provider-token',
  });

  await page.getByRole('button', { name: `Test connection to ${providerName}` }).click();
  const dialog = page.getByRole('dialog', { name: 'Test Connection' });
  await expect(dialog).toBeVisible();
  await dialog.getByRole('button', { name: 'Test Connection' }).click();
  await expect(dialog.locator('[role="alert"]')).toBeVisible();
});
