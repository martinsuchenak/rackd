import { expect, test } from '@playwright/test';

import { login } from './auth';
import { createDatacenter, createDevice, createNetwork, uniqueName } from './inventory-helpers';

test('global search supports keyboard navigation to a device detail page', async ({ page }) => {
  await login(page);

  const datacenterName = uniqueName('e2e-search-dc');
  const networkName = uniqueName('e2e-search-net');
  const deviceName = uniqueName('e2e-search-device');

  await createDatacenter(page, { name: datacenterName, location: 'Perth' });
  await createNetwork(page, { name: networkName, subnet: '10.91.0.0/24' });
  await createDevice(page, {
    name: deviceName,
    hostname: `${deviceName}.rackd.test`,
    ip: '10.91.0.10',
    networkName,
  });

  const search = page.getByLabel('Search devices, networks, datacenters');
  await search.fill(deviceName);
  const searchResults = page.locator('#search-results [role="option"]');
  await expect.poll(async () => await searchResults.count()).toBe(1);
  await expect(searchResults.first()).toContainText(deviceName);

  await search.press('ArrowDown');
  await search.press('Enter');

  await expect(page).toHaveURL(/\/devices\/detail\?id=/);
  await expect(page.getByRole('heading', { name: deviceName })).toBeVisible();
});
