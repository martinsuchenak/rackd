import type { Network, ScanProfile, ScheduledScan, ScheduledScanInput } from '../core/types';
import { api, RackdAPIError } from '../core/api';

type ModalType = '' | 'form' | 'delete';

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
    profiles: [] as ScanProfile[],
    loading: true,
    error: '',
    modalType: '' as ModalType,
    deleteTarget: null as ScheduledScan | null,
    form: resetForm(),

    get showFormModal(): boolean {
      return this.modalType === 'form';
    },

    get showDeleteModal(): boolean {
      return this.modalType === 'delete';
    },

    get showEmptyState(): boolean {
      return !this.loading && Array.isArray(this.scans) && this.scans.length === 0;
    },

    async init(): Promise<void> {
      await this.load();
    },

    async load(): Promise<void> {
      this.loading = true;
      this.error = '';
      try {
        const [scans, networks, profiles] = await Promise.all([
          api.listScheduledScans(),
          api.listNetworks(),
          api.listScanProfiles(),
        ]);
        this.scans = scans;
        this.networks = networks;
        this.profiles = profiles;
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load scheduled scans';
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

    openEditModal(scan: ScheduledScan): void {
      this.modalType = '';
      this.form = {
        id: scan.id,
        network_id: scan.network_id,
        profile_id: scan.profile_id,
        name: scan.name,
        enabled: scan.enabled,
        cron_expression: scan.cron_expression,
        description: scan.description || '',
      };
      this.error = '';
      this.modalType = 'form';
    },

    closeModal(): void {
      this.modalType = '';
      this.form = resetForm();
      this.deleteTarget = null;
      this.error = '';
    },

    async save(): Promise<void> {
      this.error = '';
      const payload: ScheduledScanInput = {
        network_id: this.form.network_id,
        profile_id: this.form.profile_id,
        name: this.form.name,
        enabled: this.form.enabled,
        cron_expression: this.form.cron_expression,
        description: this.form.description || undefined,
      };

      try {
        if (this.form.id) {
          await api.updateScheduledScan(this.form.id, payload);
        } else {
          await api.createScheduledScan(payload);
        }
        this.closeModal();
        await this.load();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to save scheduled scan';
      }
    },

    async toggleEnabled(scan: ScheduledScan): Promise<void> {
      try {
        await api.updateScheduledScan(scan.id, {
          network_id: scan.network_id,
          profile_id: scan.profile_id,
          name: scan.name,
          enabled: !scan.enabled,
          cron_expression: scan.cron_expression,
          description: scan.description,
        });
        scan.enabled = !scan.enabled;
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to update scan';
      }
    },

    openDeleteModal(scan: ScheduledScan): void {
      this.modalType = '';
      this.deleteTarget = scan;
      this.modalType = 'delete';
    },

    async deleteConfirmed(): Promise<void> {
      if (!this.deleteTarget) return;
      try {
        await api.deleteScheduledScan(this.deleteTarget.id);
        this.closeModal();
        await this.load();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete scheduled scan';
      }
    },

    getNetworkName(id: string): string {
      const network = this.networks.find((item) => item.id === id);
      return network ? `${network.name} (${network.subnet})` : id;
    },

    getProfileName(id: string): string {
      const profile = this.profiles.find((item) => item.id === id);
      return profile ? profile.name : id;
    },

    formatDate(dateStr: string | undefined): string {
      return dateStr ? new Date(dateStr).toLocaleString() : '-';
    },

    getDeleteTargetName(): string {
      return this.deleteTarget?.name || '';
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
