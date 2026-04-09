import { expect, test } from '@playwright/test';

import { login } from './auth';
import { createDNSProvider, createDNSZone } from './dns-helpers';
import { createDevice, createNetwork, uniqueName } from './inventory-helpers';

test('dns records support linked-record navigation plus edit and delete flows', async ({ page }) => {
  await login(page);

  const networkName = uniqueName('e2e-dns-records-network');
  const providerName = uniqueName('e2e-dns-records-provider');
  const zoneName = `${uniqueName('records')}.example.test`;
  const deviceName = uniqueName('e2e-dns-record-device');

  await createNetwork(page, {
    name: networkName,
    subnet: '10.96.0.0/24',
  });
  await createDNSProvider(page, {
    name: providerName,
    endpoint: 'http://127.0.0.1:9',
    token: 'records-provider-token',
  });
  await createDNSZone(page, {
    name: zoneName,
    providerName,
    networkName,
    autoSync: true,
  });
  await createDevice(page, {
    name: deviceName,
    hostname: `${deviceName}.${zoneName}`,
    ip: '10.96.0.10',
    networkName,
  });

  await page.goto('/dns/zones');
  await page.getByRole('button', { name: `View records for ${zoneName}` }).click();
  await expect(page.getByRole('heading', { name: 'DNS Records' })).toBeVisible();

  const linkedRecordRow = page.locator('tbody tr').filter({ has: page.getByRole('link', { name: deviceName }) }).first();
  await expect(linkedRecordRow).toBeVisible();

  await linkedRecordRow.getByRole('link', { name: deviceName }).click();
  await expect(page).toHaveURL(/\/devices\/detail\?id=/);
  await expect(page.getByRole('heading', { name: deviceName })).toBeVisible();

  await page.goBack();
  await expect(page.getByRole('heading', { name: 'DNS Records' })).toBeVisible();

  await linkedRecordRow.getByRole('button', { name: /Edit record / }).click();
  const editDialog = page.getByRole('dialog', { name: 'Edit DNS Record' });
  await expect(editDialog).toBeVisible();
  await editDialog.getByLabel(/TTL \(seconds\)/).fill('1800');
  await editDialog.getByRole('button', { name: 'Save Changes' }).click();
  await expect(editDialog).toBeHidden();
  await expect(linkedRecordRow).toContainText('1800s');

  const beforeRows = await page.locator('tbody tr').count();
  await linkedRecordRow.getByRole('button', { name: /Delete record / }).click();
  const deleteDialog = page.getByRole('alertdialog', { name: 'Delete DNS Record' });
  await expect(deleteDialog).toBeVisible();
  await deleteDialog.getByRole('button', { name: 'Delete' }).click();
  await expect(page.locator('tbody tr')).toHaveCount(beforeRows - 1);
});
