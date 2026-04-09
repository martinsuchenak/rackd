import { expect, test } from '@playwright/test';

import { login, logout } from './auth';
import { createDevice, createNetwork, rowByExactText, uniqueName } from './inventory-helpers';
import { createUser, grantUserRole } from './users-helpers';

test('@rbac operator role can mutate inventory but cannot access user administration', async ({ page }) => {
  await login(page);

  const username = uniqueName('e2e-operator');
  const password = 'operator-pass-123';
  const operatorNetworkName = uniqueName('e2e-operator-network');

  await createUser(page, {
    username,
    email: `${username}@test.local`,
    fullName: 'Operator User',
    password,
  });
  await grantUserRole(page, username, 'operator');

  await logout(page);
  await login(page, username, password);

  await page.goto('/networks');
  await expect(page.getByRole('button', { name: 'Add Network' })).toBeVisible();
  await page.getByRole('button', { name: 'Add Network' }).click();
  const dialog = page.getByRole('dialog', { name: 'Add Network' });
  await dialog.getByLabel(/^Name/).fill(operatorNetworkName);
  await dialog.getByLabel(/^Subnet/).fill('10.95.0.0/24');
  await dialog.getByRole('button', { name: 'Create Network' }).click();
  await expect(rowByExactText(page, operatorNetworkName)).toBeVisible();

  await expect(page.getByRole('link', { name: 'Users' })).toHaveCount(0);
  await page.goto('/users');
  await expect(page.getByText("You don't have permission to access this page.")).toBeVisible();
});

test('@rbac viewer role hides detail-page mutation controls', async ({ page }) => {
  await login(page);

  const networkName = uniqueName('e2e-viewer-detail-network');
  const deviceName = uniqueName('e2e-viewer-detail-device');
  const viewerUsername = uniqueName('e2e-viewer-detail');
  const viewerPassword = 'viewer-detail-pass';

  await createNetwork(page, {
    name: networkName,
    subnet: '10.78.0.0/24',
  });
  await createDevice(page, {
    name: deviceName,
    hostname: `${deviceName}.rackd.test`,
    ip: '10.78.0.10',
    networkName,
  });
  await createUser(page, {
    username: viewerUsername,
    email: `${viewerUsername}@test.local`,
    fullName: 'Viewer Detail',
    password: viewerPassword,
  });
  await grantUserRole(page, viewerUsername, 'viewer');

  await logout(page);
  await login(page, viewerUsername, viewerPassword);

  await page.goto('/devices');
  await page.getByRole('link', { name: deviceName }).click();
  await expect(page.getByRole('heading', { name: deviceName })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Edit' })).toHaveCount(0);
  await expect(page.getByRole('button', { name: /Add Relationship/ })).toHaveCount(0);
});
