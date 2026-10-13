import { expect, test } from '@playwright/test';

import { login } from './auth';

// Searches for the entry written by this test's own login. Logs live in a
// bounded in-memory buffer, so a startup message would be evicted by the time
// this runs in the full suite.
const LOG_MESSAGE = 'User logged in';

test('@logs application logs support filtering, export links, detail modal, and clearing filters', async ({ page }) => {
  await login(page);
  await page.goto('/logs');
  await expect(page.getByRole('heading', { name: 'Application Logs' })).toBeVisible();

  await page.locator('select').first().selectOption('info');
  await page.getByPlaceholder('Search message or fields').fill(LOG_MESSAGE);
  await page.getByRole('button', { name: 'Apply' }).click();

  await expect(page).toHaveURL(/\/logs\?.*level=info/);
  await expect(page).toHaveURL(/query=User\+logged\+in/);
  await expect(page.locator('tbody tr').filter({ hasText: LOG_MESSAGE }).first()).toBeVisible();

  await expect(page.locator('a', { hasText: 'Export JSON' })).toHaveAttribute('href', /level=info/);
  await expect(page.locator('a', { hasText: 'Export CSV' })).toHaveAttribute('href', /query=User\+logged\+in/);

  await page.locator('tbody tr').filter({ hasText: LOG_MESSAGE }).first().getByRole('button', { name: 'Details' }).click();
  const dialog = page.getByRole('dialog', { name: 'Log Entry' });
  await expect(dialog).toBeVisible();
  await expect(dialog).toContainText(LOG_MESSAGE);
  await dialog.getByRole('button', { name: 'Close dialog' }).click();
  await expect(dialog).toBeHidden();

  await page.getByRole('button', { name: 'Clear filters' }).click();
  await expect(page.locator('select').first()).toHaveValue('');
  await expect(page.getByPlaceholder('Search message or fields')).toHaveValue('');
});
