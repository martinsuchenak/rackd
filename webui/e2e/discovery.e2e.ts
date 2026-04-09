import { expect, test } from '@playwright/test';

import { login } from './auth';
import { createNetwork, uniqueName } from './inventory-helpers';

test('@discovery discovery scan modal validates and can start a scan', async ({ page }) => {
  await login(page);

  const networkName = uniqueName('e2e-discovery-network');
  await createNetwork(page, {
    name: networkName,
    subnet: '10.95.0.0/30',
  });

  await page.goto('/discovery');
  await expect(page.getByRole('heading', { name: 'Discovery' })).toBeVisible();

  await page.getByRole('button', { name: 'New Scan' }).click();
  const dialog = page.getByRole('dialog', { name: 'New Discovery Scan' });
  await expect(dialog).toBeVisible();
  const networkSelect = dialog.locator('select').first();

  await dialog.getByRole('button', { name: 'Start Scan' }).click();
  await expect.poll(async () => networkSelect.evaluate((el) => !(el as HTMLSelectElement).checkValidity())).toBe(true);

  await networkSelect.selectOption({ label: `${networkName} (10.95.0.0/30)` });
  await dialog.getByRole('button', { name: 'Start Scan' }).click();
  await expect(dialog).toBeHidden();

  await expect(page.locator('div').filter({ hasText: /quick scan/i }).first()).toBeVisible();
});
