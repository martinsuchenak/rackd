import { expect, test } from '@playwright/test';

import { login } from './auth';
import { createDatacenter, createDevice, createNetwork, rowByExactText, uniqueName } from './inventory-helpers';

test('device list search filters to the created device', async ({ page }) => {
  await login(page);

  const datacenterName = uniqueName('e2e-dc');
  const networkName = uniqueName('e2e-net');
  const deviceName = uniqueName('e2e-device');

  await createDatacenter(page, {
    name: datacenterName,
    location: 'Perth Lab',
  });

  await createNetwork(page, {
    name: networkName,
    subnet: '10.66.0.0/24',
  });

  await createDevice(page, {
    name: deviceName,
    hostname: `${deviceName}.rackd.test`,
    ip: '10.66.0.10',
    networkName,
  });

  await page.goto('/devices');
  await expect(page.getByRole('heading', { name: 'Devices' })).toBeVisible();

  const search = page.locator('#device-search');
  await search.fill(deviceName);

  const row = rowByExactText(page, deviceName);
  await expect(row).toBeVisible();
  await expect(row).toContainText('10.66.0.10');

  await page.locator('#device-search').fill('definitely-not-a-device');
  await expect(page.getByText('No devices found')).toBeVisible();
});
