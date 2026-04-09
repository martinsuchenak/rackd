import { expect, test } from '@playwright/test';

import { login } from './auth';
import { getIDByName, insertDiscoveryScan } from './db-fixtures';
import { createNetwork, uniqueName } from './inventory-helpers';
import { createScanProfile } from './scan-profile-helpers';

test('@discovery scheduled scan setup is reflected in discovery history for the same network', async ({ page }) => {
  await login(page);

  const networkName = uniqueName('e2e-scheduled-discovery-network');
  const profileName = uniqueName('e2e-scheduled-discovery-profile');
  const scheduleName = uniqueName('e2e-scheduled-discovery');
  const nextMinute = new Date(Date.now() + 60_000);
  const cronExpression = `${nextMinute.getMinutes()} ${nextMinute.getHours()} * * *`;

  await createNetwork(page, {
    name: networkName,
    subnet: '10.97.1.0/24',
  });
  await createScanProfile(page, {
    name: profileName,
    scanType: 'quick',
    description: 'Scheduled discovery profile',
  });

  await page.goto('/scheduled-scans');
  await page.getByRole('button', { name: 'Add new scheduled scan' }).click();
  const dialog = page.getByRole('dialog', { name: 'Add Schedule' });
  await dialog.getByLabel(/^Name/).fill(scheduleName);
  await dialog.getByLabel(/^Network/).selectOption({ label: `${networkName} (10.97.1.0/24)` });
  await dialog.getByLabel(/Scan Profile/).selectOption({ label: profileName });
  await dialog.getByLabel(/Cron Expression/).fill(cronExpression);
  await dialog.getByRole('button', { name: 'Save' }).click();
  await expect(dialog).toBeHidden();

  const networkID = getIDByName('networks', networkName);
  insertDiscoveryScan({
    networkName,
    scanType: 'quick',
    status: 'completed',
    totalHosts: 1,
    scannedHosts: 1,
    foundHosts: 0,
  });

  await page.goto('/discovery');
  await expect(page.locator('div').filter({ hasText: networkID }).first()).toBeVisible();
});
