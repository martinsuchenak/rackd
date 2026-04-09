import { expect, test } from '@playwright/test';

import { login } from './auth';
import { insertDiscoveredDevice } from './db-fixtures';
import { createDatacenter, createNetwork, rowByExactText, uniqueName } from './inventory-helpers';

test('@discovery discovery promotion creates an inventory device from a discovered host', async ({ page }) => {
  await login(page);

  const networkName = uniqueName('e2e-promote-network');
  const datacenterName = uniqueName('e2e-promote-dc');
  const discoveredIP = '10.98.0.42';
  const promotedName = uniqueName('e2e-promoted-device');

  await createDatacenter(page, { name: datacenterName, location: 'Perth DC' });
  await createNetwork(page, {
    name: networkName,
    subnet: '10.98.0.0/24',
  });

  insertDiscoveredDevice({
    networkName,
    ip: discoveredIP,
    hostname: 'promote-me.rackd.test',
    macAddress: '52:54:00:12:34:56',
    vendor: 'Supermicro',
    osGuess: 'Linux',
    openPorts: [22, 443],
    services: [{ port: 22, service: 'ssh' }, { port: 443, service: 'https' }],
  });

  await page.goto('/discovery');
  await expect(page.getByRole('heading', { name: 'Discovery' })).toBeVisible();

  const discoveredRow = page.locator('tbody tr').filter({ hasText: discoveredIP }).first();
  await expect(discoveredRow).toBeVisible();
  await discoveredRow.getByRole('button', { name: 'Promote' }).click();

  const dialog = page.getByRole('dialog', { name: 'Promote to Device' });
  await expect(dialog).toBeVisible();
  await dialog.locator('input[type="text"]').first().fill(promotedName);
  await dialog.locator('select').selectOption({ label: datacenterName });
  await dialog.locator('input[type="text"]').nth(1).fill('Supermicro 1U');
  await dialog.getByRole('button', { name: 'Promote Device' }).click();
  await expect(dialog).toBeHidden();

  await page.goto('/devices');
  const deviceRow = rowByExactText(page, promotedName);
  await expect(deviceRow).toBeVisible();
  await expect(deviceRow).toContainText(discoveredIP);

  await page.goto('/discovery');
  await expect(discoveredRow.getByRole('button', { name: 'Promote' })).toHaveCount(0);
});
