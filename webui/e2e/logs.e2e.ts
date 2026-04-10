import { expect, test } from '@playwright/test';

import { login } from './auth';

test('@logs application logs support filtering, export links, detail modal, and clearing filters', async ({ page }) => {
  await login(page);
  await page.goto('/logs');
  await expect(page.getByRole('heading', { name: 'Application Logs' })).toBeVisible();

  await page.locator('select').first().selectOption('info');
  await page.getByPlaceholder('Search message or fields').fill('Discovery scheduler started');
  await page.getByRole('button', { name: 'Apply' }).click();

  await expect(page).toHaveURL(/\/logs\?.*level=info/);
  await expect(page).toHaveURL(/query=Discovery\+scheduler\+started/);
  await expect(page.locator('tbody tr').filter({ hasText: 'Discovery scheduler started' }).first()).toBeVisible();

  await expect(page.locator('a', { hasText: 'Export JSON' })).toHaveAttribute('href', /level=info/);
  await expect(page.locator('a', { hasText: 'Export CSV' })).toHaveAttribute('href', /query=Discovery\+scheduler\+started/);

  await page.locator('tbody tr').filter({ hasText: 'Discovery scheduler started' }).first().getByRole('button', { name: 'Details' }).click();
  const dialog = page.getByRole('dialog', { name: 'Log Entry' });
  await expect(dialog).toBeVisible();
  await expect(dialog).toContainText('Discovery scheduler started');
  await dialog.getByRole('button', { name: 'Close dialog' }).click();
  await expect(dialog).toBeHidden();

  await page.getByRole('button', { name: 'Clear filters' }).click();
  await expect(page.locator('select').first()).toHaveValue('');
  await expect(page.getByPlaceholder('Search message or fields')).toHaveValue('');
});
