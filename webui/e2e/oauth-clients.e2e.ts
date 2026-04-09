import { expect, test } from '@playwright/test';

import { login } from './auth';
import { registerOAuthClient } from './oauth-helpers';

test('@oauth oauth clients list supports registered client delete flow', async ({ page }) => {
  await login(page);

  const client = await registerOAuthClient(page.request);

  await page.goto('/oauth-clients');
  await expect(page.getByRole('heading', { name: 'OAuth Clients' })).toBeVisible();

  const row = page.locator('tbody tr').filter({ hasText: client.client_name }).first();
  await expect(row).toBeVisible();
  await expect(row).toContainText(client.client_id);

  await row.getByRole('button', { name: `Delete client ${client.client_name}` }).click();
  const dialog = page.getByRole('alertdialog', { name: 'Delete OAuth Client' });
  await expect(dialog).toBeVisible();
  await dialog.getByRole('button', { name: 'Delete' }).click();
  await expect(row).toHaveCount(0);
});

test('@oauth oauth clients page stays usable when the API responds with HTML', async ({ page }) => {
  await login(page);

  await page.route('**/api/oauth/clients', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'text/html; charset=utf-8',
      body: '<!doctype html><html><body>OAuth disabled</body></html>',
    });
  });

  await page.goto('/oauth-clients');
  await expect(page.getByRole('heading', { name: 'OAuth Clients' })).toBeVisible();
  await expect(page.getByText('No OAuth clients registered.')).toBeVisible();
});
