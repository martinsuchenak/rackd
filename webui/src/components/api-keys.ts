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
    // 'full': all of the creator's permissions; 'scoped': only formScopes.
    formAccess: 'full' as 'full' | 'scoped',
    formScopes: [] as string[],

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
      this.formAccess = 'full';
      this.formScopes = [];
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
      if (this.formAccess === 'scoped' && this.formScopes.length === 0) {
        this.validationErrors.scopes = 'Choose at least one permission, or give the key all your permissions';
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
          scopes: this.formAccess === 'scoped' ? [...this.formScopes] : undefined,
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

    // The permissions a new key can be limited to: the creator's own,
    // grouped by resource.
    get scopeGroups(): { resource: string; scopes: string[] }[] {
      const groups = new Map<string, string[]>();
      for (const p of window.rackdConfig?.user?.permissions ?? []) {
        const list = groups.get(p.resource) ?? [];
        list.push(p.resource + ':' + p.action);
        groups.set(p.resource, list);
      }
      return [...groups.entries()]
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([resource, scopes]) => ({ resource, scopes: scopes.sort() }));
    },
    get isScoped(): boolean { return this.formAccess === 'scoped'; },
    hasScope(scope: string): boolean { return this.formScopes.includes(scope); },
    toggleScope(scope: string): void {
      this.formScopes = this.hasScope(scope)
        ? this.formScopes.filter((s) => s !== scope)
        : [...this.formScopes, scope];
    },
    scopeAction(scope: string): string { return scope.slice(scope.indexOf(':') + 1); },

    isFullAccess(key: APIKey): boolean { return !key.scopes || key.scopes.length === 0; },
    accessLabel(key: APIKey): string {
      if (this.isFullAccess(key)) return 'Full access';
      return key.scopes!.length === 1 ? key.scopes![0] : `${key.scopes!.length} permissions`;
    },
    accessTitle(key: APIKey): string {
      return this.isFullAccess(key) ? 'Can do everything its owner can' : key.scopes!.join(', ');
    },
    get fullAccessCount(): number { return this.keys.filter((k) => this.isFullAccess(k)).length; },
    get fullAccessNotice(): string {
      const n = this.fullAccessCount;
      return `${n} ${n === 1 ? 'key has' : 'keys have'} full access: ${n === 1 ? 'it can' : 'they can'} do everything ${n === 1 ? 'its owner' : 'their owners'} can. Give integrations a key limited to the permissions they need.`;
    },
  };
}
