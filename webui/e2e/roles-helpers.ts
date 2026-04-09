import { expect, type Page } from '@playwright/test';

import { rowByExactText } from './inventory-helpers';

export async function createRole(
  page: Page,
  values: { name: string; description?: string; permissions: string[] },
): Promise<void> {
  await page.goto('/roles');
  await expect(page.getByRole('heading', { name: 'Roles' })).toBeVisible();

  await page.getByRole('button', { name: 'Create Role' }).click();
  const dialog = page.getByRole('dialog', { name: 'Create Role' });
  await expect(dialog).toBeVisible();

  await dialog.getByLabel(/^Name/).fill(values.name);
  if (values.description) {
    await dialog.getByLabel('Description').fill(values.description);
  }
  for (const permissionName of values.permissions) {
    await dialog.getByLabel(permissionName, { exact: true }).check();
  }

  await dialog.getByRole('button', { name: 'Create Role' }).click();
  await expect(dialog).toBeHidden();
  await expect(rowByExactText(page, values.name)).toBeVisible();
}
