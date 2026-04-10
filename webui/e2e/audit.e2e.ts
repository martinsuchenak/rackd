import { expect, test } from '@playwright/test';

import { login } from './auth';
import { insertAuditLog } from './db-fixtures';
import { uniqueName } from './inventory-helpers';

test('@audit audit page supports filters, export links, pagination, and detail modal', async ({ page }) => {
  const marker = uniqueName('e2e-audit');
  const baseTime = Date.now();

  for (let i = 0; i < 55; i += 1) {
    insertAuditLog({
      action: 'create',
      resource: marker,
      resourceID: `resource-${i}`,
      userID: 'admin-user',
      username: 'admin',
      ipAddress: '127.0.0.1',
      changes: JSON.stringify({ sequence: i, marker }),
      status: 'success',
      source: 'e2e-audit-suite',
      timestamp: new Date(baseTime + i * 1000).toISOString(),
    });
  }

  await login(page);
  await page.goto('/audit');
  await expect(page.getByRole('heading', { name: 'Audit Logs' })).toBeVisible();

  await page.getByPlaceholder('Resource').fill(marker);
  await page.getByPlaceholder('Source', { exact: true }).fill('e2e-audit-suite');
  await page.locator('select').filter({ has: page.getByRole('option', { name: '50 rows' }) }).selectOption('50');
  await page.getByRole('button', { name: 'Apply' }).click();

  await expect(page).toHaveURL(new RegExp(`/audit\\?.*resource=${marker}.*source=e2e-audit-suite.*limit=50`));
  await expect(page.locator('tbody tr').filter({ has: page.getByText(marker, { exact: true }) }).first()).toBeVisible();

  await expect(page.locator('a', { hasText: 'Export JSON' })).toHaveAttribute('href', new RegExp(`resource=${marker}`));
  await expect(page.locator('a', { hasText: 'Export CSV' })).toHaveAttribute('href', /limit=50/);

  await expect(page.getByRole('button', { name: 'Next' })).toBeEnabled();
  await page.getByRole('button', { name: 'Next' }).click();
  await expect(page.getByText(/Page 2/)).toBeVisible();
  await expect(page.getByRole('button', { name: 'Previous' })).toBeEnabled();

  await page.getByRole('button', { name: 'Previous' }).click();
  await expect(page.getByText(/Page 1/)).toBeVisible();

  await page.locator('tbody tr').filter({ has: page.getByText(marker, { exact: true }) }).first().getByRole('button', { name: 'Details' }).click();
  const dialog = page.getByRole('dialog', { name: 'Audit Entry' });
  await expect(dialog).toBeVisible();
  await expect(dialog).toContainText(marker);
  await expect(dialog).toContainText('e2e-audit-suite');
});
