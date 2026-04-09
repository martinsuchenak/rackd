import { expect, test } from '@playwright/test';

import { login } from './auth';
import { createDatacenter, createDevice, createNetwork, uniqueName } from './inventory-helpers';

test('@inventory datacenter detail supports edit flow', async ({ page }) => {
  await login(page);

  const datacenterName = uniqueName('e2e-detail-dc');
  const updatedName = `${datacenterName}-updated`;

  await createDatacenter(page, {
    name: datacenterName,
    location: 'Sydney',
    description: 'Detail datacenter',
  });

  await page.goto('/datacenters');
  await page.getByRole('link', { name: datacenterName }).click();
  await expect(page).toHaveURL(/\/datacenters\/detail\?id=/);

  await page.getByRole('button', { name: 'Edit' }).click();
  const dialog = page.getByRole('dialog', { name: 'Edit Datacenter' });
  await expect(dialog).toBeVisible();
  await dialog.locator('input[type="text"]').first().fill(updatedName);
  await dialog.locator('input[type="text"]').nth(1).fill('Melbourne');
  await dialog.getByRole('button', { name: 'Save Changes' }).click();
  await expect(dialog).toBeHidden();

  await expect(page.getByRole('heading', { name: updatedName })).toBeVisible();
  await expect(page.getByText('Melbourne', { exact: true })).toBeVisible();
});

test('@inventory network detail supports pool create and delete flows', async ({ page }) => {
  await login(page);

  const networkName = uniqueName('e2e-detail-net');
  const poolName = uniqueName('e2e-pool');

  await createNetwork(page, {
    name: networkName,
    subnet: '10.93.0.0/24',
  });

  await page.goto('/networks');
  await page.getByRole('link', { name: networkName }).click();
  await expect(page).toHaveURL(/\/networks\/detail\?id=/);

  await page.getByRole('button', { name: 'Add Pool' }).click();
  const dialog = page.getByRole('dialog', { name: 'Add Pool' });
  await expect(dialog).toBeVisible();
  await dialog.getByLabel(/^Name/).fill(poolName);
  await dialog.getByLabel(/^Start IP/).fill('10.93.0.20');
  await dialog.getByLabel(/^End IP/).fill('10.93.0.30');
  await dialog.getByLabel('Gateway').fill('10.93.0.1');
  await dialog.getByRole('button', { name: 'Create Pool' }).click();
  await expect(dialog).toBeHidden();

  const poolRow = page.locator('tbody tr').filter({ has: page.getByText(poolName, { exact: true }) });
  await expect(poolRow).toBeVisible();

  await poolRow.getByRole('button', { name: `Delete pool ${poolName}` }).click();
  const deleteDialog = page.getByRole('alertdialog', { name: 'Delete Pool' });
  await expect(deleteDialog).toBeVisible();
  await deleteDialog.getByRole('button', { name: 'Delete' }).click();
  await expect(poolRow).toHaveCount(0);
});

test('@inventory device detail supports edit flow', async ({ page }) => {
  await login(page);

  const networkName = uniqueName('e2e-detail-device-net');
  const deviceName = uniqueName('e2e-detail-device');
  const updatedName = `${deviceName}-updated`;

  await createNetwork(page, {
    name: networkName,
    subnet: '10.94.0.0/24',
  });
  await createDevice(page, {
    name: deviceName,
    hostname: `${deviceName}.rackd.test`,
    ip: '10.94.0.10',
    networkName,
  });

  await page.goto('/devices');
  await page.getByRole('link', { name: deviceName }).click();
  await expect(page).toHaveURL(/\/devices\/detail\?id=/);

  await page.getByRole('button', { name: 'Edit' }).click();
  const dialog = page.getByRole('dialog', { name: 'Edit Device' });
  await expect(dialog).toBeVisible();
  await dialog.locator('input[type="text"]').first().fill(updatedName);
  await dialog.locator('input[type="text"]').nth(2).fill('Juniper EX4300');
  await dialog.getByRole('button', { name: 'Save Changes' }).click();
  await expect(dialog).toBeHidden();

  await expect(page.getByRole('heading', { name: updatedName })).toBeVisible();
  await expect(page.getByText('Juniper EX4300', { exact: true })).toBeVisible();
});
