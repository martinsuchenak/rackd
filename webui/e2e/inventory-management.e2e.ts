import { expect, test } from '@playwright/test';

import { login } from './auth';
import { createDatacenter, createDevice, createNetwork, rowByExactText, uniqueName } from './inventory-helpers';

test('@inventory datacenter list supports edit and delete flows', async ({ page }) => {
  await login(page);

  const datacenterName = uniqueName('e2e-dc');
  const updatedName = `${datacenterName}-updated`;

  await createDatacenter(page, {
    name: datacenterName,
    location: 'Sydney',
    description: 'Initial datacenter',
  });

  await page.getByRole('button', { name: `Edit ${datacenterName}` }).click();
  const editDialog = page.getByRole('dialog', { name: 'Edit Datacenter' });
  await expect(editDialog).toBeVisible();
  await editDialog.getByLabel(/^Name/).fill(updatedName);
  await editDialog.getByLabel('Location').fill('Melbourne');
  await editDialog.getByRole('button', { name: 'Save Changes' }).click();

  await expect(editDialog).toBeHidden();
  const updatedRow = rowByExactText(page, updatedName);
  await expect(updatedRow).toBeVisible();
  await expect(updatedRow).toContainText('Melbourne');

  await page.getByRole('button', { name: `Delete ${updatedName}` }).click();
  const deleteDialog = page.getByRole('alertdialog', { name: 'Delete Datacenter' });
  await expect(deleteDialog).toBeVisible();
  await deleteDialog.getByRole('button', { name: 'Delete' }).click();

  await expect(updatedRow).toHaveCount(0);
});

test('@inventory network list supports edit and delete flows', async ({ page }) => {
  await login(page);

  const datacenterName = uniqueName('e2e-dc');
  const networkName = uniqueName('e2e-net');
  const updatedName = `${networkName}-updated`;

  await createDatacenter(page, {
    name: datacenterName,
    location: 'Perth',
  });

  await createNetwork(page, {
    name: networkName,
    subnet: '10.55.0.0/24',
    description: 'Initial network',
  });

  await page.getByRole('button', { name: `Edit ${networkName}` }).click();
  const editDialog = page.getByRole('dialog', { name: 'Edit Network' });
  await expect(editDialog).toBeVisible();
  await editDialog.getByLabel(/^Name/).fill(updatedName);
  await editDialog.getByLabel(/^Subnet/).fill('10.55.1.0/24');
  await editDialog.getByRole('button', { name: 'Save Changes' }).click();

  await expect(editDialog).toBeHidden();
  const updatedRow = rowByExactText(page, updatedName);
  await expect(updatedRow).toBeVisible();
  await expect(updatedRow).toContainText('10.55.1.0/24');

  await page.getByRole('button', { name: `Delete ${updatedName}` }).click();
  const deleteDialog = page.getByRole('alertdialog', { name: 'Delete Network' });
  await expect(deleteDialog).toBeVisible();
  await deleteDialog.getByRole('button', { name: 'Delete' }).click();

  await expect(updatedRow).toHaveCount(0);
});

test('@inventory device list supports edit and delete flows', async ({ page }) => {
  await login(page);

  const datacenterName = uniqueName('e2e-dc');
  const networkName = uniqueName('e2e-net');
  const deviceName = uniqueName('e2e-device');
  const updatedName = `${deviceName}-updated`;

  await createDatacenter(page, {
    name: datacenterName,
    location: 'Perth',
  });

  await createNetwork(page, {
    name: networkName,
    subnet: '10.44.0.0/24',
  });

  await createDevice(page, {
    name: deviceName,
    hostname: `${deviceName}.rackd.test`,
    ip: '10.44.0.10',
    networkName,
  });

  await page.getByRole('button', { name: `Edit ${deviceName}` }).click();
  const editDialog = page.getByRole('dialog', { name: 'Edit Device' });
  await expect(editDialog).toBeVisible();
  await editDialog.getByLabel(/^Name/).fill(updatedName);
  await editDialog.locator('#device-make-model').fill('Dell R740');
  await editDialog.getByRole('button', { name: 'Save Changes' }).click();

  await expect(editDialog).toBeHidden();
  const updatedRow = rowByExactText(page, updatedName);
  await expect(updatedRow).toBeVisible();
  await expect(updatedRow).toContainText('Dell R740');

  await page.getByRole('button', { name: `Delete ${updatedName}` }).click();
  const deleteDialog = page.getByRole('alertdialog', { name: 'Delete Device' });
  await expect(deleteDialog).toBeVisible();
  await deleteDialog.getByRole('button', { name: 'Delete' }).click();

  await expect(updatedRow).toHaveCount(0);
});
