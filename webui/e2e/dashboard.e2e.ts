import { expect, test } from '@playwright/test';

import { login } from './auth';

test('dashboard overview cards navigate to the main inventory pages', async ({ page }) => {
  await login(page);

  const overview = page.getByRole('list', { name: 'Statistics overview' });

  await overview.locator('a[href="/devices"]').click();
  await expect(page).toHaveURL(/\/devices$/);
  await expect(page.getByRole('heading', { name: 'Devices' })).toBeVisible();

  await page.goto('/');
  await page.getByRole('list', { name: 'Statistics overview' }).locator('a[href="/networks"]').click();
  await expect(page).toHaveURL(/\/networks$/);
  await expect(page.getByRole('heading', { name: 'Networks' })).toBeVisible();

  await page.goto('/');
  await page.getByRole('list', { name: 'Statistics overview' }).locator('a[href="/datacenters"]').click();
  await expect(page).toHaveURL(/\/datacenters$/);
  await expect(page.getByRole('heading', { name: 'Datacenters' })).toBeVisible();
});
