// API Keys Management Component

import type { APIKey, CreateAPIKeyRequest, User } from '../core/types';
import { api, RackdAPIError } from '../core/api';

type ModalType = '' | 'create' | 'delete' | 'reveal';

export function apiKeysList() {
  return {
    keys: [] as APIKey[],
    users: [] as User[],
    loading: true,
    error: '',

    modalType: '' as ModalType,
    selectedKey: null as APIKey | null,
    newKeyValue: '',

    // Flat form fields (CSP Alpine doesn't allow x-model on nested obj properties)
    formName: '',
    formDescription: '',
    formExpiresAt: '',

    validationErrors: {} as Record<string, string>,
    saving: false,
    deleting: false,

    get showCreateModal(): boolean { return this.modalType === 'create'; },
    get showDeleteModal(): boolean { return this.modalType === 'delete'; },
    get showRevealModal(): boolean { return this.modalType === 'reveal'; },
    get deleteModalTitle(): string { return 'Delete API Key'; },
    get deleteModalName(): string { return this.getSelectedKeyName(); },
    get deleteModalDescription(): string {
      return `Are you sure you want to delete ${this.getSelectedKeyName()}? Any clients using this key will lose access immediately.`;
    },

    async init(): Promise<void> {
      await Promise.all([this.loadKeys(), this.loadUsers()]);
    },

    async loadKeys(): Promise<void> {
      this.loading = true;
      try {
        this.keys = await api.listAPIKeys();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load API keys';
      } finally {
        this.loading = false;
      }
    },

    async loadUsers(): Promise<void> {
      try {
        this.users = await api.listUsers();
      } catch {
        // non-critical
      }
    },

    getUserName(userId: string): string {
      const user = this.users.find((u) => u.id === userId);
      return user ? user.username : userId || '-';
    },

    openCreateModal(): void {
      this.modalType = '';
      this.formName = '';
      this.formDescription = '';
      this.formExpiresAt = '';
      this.validationErrors = {};
      this.modalType = 'create';
    },

    openDeleteModal(key: APIKey): void {
      this.modalType = '';
      this.selectedKey = key;
      this.modalType = 'delete';
    },

    closeModal(): void {
      this.modalType = '';
      this.selectedKey = null;
      this.newKeyValue = '';
      this.validationErrors = {};
    },

    cancelDelete(): void {
      this.closeModal();
    },

    validateForm(): boolean {
      this.validationErrors = {};
      if (!this.formName.trim()) {
        this.validationErrors.name = 'Name is required';
      }
      return Object.keys(this.validationErrors).length === 0;
    },

    async doCreate(): Promise<void> {
      if (!this.validateForm()) return;
      this.saving = true;
      this.error = '';
      try {
        const req: CreateAPIKeyRequest = {
          name: this.formName,
          description: this.formDescription || undefined,
          // Convert YYYY-MM-DD to ISO timestamp (end of day UTC)
          expires_at: this.formExpiresAt
            ? new Date(this.formExpiresAt + 'T23:59:59Z').toISOString()
            : undefined,
        };
        const created = await api.createAPIKey(req);
        this.newKeyValue = created.key || '';
        await this.loadKeys();
        this.modalType = 'reveal';
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to create API key';
      } finally {
        this.saving = false;
      }
    },

    async doDelete(): Promise<void> {
      if (!this.selectedKey) return;
      this.deleting = true;
      try {
        await api.deleteAPIKey(this.selectedKey.id);
        await this.loadKeys();
        this.closeModal();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete API key';
      } finally {
        this.deleting = false;
      }
    },

    copyKey(): void {
      if (this.newKeyValue) {
        navigator.clipboard.writeText(this.newKeyValue).then(() => {
          window.dispatchEvent(new CustomEvent('toast:success', {
            detail: { message: 'API key copied to clipboard' }
          }));
        });
      }
    },

    formatDate(dateStr: string | undefined): string {
      if (!dateStr) return 'Never';
      return new Date(dateStr).toLocaleString();
    },

    formatExpiry(dateStr: string | undefined): string {
      if (!dateStr) return 'Never';
      const d = new Date(dateStr);
      return d < new Date() ? `Expired ${d.toLocaleDateString()}` : d.toLocaleDateString();
    },

    isExpired(dateStr: string | undefined): boolean {
      if (!dateStr) return false;
      return new Date(dateStr) < new Date();
    },

    getSelectedKeyName(): string {
      return this.selectedKey?.name || '';
    },
  };
}
