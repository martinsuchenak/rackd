import { expect, type Page } from '@playwright/test';

import { rowByExactText } from './inventory-helpers';

export async function createScanProfile(
  page: Page,
  values: {
    name: string;
    scanType?: 'quick' | 'full' | 'deep' | 'custom';
    ports?: string;
    enableSNMP?: boolean;
    enableSSH?: boolean;
    timeoutSec?: number;
    maxWorkers?: number;
    description?: string;
  },
): Promise<void> {
  await page.goto('/scan-profiles');
  await expect(page.getByRole('heading', { name: 'Scan Profiles' })).toBeVisible();

  await page.getByRole('button', { name: 'New Profile' }).click();
  const dialog = page.getByRole('dialog', { name: 'New Scan Profile' });
  await expect(dialog).toBeVisible();

  await dialog.locator('input[type="text"]').first().fill(values.name);
  if (values.scanType) {
    await dialog.locator('select').first().selectOption(values.scanType);
  }
  if (values.ports) {
    await dialog.locator('input[placeholder="22,80,443,3389"]').fill(values.ports);
  }
  if (values.enableSNMP !== undefined) {
    const checkbox = dialog.getByLabel('Enable SNMP');
    if ((await checkbox.isChecked()) !== values.enableSNMP) {
      await checkbox.click();
    }
  }
  if (values.enableSSH !== undefined) {
    const checkbox = dialog.getByLabel('Enable SSH');
    if ((await checkbox.isChecked()) !== values.enableSSH) {
      await checkbox.click();
    }
  }
  if (values.timeoutSec !== undefined) {
    await dialog.locator('input[type="number"]').first().fill(String(values.timeoutSec));
  }
  if (values.maxWorkers !== undefined) {
    await dialog.locator('input[type="number"]').nth(1).fill(String(values.maxWorkers));
  }
  if (values.description) {
    await dialog.locator('textarea').fill(values.description);
  }

  await dialog.getByRole('button', { name: 'Create Profile' }).click();
  await expect(dialog).toBeHidden();
  await expect(rowByExactText(page, values.name)).toBeVisible();
}
