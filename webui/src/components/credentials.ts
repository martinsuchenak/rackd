import type { Credential, CredentialInput, CredentialType } from '../core/types';
import { api, RackdAPIError } from '../core/api';

type ModalType = '' | 'form' | 'delete';

interface CredentialFormData {
  id: string;
  name: string;
  type: CredentialType;
  description: string;
  datacenter_id: string;
  snmp_community: string;
  snmp_v3_user: string;
  snmp_v3_auth: string;
  snmp_v3_priv: string;
  ssh_username: string;
  ssh_key_id: string;
}

export function credentialsList() {
  return {
    credentials: [] as Credential[],
    loading: true,
    error: '',
    modalType: '' as ModalType,
    deleteTarget: null as Credential | null,
    form: resetForm(),

    get showFormModal(): boolean {
      return this.modalType === 'form';
    },

    get showDeleteModal(): boolean {
      return this.modalType === 'delete';
    },

    async init(): Promise<void> {
      await this.load();
    },

    async load(): Promise<void> {
      this.loading = true;
      this.error = '';
      try {
        this.credentials = await api.listCredentials();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load credentials';
      } finally {
        this.loading = false;
      }
    },

    openAddModal(): void {
      this.modalType = '';
      this.form = resetForm();
      this.error = '';
      this.modalType = 'form';
    },

    openEditModal(cred: Credential): void {
      this.modalType = '';
      this.form = {
        id: cred.id,
        name: cred.name,
        type: cred.type,
        description: cred.description || '',
        datacenter_id: cred.datacenter_id || '',
        snmp_community: '',
        snmp_v3_user: '',
        snmp_v3_auth: '',
        snmp_v3_priv: '',
        ssh_username: cred.ssh_username || '',
        ssh_key_id: '',
      };
      this.error = '';
      this.modalType = 'form';
    },

    closeModal(): void {
      this.modalType = '';
      this.deleteTarget = null;
      this.form = resetForm();
      this.error = '';
    },

    async save(): Promise<void> {
      this.error = '';
      const payload: CredentialInput = {
        name: this.form.name,
        type: this.form.type,
        description: this.form.description || undefined,
        datacenter_id: this.form.datacenter_id || undefined,
        snmp_community: this.form.snmp_community || undefined,
        snmp_v3_user: this.form.snmp_v3_user || undefined,
        snmp_v3_auth: this.form.snmp_v3_auth || undefined,
        snmp_v3_priv: this.form.snmp_v3_priv || undefined,
        ssh_username: this.form.ssh_username || undefined,
        ssh_key_id: this.form.ssh_key_id || undefined,
      };

      try {
        if (this.form.id) {
          await api.updateCredential(this.form.id, payload);
        } else {
          await api.createCredential(payload);
        }
        this.closeModal();
        await this.load();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to save credential';
      }
    },

    openDeleteModal(cred: Credential): void {
      this.modalType = '';
      this.deleteTarget = cred;
      this.modalType = 'delete';
    },

    async deleteConfirmed(): Promise<void> {
      if (!this.deleteTarget) return;
      try {
        await api.deleteCredential(this.deleteTarget.id);
        this.closeModal();
        await this.load();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete credential';
      }
    },

    hasCredentials(): boolean {
      return this.credentials.length > 0;
    },

    getTypeClass(type: string): string {
      if (type.startsWith('snmp')) {
        return 'bg-blue-100 text-blue-800 border-blue-200 dark:bg-blue-900/30 dark:text-blue-400 dark:border-blue-800';
      }
      if (type.startsWith('ssh')) {
        return 'bg-green-100 text-green-800 border-green-200 dark:bg-green-900/30 dark:text-green-400 dark:border-green-800';
      }
      return 'bg-gray-100 text-gray-800 border-gray-200 dark:bg-gray-900/30 dark:text-gray-400 dark:border-gray-800';
    },

    getSSHSecretLabel(): string {
      return (this.form.type === 'ssh_password' ? 'Password' : 'Private Key') + ' *';
    },

    getCredentialAriaLabel(credName: string, action: string): string {
      return action + ' credential: ' + credName;
    },

    getDeleteTargetName(): string {
      return this.deleteTarget?.name || '';
    },
  };
}

function resetForm(): CredentialFormData {
  return {
    id: '',
    name: '',
    type: 'snmp_v2c',
    description: '',
    datacenter_id: '',
    snmp_community: '',
    snmp_v3_user: '',
    snmp_v3_auth: '',
    snmp_v3_priv: '',
    ssh_username: '',
    ssh_key_id: '',
  };
}
