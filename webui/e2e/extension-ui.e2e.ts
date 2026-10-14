import { expect, test } from '@playwright/test';

import { login, logout } from './auth';
import { extensionBaseURL } from './servers';
import { createUser } from './users-helpers';

// These run against Rackd with the example extension in examples/hello.
test.use({ baseURL: extensionBaseURL });

test.describe('Extension UI @extensions', () => {
  test('an extension page draws, styles and calls its API', async ({ page }) => {
    await login(page);

    await page.getByRole('link', { name: 'Hello' }).click();
    await expect(page).toHaveURL(/\/hello$/);
    await expect(page).toHaveTitle('Hello - Rackd');

    const title = page.getByRole('heading', { name: 'Hello', exact: true });
    await expect(title).toBeVisible();
    // The extension's own stylesheet is loaded.
    await expect(title).toHaveCSS('font-weight', '700');
    await expect(page.locator('[data-hello="path"]')).toHaveText('/hello');
    await expect(page.locator('[data-hello="greeting"]')).toHaveText('Hello, admin');

    // An Alpine component the extension registered works in its markup.
    const counter = page.getByRole('button', { name: /Clicked \d+ times/ });
    await counter.click();
    await counter.click();
    await expect(counter).toHaveText('Clicked 2 times');

    // Routes below the page belong to it, and leaving it clears it.
    await page.goto('/hello/sub?x=1');
    await expect(page.locator('[data-hello="path"]')).toHaveText('/hello/sub');
    await page.getByRole('link', { name: 'Devices' }).click();
    await expect(page.getByRole('heading', { name: 'Devices' })).toBeVisible();
    await expect(page.locator('[data-hello="path"]')).toHaveCount(0);
  });

  test('an extension adds content to a device page', async ({ page }) => {
    await login(page);
    const name = `hello-device-${Date.now()}`;
    const response = await page.request.post('/api/v1/devices', {
      data: { name },
      headers: { 'X-Requested-With': 'XMLHttpRequest' },
    });
    expect(response.ok()).toBeTruthy();
    const device = await response.json();

    await page.goto(`/devices/detail?id=${device.id}`);
    await expect(page.getByText(`Hello from ${name}`)).toBeVisible();
  });

  test("an extension page checks the page's permission", async ({ page }) => {
    await login(page);
    const username = `hello-user-${Date.now()}`;
    await createUser(page, { username, email: `${username}@test.local`, password: 'securepassword123' });
    await logout(page);

    await login(page, username, 'securepassword123');
    await expect(page.getByRole('link', { name: 'Hello' })).toHaveCount(0);
    await page.goto('/hello');
    await expect(page.getByRole('heading', { name: 'Access Denied' })).toBeVisible();
    await expect(page.locator('[data-hello="path"]')).toHaveCount(0);
  });
});
