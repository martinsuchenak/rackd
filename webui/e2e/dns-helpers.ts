import { expect, type Page } from '@playwright/test';

import { rowByExactText } from './inventory-helpers';

export async function createDNSProvider(
  page: Page,
  values: {
    name: string;
    type?: 'technitium' | 'powerdns' | 'bind';
    endpoint: string;
    token: string;
    description?: string;
  },
): Promise<void> {
  await page.goto('/dns/providers');
  await expect(page.getByRole('heading', { name: 'DNS Providers' })).toBeVisible();

  await page.getByRole('button', { name: 'Add Provider' }).click();
  const dialog = page.getByRole('dialog', { name: 'Create DNS Provider' });
  await expect(dialog).toBeVisible();

  await dialog.getByLabel(/^Name/).fill(values.name);
  if (values.type) {
    await dialog.getByLabel('Type').selectOption(values.type);
  }
  await dialog.locator('#create-endpoint').fill(values.endpoint);
  await dialog.locator('#create-token').fill(values.token);
  if (values.description) {
    await dialog.locator('#create-description').fill(values.description);
  }
  await dialog.getByRole('button', { name: 'Create Provider' }).click();

  await expect(dialog).toBeHidden();
  await expect(rowByExactText(page, values.name)).toBeVisible();
}

export async function createDNSZone(
  page: Page,
  values: {
    name: string;
    providerName: string;
    networkName?: string;
    autoSync?: boolean;
    ttl?: number;
    description?: string;
  },
): Promise<void> {
  await page.goto('/dns/zones');
  await expect(page.getByRole('heading', { name: 'DNS Zones' })).toBeVisible();

  await page.getByRole('button', { name: 'Add Zone' }).click();
  const dialog = page.getByRole('dialog', { name: 'Create DNS Zone' });
  await expect(dialog).toBeVisible();

  await dialog.locator('#create-name').fill(values.name);
  await dialog.locator('#create-provider').selectOption({ label: values.providerName });
  if (values.networkName) {
    await dialog.locator('#create-network').selectOption({ label: values.networkName });
  }
  if (values.autoSync !== undefined) {
    const checkbox = dialog.getByLabel('Auto-Sync');
    if ((await checkbox.isChecked()) !== values.autoSync) {
      await checkbox.click();
    }
  }
  if (values.ttl !== undefined) {
    await dialog.locator('#create-ttl').fill(String(values.ttl));
  }
  if (values.description) {
    await dialog.locator('#create-description').fill(values.description);
  }
  await dialog.getByRole('button', { name: 'Create Zone' }).click();

  await expect(dialog).toBeHidden();
  await expect(rowByExactText(page, values.name)).toBeVisible();
}
