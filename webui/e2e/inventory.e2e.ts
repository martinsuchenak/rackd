import { expect, test } from '@playwright/test';

import { login } from './auth';
import { createDatacenter, createDevice, createNetwork, uniqueName } from './inventory-helpers';

test('can create a datacenter and network, then open network detail', async ({ page }) => {
  await login(page);

  const datacenterName = uniqueName('e2e-dc');
  const networkName = uniqueName('e2e-net');

  await createDatacenter(page, {
    name: datacenterName,
    location: 'Perth Lab',
    description: 'E2E datacenter',
  });

  await createNetwork(page, {
    name: networkName,
    subnet: '10.77.0.0/24',
    vlanId: 77,
    description: 'E2E network',
  });

  await page.getByRole('link', { name: networkName, exact: true }).click();
  await expect(page).toHaveURL(/\/networks\/detail\?id=/);
  await expect(page.getByRole('heading', { name: networkName })).toBeVisible();
  await expect(page.getByText('10.77.0.0/24')).toBeVisible();
});

test('can create a device with an address and open its detail view', async ({ page }) => {
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
    subnet: '10.88.0.0/24',
  });

  await createDevice(page, {
    name: deviceName,
    hostname: `${deviceName}.rackd.test`,
    makeModel: 'Dell R740',
    os: 'Ubuntu 24.04',
    location: 'Rack 1',
    username: 'root',
    ip: '10.88.0.10',
    networkName,
  });

  await page.getByRole('link', { name: deviceName, exact: true }).click();
  await expect(page).toHaveURL(/\/devices\/detail\?id=/);
  await expect(page.getByRole('heading', { name: deviceName })).toBeVisible();
  await expect(page.getByText(`${deviceName}.rackd.test`)).toBeVisible();
  await expect(page.getByText('Dell R740')).toBeVisible();
  await expect(page.getByText('10.88.0.10')).toBeVisible();
  await expect(page.getByText(networkName)).toBeVisible();
});
