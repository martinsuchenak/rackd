import { expect, type Page } from '@playwright/test';

import { rowByExactText } from './inventory-helpers';

export async function createUser(
  page: Page,
  values: { username: string; email: string; fullName?: string; password: string },
): Promise<void> {
  await page.goto('/users');
  await expect(page.getByRole('heading', { name: 'Users' })).toBeVisible();

  await page.getByRole('button', { name: 'Add User' }).click();
  const dialog = page.getByRole('dialog', { name: 'Create User' });
  await expect(dialog).toBeVisible();

  await dialog.getByLabel(/^Username/).fill(values.username);
  await dialog.getByLabel(/^Email/).fill(values.email);
  if (values.fullName) {
    await dialog.getByLabel('Full Name').fill(values.fullName);
  }
  await dialog.getByLabel(/^Password/).fill(values.password);
  await dialog.getByRole('button', { name: 'Create User' }).click();

  await expect(dialog).toBeHidden();
  await expect(rowByExactText(page, values.username)).toBeVisible();
}

export async function grantUserRole(page: Page, username: string, roleName: string): Promise<void> {
  await page.goto('/users');
  await expect(page.getByRole('heading', { name: 'Users' })).toBeVisible();

  const row = rowByExactText(page, username);
  await expect(row).toBeVisible();
  await row.getByRole('button', { name: `Manage roles for ${username}` }).click();

  const dialog = page.getByRole('dialog', { name: 'Manage Roles' });
  await expect(dialog).toBeVisible();

  const roleCard = dialog
    .getByText(roleName, { exact: true })
    .locator('xpath=ancestor::div[contains(@class,"justify-between")][1]');

  await expect(roleCard).toBeVisible();
  await roleCard.getByRole('button', { name: 'Grant' }).click();
  await expect(roleCard.getByRole('button', { name: 'Revoke' })).toBeVisible();

  await dialog.getByRole('button', { name: 'Close', exact: true }).click();
  await expect(dialog).toBeHidden();
}
