import type { Page } from '@playwright/test';

export async function acceptNextDialog(page: Page, action: () => Promise<void>): Promise<void> {
  const dialogPromise = page.waitForEvent('dialog');
  await action();
  const dialog = await dialogPromise;
  await dialog.accept();
}
