import { expect, test } from '@playwright/test';

import { login } from './auth';

test('@mobile @smoke mobile sidebar navigation opens and routes correctly', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await login(page);

  const toggle = page.getByRole('button', { name: 'Toggle sidebar' });
  await toggle.click();
  await expect(page.locator('aside[aria-label="Main navigation"]')).toBeVisible();
  await expect(page.locator('div.bg-black\\/50').first()).toBeVisible();

  // On a phone only the current page's section starts open.
  const ipam = page.getByRole('button', { name: 'IPAM' });
  await expect(ipam).toHaveAttribute('aria-expanded', 'false');
  await expect(page.getByRole('link', { name: 'Networks' })).toBeHidden();
  await ipam.click();
  await expect(ipam).toHaveAttribute('aria-expanded', 'true');
  await page.getByRole('link', { name: 'Networks' }).click();
  await expect(page).toHaveURL('/networks');
  await expect(page.getByRole('heading', { name: 'Networks' })).toBeVisible();
});
