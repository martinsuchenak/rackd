// Scheduled Scans Management Components

export interface ScheduledScan {
  id: string;
  network_id: string;
  profile_id: string;
  name: string;
  enabled: boolean;
  cron_expression: string;
  description?: string;
  last_run_at?: string;
  next_run_at?: string;
  created_at: string;
  updated_at: string;
}

interface Network {
  id: string;
  name: string;
  subnet: string;
}

interface Profile {
  id: string;
  name: string;
}

interface ScheduledScanFormData {
  id: string;
  network_id: string;
  profile_id: string;
  name: string;
  enabled: boolean;
  cron_expression: string;
  description: string;
}

export function scheduledScansList() {
  return {
    scans: [] as ScheduledScan[],
    networks: [] as Network[],
    profiles: [] as Profile[],
    loading: true,
    error: '',
    showModal: false,
    showDeleteModal: false,
    deleteTarget: null as ScheduledScan | null,
    form: resetForm(),

    async init() {
      await this.load();
    },

    async load() {
      this.loading = true;
      this.error = '';
      try {
        const [scansRes, networksRes, profilesRes] = await Promise.all([
          fetch('/api/scheduled-scans'),
          fetch('/api/networks'),
          fetch('/api/scan-profiles'),
        ]);

        if (scansRes.ok) this.scans = (await scansRes.json()) || [];
        if (networksRes.ok) this.networks = (await networksRes.json()) || [];
        if (profilesRes.ok) this.profiles = (await profilesRes.json()) || [];
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

    openEditModal(scan: ScheduledScan) {
      this.form = {
        id: scan.id,
        network_id: scan.network_id,
        profile_id: scan.profile_id,
        name: scan.name,
        enabled: scan.enabled,
        cron_expression: scan.cron_expression,
        description: scan.description || '',
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
        const url = isEdit ? `/api/scheduled-scans/${this.form.id}` : '/api/scheduled-scans';
        const response = await fetch(url, {
          method: isEdit ? 'PUT' : 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(this.form),
        });

        if (response.ok) {
          this.closeModal();
          await this.load();
        } else {
          const data = await response.json();
          this.error = data.error || 'Failed to save scheduled scan';
        }
      } catch {
        this.error = 'Network error';
      }
    },

    async toggleEnabled(scan: ScheduledScan) {
      try {
        const updated = { ...scan, enabled: !scan.enabled };
        const response = await fetch(`/api/scheduled-scans/${scan.id}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(updated),
        });

        if (response.ok) {
          scan.enabled = !scan.enabled;
        } else {
          this.error = 'Failed to update scan';
        }
      } catch {
        this.error = 'Network error';
      }
    },

    confirmDelete(scan: ScheduledScan) {
      this.deleteTarget = scan;
      this.showDeleteModal = true;
    },

    async deleteConfirmed() {
      if (!this.deleteTarget) return;
      try {
        const response = await fetch(`/api/scheduled-scans/${this.deleteTarget.id}`, {
          method: 'DELETE',
        });
        if (response.ok) {
          this.showDeleteModal = false;
          this.deleteTarget = null;
          await this.load();
        } else {
          this.error = 'Failed to delete scheduled scan';
        }
      } catch {
        this.error = 'Network error';
      }
    },

    cancelDelete() {
      this.showDeleteModal = false;
      this.deleteTarget = null;
    },

    getNetworkName(id: string): string {
      const net = this.networks.find((n: Network) => n.id === id);
      return net ? `${net.name} (${net.subnet})` : id;
    },

    getProfileName(id: string): string {
      const profile = this.profiles.find((p: Profile) => p.id === id);
      return profile ? profile.name : id;
    },

    formatDate(dateStr: string | undefined): string {
      if (!dateStr) return '-';
      return new Date(dateStr).toLocaleString();
    },
  };
}

function resetForm(): ScheduledScanFormData {
  return {
    id: '',
    network_id: '',
    profile_id: '',
    name: '',
    enabled: true,
    cron_expression: '0 2 * * *',
    description: '',
  };
}
