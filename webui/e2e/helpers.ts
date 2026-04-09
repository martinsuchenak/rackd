import { expect, type Page } from '@playwright/test';

export async function login(page: Page, username = 'admin', password = 'securepassword123'): Promise<void> {
  await page.goto('/login');
  await page.getByLabel('Username').fill(username);
  await page.getByLabel('Password').fill(password);
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(page).toHaveURL(/\/$/);
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
}
