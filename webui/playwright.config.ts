import { defineConfig, devices } from '@playwright/test';

import { extensionBaseURL } from './e2e/servers';

const baseURL = process.env.E2E_BASE_URL || 'http://127.0.0.1:18080';

export default defineConfig({
  testDir: './e2e',
  testMatch: '**/*.e2e.ts',
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 2 : 0,
  reporter: [['list'], ['html', { open: 'never' }]],
  timeout: 30_000,
  expect: {
    timeout: 10_000,
  },
  use: {
    baseURL,
    trace: 'on-first-retry',
  },
  webServer: [
    {
      command: './webui/scripts/run-e2e-server.sh',
      cwd: '..',
      url: `${baseURL}/login`,
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
      stdout: 'pipe',
      stderr: 'pipe',
    },
    {
      command: './webui/scripts/run-e2e-server.sh',
      cwd: '..',
      url: `${extensionBaseURL}/login`,
      env: {
        E2E_PACKAGE: './examples/hello',
        E2E_DATA_NAME: 'e2e-extension-data',
        E2E_LISTEN_ADDR: new URL(extensionBaseURL).host,
      },
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
      stdout: 'pipe',
      stderr: 'pipe',
    },
  ],
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
