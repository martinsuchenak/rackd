import { expect, test } from '@playwright/test';

import { login, openUserMenu } from './auth';

test('@modal discovery scan modal closes via overlay click', async ({ page }) => {
  await login(page);

  await page.goto('/discovery');
  await page.getByRole('button', { name: 'New Scan' }).click();
  const dialog = page.getByRole('dialog', { name: 'New Discovery Scan' });
  await expect(dialog).toBeVisible();

  await page.mouse.click(10, 10);
  await expect(dialog).toBeHidden();
});

test('@modal user menu password modal closes with escape', async ({ page }) => {
  await login(page);

  await openUserMenu(page);
  await page.getByRole('menuitem', { name: 'Change Password' }).click();
  const dialog = page.getByRole('dialog', { name: 'Change Password' });
  await expect(dialog).toBeVisible();

  await page.keyboard.press('Escape');
  await expect(dialog).toBeHidden();
});

test('@modal scan profile modal supports top-right close control', async ({ page }) => {
  await login(page);

  await page.goto('/scan-profiles');
  await page.getByRole('button', { name: 'New Profile' }).click();
  const dialog = page.getByRole('dialog', { name: 'New Scan Profile' });
  await expect(dialog).toBeVisible();

  await dialog.getByRole('button', { name: 'Close dialog' }).click();
  await expect(dialog).toBeHidden();
});
