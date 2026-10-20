import { expect, test, type Page } from '@playwright/test';

import { login } from './auth';
import { uniqueName } from './inventory-helpers';

const headers = { 'X-Requested-With': 'XMLHttpRequest' };

async function apiDevice(page: Page, name: string, ip: string): Promise<string> {
  const res = await page.request.post('/api/v1/devices', {
    headers,
    data: { name, addresses: [{ ip, type: 'ipv4' }] },
  });
  expect(res.ok(), await res.text()).toBeTruthy();
  return (await res.json()).id;
}

// Regressions: conflicts naming deleted devices stayed listed, and "Detect
// Conflicts" emptied the list until a reload brought them back.
test('@conflicts the list follows the inventory and survives detection', async ({ page }) => {
  await login(page);
  // An address no earlier run used: the e2e database is shared across runs.
  const now = Date.now();
  const ip = `10.97.${Math.floor(now / 1000) % 250}.${10 + (now % 200)}`;
  const first = await apiDevice(page, uniqueName('e2e-dup-a'), ip);
  await apiDevice(page, uniqueName('e2e-dup-b'), ip);

  await page.goto('/conflicts');
  await page.getByRole('button', { name: /Detect/ }).click();
  const card = page.locator('div.rounded-lg').filter({ hasText: ip }).first();
  await expect(card).toBeVisible();

  // Detecting again keeps it listed.
  await page.getByRole('button', { name: /Detect/ }).click();
  await expect(card).toBeVisible();

  // Once one of the devices is gone, the conflict is too.
  const del = await page.request.delete(`/api/v1/devices/${first}`, { headers });
  expect(del.ok()).toBeTruthy();
  await page.reload();
  await expect(page.getByRole('heading', { name: /Conflicts/ }).first()).toBeVisible();
  await expect(page.getByText(ip, { exact: true })).toHaveCount(0);
});

// Regression: a device page for a missing device reloaded every 100ms,
// flickering between "Loading..." and the error.
test('@conflicts a missing device shows a steady not-found error', async ({ page }) => {
  await login(page);
  let requests = 0;
  const missing = '01a13e45-0000-7000-8000-000000000000';
  page.on('request', (r) => {
    if (r.url().includes(`/api/v1/devices/${missing}`)) requests++;
  });
  await page.goto(`/devices/detail?id=${missing}`);
  await expect(page.getByText(/not found/i)).toBeVisible();
  await page.waitForTimeout(1000);
  await expect(page.getByText(/not found/i)).toBeVisible();
  await expect(page.getByRole('status').filter({ hasText: 'Loading...' })).toBeHidden();
  expect(requests).toBe(1);
});
