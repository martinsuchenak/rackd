// Credentials Management Components

export interface Credential {
  id: string;
  name: string;
  type: string;
  description?: string;
  datacenter_id?: string;
  has_community?: boolean;
  has_auth?: boolean;
  has_username?: boolean;
  created_at: string;
  updated_at: string;
}

interface CredentialFormData {
  id: string;
  name: string;
  type: string;
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
    showModal: false,
    showDeleteModal: false,
    deleteTarget: null as Credential | null,
    form: resetForm(),

    async init() {
      await this.load();
    },

    async load() {
      this.loading = true;
      this.error = '';
      try {
        const response = await fetch('/api/credentials', {
          credentials: 'same-origin',
          headers: { 'X-Requested-With': 'XMLHttpRequest' },
        });
        if (response.ok) {
          this.credentials = (await response.json()) || [];
        } else {
          this.error = 'Failed to load credentials';
        }
      } catch {
        this.error = 'Network error';
      } finally {
        this.loading = false;
      }
    },

    openAddModal() {
      this.form = resetForm();
      this.showModal = true;
    },

    openEditModal(cred: Credential) {
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
        ssh_username: '',
        ssh_key_id: '',
      };
      this.showModal = true;
    },

    closeModal() {
      this.showModal = false;
      this.form = resetForm();
      this.error = '';
    },

    async save() {
      this.error = '';
      try {
        const isEdit = !!this.form.id;
        const url = isEdit ? `/api/credentials/${this.form.id}` : '/api/credentials';
        const response = await fetch(url, {
          method: isEdit ? 'PUT' : 'POST',
          headers: { 'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest' },
          credentials: 'same-origin',
          body: JSON.stringify(this.form),
        });

        if (response.ok) {
          this.closeModal();
          await this.load();
        } else {
          const data = await response.json();
          this.error = data.error || 'Failed to save credential';
        }
      } catch {
        this.error = 'Network error';
      }
    },

    confirmDelete(cred: Credential) {
      this.deleteTarget = cred;
      this.showDeleteModal = true;
    },

    async deleteConfirmed() {
      if (!this.deleteTarget) return;
      try {
        const response = await fetch(`/api/credentials/${this.deleteTarget.id}`, {
          method: 'DELETE',
          headers: { 'X-Requested-With': 'XMLHttpRequest' },
          credentials: 'same-origin',
        });
        if (response.ok) {
          this.showDeleteModal = false;
          this.deleteTarget = null;
          await this.load();
        } else {
          this.error = 'Failed to delete credential';
        }
      } catch {
        this.error = 'Network error';
      }
    },

    cancelDelete() {
      this.showDeleteModal = false;
      this.deleteTarget = null;
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
      return this.deleteTarget ? this.deleteTarget.name : '';
    }
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
