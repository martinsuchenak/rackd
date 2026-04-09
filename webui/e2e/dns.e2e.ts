import { expect, test } from '@playwright/test';

import { login } from './auth';
import { createDNSProvider, createDNSZone } from './dns-helpers';
import { createNetwork, rowByExactText, uniqueName } from './inventory-helpers';

test('dns providers support create, edit, and delete flows', async ({ page }) => {
  await login(page);

  const providerName = uniqueName('e2e-dns-provider');
  const updatedName = `${providerName}-updated`;

  await createDNSProvider(page, {
    name: providerName,
    endpoint: 'https://dns-provider.test.local',
    token: 'provider-token',
    description: 'Initial provider description',
  });

  await page.getByRole('button', { name: `Edit ${providerName}` }).click();
  const editDialog = page.getByRole('dialog', { name: 'Edit DNS Provider' });
  await expect(editDialog).toBeVisible();
  await editDialog.getByLabel(/^Name/).fill(updatedName);
  await editDialog.getByLabel(/^Endpoint/).fill('https://updated-dns-provider.test.local');
  await editDialog.getByLabel('Description').fill('Updated provider description');
  await editDialog.getByRole('button', { name: 'Save Changes' }).click();

  await expect(editDialog).toBeHidden();
  const updatedRow = rowByExactText(page, updatedName);
  await expect(updatedRow).toBeVisible();
  await expect(updatedRow).toContainText('https://updated-dns-provider.test.local');
  await expect(updatedRow).toContainText('Updated provider description');

  await page.getByRole('button', { name: `Delete ${updatedName}` }).click();
  const deleteDialog = page.getByRole('alertdialog', { name: 'Delete DNS Provider' });
  await expect(deleteDialog).toBeVisible();
  await deleteDialog.getByRole('button', { name: 'Delete' }).click();

  await expect(updatedRow).toHaveCount(0);
});

test('dns zones support create, records navigation, edit, and delete flows', async ({ page }) => {
  await login(page);

  const networkName = uniqueName('e2e-dns-network');
  const providerName = uniqueName('e2e-zone-provider');
  const zoneName = `${uniqueName('zone')}.example.test`;
  const updatedZoneName = `${uniqueName('zone-updated')}.example.test`;

  await createNetwork(page, {
    name: networkName,
    subnet: '10.66.0.0/24',
  });

  await createDNSProvider(page, {
    name: providerName,
    endpoint: 'https://zone-provider.test.local',
    token: 'zone-provider-token',
  });

  await createDNSZone(page, {
    name: zoneName,
    providerName,
    networkName,
    ttl: 3600,
    description: 'Initial DNS zone',
  });

  const zoneRow = rowByExactText(page, zoneName);
  await expect(zoneRow).toBeVisible();
  await expect(zoneRow).toContainText(providerName);
  await expect(zoneRow).toContainText(networkName);

  await page.getByRole('button', { name: `View records for ${zoneName}` }).click();
  await expect(page).toHaveURL(/\/dns\/records\//);
  await expect(page.getByRole('heading', { name: 'DNS Records' })).toBeVisible();
  await expect(page.getByText(zoneName, { exact: true })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Back to Zones' })).toBeVisible();

  await page.getByRole('link', { name: 'Back to Zones' }).click();
  await expect(page).toHaveURL('/dns/zones');

  await page.getByRole('button', { name: `Edit ${zoneName}` }).click();
  const editDialog = page.getByRole('dialog', { name: 'Edit DNS Zone' });
  await expect(editDialog).toBeVisible();
  await editDialog.getByLabel(/^Zone Name/).fill(updatedZoneName);
  await editDialog.getByLabel('Default TTL').fill('7200');
  await editDialog.getByLabel('Description').fill('Updated DNS zone');
  await editDialog.getByRole('button', { name: 'Save Changes' }).click();

  await expect(editDialog).toBeHidden();
  const updatedRow = rowByExactText(page, updatedZoneName);
  await expect(updatedRow).toBeVisible();
  await expect(updatedRow).toContainText(providerName);
  await expect(updatedRow).toContainText(networkName);

  await page.getByRole('button', { name: `Delete ${updatedZoneName}` }).click();
  const deleteDialog = page.getByRole('alertdialog', { name: 'Delete DNS Zone' });
  await expect(deleteDialog).toBeVisible();
  await deleteDialog.getByRole('button', { name: 'Delete' }).click();

  await expect(updatedRow).toHaveCount(0);
});
