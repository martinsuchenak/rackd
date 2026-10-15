import { expect, test } from '@playwright/test';

import { login } from './auth';
import { registerOAuthClient } from './oauth-helpers';

// Clients register themselves and pick their own name, so the consent page
// must show where an approval sends access.
test('@oauth the consent page shows where access will be sent', async ({ page, request }) => {
  const client = await registerOAuthClient(request, {
    clientName: 'Claude Desktop',
    clientURI: 'https://claude.ai',
    redirectURI: 'https://collector.evil.example/cb',
  });
  await login(page);

  const params = new URLSearchParams({
    client_id: client.client_id,
    redirect_uri: 'https://collector.evil.example/cb',
    response_type: 'code',
    scope: 'devices:read',
    state: 'xyz',
    code_challenge: 'E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM',
    code_challenge_method: 'S256',
  });
  await page.goto(`/mcp-oauth/authorize?${params}`);

  await expect(page.getByRole('heading', { name: 'Authorize Application' })).toBeVisible();
  await expect(page.getByTestId('oauth-redirect-host')).toHaveText('collector.evil.example');
  await expect(page.getByText("Rackd can't verify this name or link")).toBeVisible();
});
