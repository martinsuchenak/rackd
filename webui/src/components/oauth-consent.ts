interface ConsentData {
  client_name: string;
  client_uri: string;
  scopes: string[];
  client_id: string;
  redirect_uri: string;
  state: string;
  scope: string;
  code_challenge: string;
  code_challenge_method: string;
  user: string;
}

export function oauthConsent() {
  return {
    loading: true,
    error: '',
    consentData: null as ConsentData | null,

    async init() {
      try {
        // Fetch consent data from the authorize endpoint using current URL params
        const params = new URLSearchParams(window.location.search);
        const response = await fetch('/mcp-oauth/authorize?' + params.toString(), {
          credentials: 'same-origin',
        });

        if (response.status === 302 || response.redirected) {
          // Redirect to login
          window.location.href = response.url;
          return;
        }

        if (!response.ok) {
          const data = await response.json();
          this.error = data.error_description || data.error || 'Authorization request failed';
          this.loading = false;
          return;
        }

        this.consentData = await response.json();
        this.loading = false;
      } catch (e) {
        this.error = 'Failed to load authorization request';
        this.loading = false;
      }
    },

    get scopeLabels(): string[] {
      if (!this.consentData) return [];
      return this.consentData.scopes.map(s => {
        if (s === '*') return 'Full access (all permissions)';
        return s.replace(':', ' - ');
      });
    },

    async approve() {
      if (!this.consentData) return;
      this.loading = true;
      this.error = '';

      try {
        const response = await fetch('/mcp-oauth/authorize', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'same-origin',
          body: JSON.stringify({
            client_id: this.consentData.client_id,
            redirect_uri: this.consentData.redirect_uri,
            scope: this.consentData.scope,
            state: this.consentData.state,
            code_challenge: this.consentData.code_challenge,
            code_challenge_method: this.consentData.code_challenge_method,
            approved: true,
          }),
        });

        if (!response.ok) {
          const data = await response.json();
          this.error = data.error_description || 'Authorization failed';
          this.loading = false;
          return;
        }

        const data = await response.json();
        if (data.redirect_uri) {
          window.location.href = data.redirect_uri;
        }
      } catch (e) {
        this.error = 'Failed to process authorization';
        this.loading = false;
      }
    },

    async deny() {
      if (!this.consentData) return;
      this.loading = true;

      try {
        const response = await fetch('/mcp-oauth/authorize', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'same-origin',
          body: JSON.stringify({
            client_id: this.consentData.client_id,
            redirect_uri: this.consentData.redirect_uri,
            scope: this.consentData.scope,
            state: this.consentData.state,
            code_challenge: this.consentData.code_challenge,
            code_challenge_method: this.consentData.code_challenge_method,
            approved: false,
          }),
        });

        // The server will redirect with error=access_denied
        if (response.redirected) {
          window.location.href = response.url;
          return;
        }

        const data = await response.json();
        if (data.redirect_uri) {
          window.location.href = data.redirect_uri;
        }
      } catch (e) {
        this.error = 'Failed to process denial';
        this.loading = false;
      }
    },
  };
}
