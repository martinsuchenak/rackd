import { expect, test } from '@playwright/test';

import { login } from './auth';
import { getIDByName } from './db-fixtures';
import { acceptNextDialog } from './ui-helpers';
import { createDevice, createNetwork, uniqueName } from './inventory-helpers';

test('@inventory @relationships device detail supports relationship note edit and remove flows', async ({ page }) => {
  await login(page);

  const networkName = uniqueName('e2e-rel-network');
  const parentName = uniqueName('e2e-parent-device');
  const childName = uniqueName('e2e-child-device');

  await createNetwork(page, {
    name: networkName,
    subnet: '10.94.0.0/24',
  });
  await createDevice(page, {
    name: parentName,
    hostname: `${parentName}.rackd.test`,
    ip: '10.94.0.10',
    networkName,
  });
  await createDevice(page, {
    name: childName,
    hostname: `${childName}.rackd.test`,
    ip: '10.94.0.11',
    networkName,
  });

  const parentID = getIDByName('devices', parentName);
  const childID = getIDByName('devices', childName);
  const addResponse = await page.request.post(`/api/devices/${parentID}/relationships`, {
    headers: {
      'X-Requested-With': 'XMLHttpRequest',
    },
    data: {
      child_id: childID,
      type: 'contains',
      notes: 'Primary uplink relationship',
    },
  });
  expect(addResponse.ok()).toBeTruthy();

  await page.goto('/devices');
  await page.getByRole('link', { name: parentName }).click();
  await expect(page.getByRole('heading', { name: parentName })).toBeVisible();

  const relationshipCard = page.locator('div').filter({ hasText: childName }).filter({ hasText: 'Contains' }).first();
  await expect(relationshipCard).toBeVisible();
  await expect(relationshipCard).toContainText('Primary uplink relationship');

  await relationshipCard.getByRole('button', { name: 'Edit note' }).click();
  await relationshipCard.locator('textarea').first().fill('Updated relationship note');
  await relationshipCard.getByRole('button', { name: 'Save' }).click();
  await expect(relationshipCard).toContainText('Updated relationship note');

  await acceptNextDialog(page, async () => {
    await relationshipCard.getByRole('button', { name: 'Remove' }).click({ force: true });
  });
  await expect(page.getByText('Updated relationship note')).toHaveCount(0);
});
