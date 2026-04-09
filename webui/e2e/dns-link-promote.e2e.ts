import { expect, test } from '@playwright/test';

import { login } from './auth';
import { insertDNSRecord } from './db-fixtures';
import { createDNSProvider, createDNSZone } from './dns-helpers';
import { createDatacenter, createDevice, createNetwork, rowByExactText, uniqueName } from './inventory-helpers';

test('@dns dns records support linking unlinked CNAME records to an existing device', async ({ page }) => {
  await login(page);

  const networkName = uniqueName('e2e-dns-link-network');
  const providerName = uniqueName('e2e-dns-link-provider');
  const zoneName = `${uniqueName('link-zone')}.example.test`;
  const deviceName = uniqueName('e2e-dns-link-device');
  const recordName = 'app';

  await createNetwork(page, {
    name: networkName,
    subnet: '10.97.0.0/24',
  });
  await createDNSProvider(page, {
    name: providerName,
    endpoint: 'http://127.0.0.1:9',
    token: 'dns-link-provider-token',
  });
  await createDNSZone(page, {
    name: zoneName,
    providerName,
    networkName,
  });
  await createDevice(page, {
    name: deviceName,
    hostname: `${deviceName}.${zoneName}`,
    ip: '10.97.0.10',
    networkName,
  });

  insertDNSRecord({
    zoneName,
    name: recordName,
    type: 'CNAME',
    value: `${deviceName}.${zoneName}`,
  });

  await page.goto('/dns/zones');
  await page.getByRole('button', { name: `View records for ${zoneName}` }).click();
  await expect(page.getByRole('heading', { name: 'DNS Records' })).toBeVisible();

  const row = page.locator('tbody tr').filter({ hasText: recordName }).first();
  await expect(row).toContainText('Unlinked');
  await row.getByRole('button', { name: /Link record app/ }).click();

  const dialog = page.getByRole('dialog', { name: 'Link DNS Record to Device' });
  await expect(dialog).toBeVisible();
  await dialog.locator('#link-device-search').fill(deviceName);
  await dialog.getByLabel('Select a device').selectOption({ label: deviceName });
  await dialog.getByLabel('Add to device domains').check();
  await dialog.getByRole('button', { name: 'Link' }).click();
  await expect(dialog).toBeHidden();

  await expect(row.getByRole('link', { name: deviceName })).toBeVisible();
});

test('@dns dns records support promoting an unlinked A record to a new device', async ({ page }) => {
  await login(page);

  const datacenterName = uniqueName('e2e-dns-promote-dc');
  const networkName = uniqueName('e2e-dns-promote-network');
  const providerName = uniqueName('e2e-dns-promote-provider');
  const zoneName = `${uniqueName('promote-zone')}.example.test`;
  const recordName = 'orphan';
  const promotedName = uniqueName('e2e-dns-promoted-device');

  await createDatacenter(page, { name: datacenterName, location: 'Perth' });
  await createNetwork(page, {
    name: networkName,
    subnet: '10.99.0.0/24',
  });
  await createDNSProvider(page, {
    name: providerName,
    endpoint: 'http://127.0.0.1:9',
    token: 'dns-promote-provider-token',
  });
  await createDNSZone(page, {
    name: zoneName,
    providerName,
    networkName,
  });

  insertDNSRecord({
    zoneName,
    name: recordName,
    type: 'A',
    value: '10.99.0.25',
  });

  await page.goto('/dns/zones');
  await page.getByRole('button', { name: `View records for ${zoneName}` }).click();

  const row = page.locator('tbody tr').filter({ hasText: recordName }).first();
  await row.getByRole('button', { name: /Promote record orphan/ }).click();

  const dialog = page.getByRole('dialog', { name: 'Promote DNS Record to Device' });
  await expect(dialog).toBeVisible();
  await dialog.getByLabel('Device Name').fill(promotedName);
  await dialog.getByLabel('Datacenter').selectOption({ label: datacenterName });
  await dialog.getByLabel('Tags').fill('discovered,dns');
  await dialog.getByRole('button', { name: 'Promote' }).click();
  await expect(dialog).toBeHidden();

  await page.goto('/devices');
  await expect(rowByExactText(page, promotedName)).toBeVisible();
});
