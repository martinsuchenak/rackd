interface OAuthClient {
  client_id: string;
  client_name: string;
  redirect_uris: string[];
  grant_types: string[];
  token_endpoint_auth_method: string;
  client_uri: string;
  is_confidential: boolean;
  created_at: string;
}

export function oauthClients() {
  return {
    clients: [] as OAuthClient[],
    loading: true,
    error: '',
    deleteTarget: null as OAuthClient | null,
    showDeleteModal: false,

    async init() {
      await this.loadClients();
    },

    async loadClients() {
      this.loading = true;
      this.error = '';
      try {
        const response = await fetch('/api/oauth/clients', {
          credentials: 'same-origin',
        });
        if (!response.ok) throw new Error('Failed to load clients');
        this.clients = await response.json() || [];
      } catch (e: any) {
        this.error = e.message || 'Failed to load OAuth clients';
      } finally {
        this.loading = false;
      }
    },

    confirmDelete(client: OAuthClient) {
      this.deleteTarget = client;
      this.showDeleteModal = true;
    },

    cancelDelete() {
      this.deleteTarget = null;
      this.showDeleteModal = false;
    },

    async doDelete() {
      if (!this.deleteTarget) return;
      try {
        const response = await fetch(`/api/oauth/clients/${this.deleteTarget.client_id}`, {
          method: 'DELETE',
          credentials: 'same-origin',
        });
        if (!response.ok) throw new Error('Failed to delete client');
        this.showDeleteModal = false;
        this.deleteTarget = null;
        await this.loadClients();
      } catch (e: any) {
        this.error = e.message || 'Failed to delete client';
      }
    },

    formatDate(dateStr: string): string {
      return new Date(dateStr).toLocaleDateString(undefined, {
        year: 'numeric', month: 'short', day: 'numeric',
        hour: '2-digit', minute: '2-digit',
      });
    },
  };
}
