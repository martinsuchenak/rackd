import { expect, test } from '@playwright/test';

import { login } from './auth';

test('@mobile @smoke mobile sidebar navigation opens and routes correctly', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await login(page);

  const toggle = page.getByRole('button', { name: 'Toggle sidebar' });
  await toggle.click();
  await expect(page.locator('aside[aria-label="Main navigation"]')).toBeVisible();
  await expect(page.locator('div.bg-black\\/50').first()).toBeVisible();

  await page.getByRole('link', { name: 'Networks' }).click();
  await expect(page).toHaveURL('/networks');
  await expect(page.getByRole('heading', { name: 'Networks' })).toBeVisible();
});
