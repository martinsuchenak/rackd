import { expect, test } from '@playwright/test';

import { login, openUserMenu } from './auth';

test('discovery scan modal closes via overlay click', async ({ page }) => {
  await login(page);

  await page.goto('/discovery');
  await page.getByRole('button', { name: 'New Scan' }).click();
  const dialog = page.getByRole('dialog', { name: 'New Discovery Scan' });
  await expect(dialog).toBeVisible();

  await page.mouse.click(10, 10);
  await expect(dialog).toBeHidden();
});

test('user menu password modal closes with escape', async ({ page }) => {
  await login(page);

  await openUserMenu(page);
  await page.getByRole('menuitem', { name: 'Change Password' }).click();
  const dialog = page.getByRole('dialog', { name: 'Change Password' });
  await expect(dialog).toBeVisible();

  await page.keyboard.press('Escape');
  await expect(dialog).toBeHidden();
});
