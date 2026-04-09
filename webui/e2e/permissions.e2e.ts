import { expect, test } from '@playwright/test';

import { login } from './auth';
import { createDNSProvider, createDNSZone } from './dns-helpers';
import { createNetwork, rowByExactText, uniqueName } from './inventory-helpers';
import { createUser, grantUserRole } from './users-helpers';

test('viewer role can access readable pages but write controls stay hidden', async ({ page }) => {
  await login(page);

  const providerName = uniqueName('e2e-viewer-provider');
  const networkName = uniqueName('e2e-viewer-network');
  const zoneName = `${uniqueName('viewer-zone')}.example.test`;
  const viewerUsername = uniqueName('e2e-viewer');
  const viewerPassword = 'viewer-pass-123';

  await createNetwork(page, {
    name: networkName,
    subnet: '10.77.0.0/24',
  });

  await createDNSProvider(page, {
    name: providerName,
    endpoint: 'https://viewer-dns-provider.test.local',
    token: 'viewer-provider-token',
  });

  await createDNSZone(page, {
    name: zoneName,
    providerName,
    networkName,
  });

  await createUser(page, {
    username: viewerUsername,
    email: `${viewerUsername}@test.local`,
    fullName: 'E2E Viewer',
    password: viewerPassword,
  });
  await grantUserRole(page, viewerUsername, 'viewer');

  await page.context().clearCookies();
  await login(page, viewerUsername, viewerPassword);

  await page.goto('/devices');
  await expect(page.getByRole('heading', { name: 'Devices' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Add Device' })).not.toBeVisible();

  await page.goto('/dns/providers');
  await expect(page.getByRole('heading', { name: 'DNS Providers' })).toBeVisible();
  const providerRow = rowByExactText(page, providerName);
  await expect(providerRow).toBeVisible();
  await expect(page.getByRole('button', { name: 'Add Provider' })).not.toBeVisible();
  await expect(providerRow.getByRole('button', { name: `Edit ${providerName}` })).not.toBeVisible();
  await expect(providerRow.getByRole('button', { name: `Delete ${providerName}` })).not.toBeVisible();

  await page.goto('/dns/zones');
  await expect(page.getByRole('heading', { name: 'DNS Zones' })).toBeVisible();
  const zoneRow = rowByExactText(page, zoneName);
  await expect(zoneRow).toBeVisible();
  await expect(page.getByRole('button', { name: 'Add Zone' })).not.toBeVisible();
  await expect(zoneRow.getByRole('button', { name: `View records for ${zoneName}` })).toBeVisible();
  await expect(zoneRow.getByRole('button', { name: `Edit ${zoneName}` })).not.toBeVisible();
  await expect(zoneRow.getByRole('button', { name: `Delete ${zoneName}` })).not.toBeVisible();

  await page.goto('/users');
  await expect(page.getByText("You don't have permission to access this page.")).toBeVisible();
});
