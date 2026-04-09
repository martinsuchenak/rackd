import { expect, type Locator, type Page } from '@playwright/test';

export function uniqueName(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

export function rowByExactText(page: Page, text: string): Locator {
  return page.locator('tbody tr').filter({ has: page.getByText(text, { exact: true }) });
}

export async function createDatacenter(
  page: Page,
  values: { name: string; location: string; description?: string },
): Promise<void> {
  await page.goto('/datacenters');
  await expect(page.getByRole('heading', { name: 'Datacenters' })).toBeVisible();

  await page.getByRole('button', { name: 'Add Datacenter' }).click();
  const dialog = page.getByRole('dialog', { name: 'Add Datacenter' });
  await expect(dialog).toBeVisible();

  await dialog.getByLabel(/^Name/).fill(values.name);
  await dialog.getByLabel('Location').fill(values.location);
  if (values.description) {
    await dialog.getByLabel('Description').fill(values.description);
  }
  await dialog.getByRole('button', { name: 'Create Datacenter' }).click();

  await expect(dialog).toBeHidden();
  const row = rowByExactText(page, values.name);
  await expect(row).toBeVisible();
  await expect(row).toContainText(values.location);
}

export async function createNetwork(
  page: Page,
  values: { name: string; subnet: string; vlanId?: number; description?: string },
): Promise<void> {
  await page.goto('/networks');
  await expect(page.getByRole('heading', { name: 'Networks' })).toBeVisible();

  await page.getByRole('button', { name: 'Add Network' }).click();
  const dialog = page.getByRole('dialog', { name: 'Add Network' });
  await expect(dialog).toBeVisible();

  await dialog.getByLabel(/^Name/).fill(values.name);
  await dialog.getByLabel(/^Subnet/).fill(values.subnet);
  if (values.vlanId !== undefined) {
    await dialog.locator('#network-vlan').fill(String(values.vlanId));
  }
  if (values.description) {
    await dialog.getByLabel('Description').fill(values.description);
  }
  await dialog.getByRole('button', { name: 'Create Network' }).click();

  await expect(dialog).toBeHidden();
  const row = rowByExactText(page, values.name);
  await expect(row).toBeVisible();
  await expect(row).toContainText(values.subnet);
}

export async function createDevice(
  page: Page,
  values: {
    name: string;
    hostname?: string;
    makeModel?: string;
    os?: string;
    location?: string;
    username?: string;
    ip: string;
    networkName: string;
  },
): Promise<void> {
  await page.goto('/devices');
  await expect(page.getByRole('heading', { name: 'Devices' })).toBeVisible();

  await page.getByRole('button', { name: 'Add Device' }).click();
  const dialog = page.getByRole('dialog', { name: 'Add Device' });
  await expect(dialog).toBeVisible();

  await dialog.getByLabel(/^Name/).fill(values.name);
  if (values.hostname) {
    await dialog.locator('#device-hostname').fill(values.hostname);
  }
  if (values.makeModel) {
    await dialog.locator('#device-make-model').fill(values.makeModel);
  }
  if (values.os) {
    await dialog.locator('#device-os').fill(values.os);
  }
  if (values.location) {
    await dialog.locator('#device-location').fill(values.location);
  }
  if (values.username) {
    await dialog.locator('#device-username').fill(values.username);
  }

  await dialog.getByRole('tab', { name: /Addresses/ }).click();
  await dialog.getByRole('button', { name: '+ Add Address' }).click();
  await dialog.locator('#addr-network-0').selectOption({ label: values.networkName });
  await dialog.getByLabel(/^IP Address/).fill(values.ip);

  await dialog.getByRole('button', { name: 'Create Device' }).click();

  await expect(dialog).toBeHidden();
  const row = rowByExactText(page, values.name);
  await expect(row).toBeVisible();
  await expect(row).toContainText(values.ip);
}
