import type { OAuthClient } from '../core/types';
import { api, RackdAPIError } from '../core/api';

type ModalType = '' | 'delete';

export function oauthClients() {
  return {
    clients: [] as OAuthClient[],
    loading: true,
    error: '',
    modalType: '' as ModalType,
    selectedClient: null as OAuthClient | null,

    get showDeleteModal(): boolean {
      return this.modalType === 'delete';
    },

    async init(): Promise<void> {
      await this.loadClients();
    },

    async loadClients(): Promise<void> {
      this.loading = true;
      this.error = '';
      try {
        this.clients = await api.listOAuthClients();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load OAuth clients';
      } finally {
        this.loading = false;
      }
    },

    openDeleteModal(client: OAuthClient): void {
      this.modalType = '';
      this.selectedClient = client;
      this.modalType = 'delete';
    },

    closeModal(): void {
      this.modalType = '';
      this.selectedClient = null;
    },

    async doDelete(): Promise<void> {
      if (!this.selectedClient) return;
      try {
        await api.deleteOAuthClient(this.selectedClient.client_id);
        this.closeModal();
        await this.loadClients();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete OAuth client';
      }
    },

    formatDate(dateStr: string): string {
      return new Date(dateStr).toLocaleDateString(undefined, {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    },

    getClientTypeLabel(client: OAuthClient): string {
      return client.is_confidential ? 'Confidential' : 'Public';
    },

    getClientTypeClass(client: OAuthClient): string {
      return client.is_confidential
        ? 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400'
        : 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400';
    },

    getSelectedClientName(): string {
      return this.selectedClient?.client_name || '';
    },
  };
}
