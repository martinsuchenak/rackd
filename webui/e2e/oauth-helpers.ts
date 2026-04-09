import { expect, type APIRequestContext } from '@playwright/test';

import { uniqueName } from './inventory-helpers';

export async function registerOAuthClient(
  request: APIRequestContext,
  values?: {
    clientName?: string;
    redirectURI?: string;
    clientURI?: string;
  },
): Promise<{ client_id: string; client_name: string }> {
  const clientName = values?.clientName || uniqueName('e2e-oauth-client');
  const redirectURI = values?.redirectURI || `https://${clientName}.example.test/callback`;
  const response = await request.post('/mcp-oauth/register', {
    data: {
      client_name: clientName,
      redirect_uris: [redirectURI],
      grant_types: ['authorization_code', 'refresh_token'],
      response_types: ['code'],
      token_endpoint_auth_method: 'client_secret_post',
      client_uri: values?.clientURI || `https://${clientName}.example.test`,
    },
  });

  expect(response.status()).toBe(201);
  const body = await response.json();
  return {
    client_id: body.client_id,
    client_name: body.client_name,
  };
}
