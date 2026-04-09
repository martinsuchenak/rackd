import { expect, test } from '@playwright/test';

import { login } from './auth';
import { rowByExactText, uniqueName } from './inventory-helpers';

test('@credentials credentials support create, edit, and delete flows', async ({ page }) => {
  await login(page);

  const credentialName = uniqueName('e2e-credential');
  const updatedName = `${credentialName}-updated`;

  await page.goto('/credentials');
  await expect(page.getByRole('heading', { name: 'Credentials' })).toBeVisible();

  await page.getByRole('button', { name: 'Add new credential' }).click();
  const dialog = page.getByRole('dialog', { name: 'Add Credential' });
  await expect(dialog).toBeVisible();

  await dialog.getByLabel(/^Name/).fill(credentialName);
  await dialog.getByLabel('Type').selectOption('ssh_password');
  await dialog.getByLabel(/^Username/).fill('rackd');
  await dialog.locator('#cred-ssh-secret').fill('credential-secret');
  await dialog.getByLabel('Description').fill('Initial SSH credential');
  await dialog.getByRole('button', { name: 'Save' }).click();
  await expect(dialog).toBeHidden();

  const row = rowByExactText(page, credentialName);
  await expect(row).toBeVisible();
  await expect(row).toContainText('SSH_PASSWORD');

  await row.getByRole('button', { name: /Edit/i }).last().click();
  const editDialog = page.getByRole('dialog', { name: 'Edit Credential' });
  await expect(editDialog).toBeVisible();
  await editDialog.getByLabel(/^Name/).fill(updatedName);
  await editDialog.getByLabel(/^Username/).fill('rackd');
  await editDialog.locator('#cred-ssh-secret').fill('credential-secret-updated');
  await editDialog.getByLabel('Description').fill('Updated SSH credential');
  await editDialog.getByRole('button', { name: 'Save' }).click();
  await expect(editDialog).toBeHidden();

  const updatedRow = rowByExactText(page, updatedName);
  await expect(updatedRow).toBeVisible();
  await expect(updatedRow).toContainText('Updated SSH credential');

  await updatedRow.getByRole('button', { name: /Delete/i }).click();
  const deleteDialog = page.getByRole('alertdialog', { name: 'Delete Credential' });
  await expect(deleteDialog).toBeVisible();
  await deleteDialog.getByRole('button', { name: 'Delete' }).click();
  await expect(updatedRow).toHaveCount(0);
});
