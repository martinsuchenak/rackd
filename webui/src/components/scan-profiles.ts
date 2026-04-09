// Scan Profiles Components for Rackd Web UI

import type { ScanProfile, Network } from '../core/types';
import { api, RackdAPIError } from '../core/api';
import type { ListPageState, ValidationErrors } from '../core/page-state';

type ModalType = '' | 'create' | 'edit' | 'delete';

interface ScanProfilesData extends ListPageState<ScanProfile, Exclude<ModalType, ''>> {
  profiles: ScanProfile[];
  selectedProfile: ScanProfile | null;
  networks: Network[];
  get showCreateModal(): boolean;
  get showEditModal(): boolean;
  get showDeleteModal(): boolean;
  get items(): ScanProfile[];
  get selectedItem(): ScanProfile | null;
  get deleteModalTitle(): string;
  get deleteModalName(): string;
  get deleteModalDescription(): string;
  init(): Promise<void>;
  loadProfiles(): Promise<void>;
  loadNetworks(): Promise<void>;
  openCreateModal(): void;
  openEditModal(profile: ScanProfile): void;
  openDeleteModal(profile: ScanProfile): void;
  closeModal(): void;
  createProfile(): Promise<void>;
  updateProfile(): Promise<void>;
  save(): Promise<void>;
  deleteProfile(id: string): Promise<void>;
  deleteConfirmed(): Promise<void>;
  getPortsString(profile: ScanProfile): string;
}

export function scanProfilesList() {
  return {
    profiles: [] as ScanProfile[],
    selectedProfile: null as ScanProfile | null,
    networks: [] as Network[],
    modalType: '' as ModalType,
    loading: true,
    saving: false,
    deleting: false,
    error: '',
    validationErrors: {} as ValidationErrors,
    // Form fields
    formName: '',
    formScanType: 'quick' as ScanProfile['scan_type'],
    formPorts: '',
    formEnableSNMP: false,
    formEnableSSH: false,
    formTimeoutSec: 5,
    formMaxWorkers: 10,
    formDescription: '',

    get showCreateModal(): boolean { return this.modalType === 'create'; },
    get showEditModal(): boolean { return this.modalType === 'edit'; },
    get showDeleteModal(): boolean { return this.modalType === 'delete'; },
    get items(): ScanProfile[] { return this.profiles; },
    get selectedItem(): ScanProfile | null { return this.selectedProfile; },
    get deleteModalTitle(): string { return 'Delete Scan Profile'; },
    get deleteModalName(): string { return this.selectedProfile?.name || ''; },
    get deleteModalDescription(): string {
      return `Are you sure you want to delete ${this.deleteModalName}? This action cannot be undone.`;
    },

    async init(): Promise<void> {
      await Promise.all([this.loadProfiles(), this.loadNetworks()]);
      this.loading = false;
    },

    async loadProfiles(): Promise<void> {
      try {
        this.profiles = (await api.listScanProfiles()) || [];
      } catch (e) {
        this.profiles = [];
        this.error = e instanceof Error ? e.message : 'Failed to load scan profiles';
      }
    },

    async loadNetworks(): Promise<void> {
      try {
        this.networks = (await api.listNetworks()) || [];
      } catch {
        this.networks = [];
      }
    },

    openCreateModal(): void {
      this.modalType = '';
      this.selectedProfile = null;
      this.validationErrors = {};
      this.formName = '';
      this.formScanType = 'quick';
      this.formPorts = '';
      this.formEnableSNMP = false;
      this.formEnableSSH = false;
      this.formTimeoutSec = 5;
      this.formMaxWorkers = 10;
      this.formDescription = '';
      this.modalType = 'create';
    },

    closeCreateModal(): void {
      this.closeModal();
    },

    openEditModal(profile: ScanProfile): void {
      this.modalType = '';
      this.selectedProfile = profile;
      this.validationErrors = {};
      this.formName = profile.name;
      this.formScanType = profile.scan_type;
      this.formPorts = profile.ports?.join(', ') || '';
      this.formEnableSNMP = profile.enable_snmp || false;
      this.formEnableSSH = profile.enable_ssh || false;
      this.formTimeoutSec = profile.timeout_sec || 5;
      this.formMaxWorkers = profile.max_workers || 10;
      this.formDescription = profile.description || '';
      this.modalType = 'edit';
    },

    openDeleteModal(profile: ScanProfile): void {
      this.modalType = '';
      this.selectedProfile = profile;
      this.validationErrors = {};
      this.modalType = 'delete';
    },

    closeModal(): void {
      this.modalType = '';
      this.selectedProfile = null;
      this.validationErrors = {};
    },

    async createProfile(): Promise<void> {
      if (!this.formName.trim()) {
        this.error = 'Profile name is required';
        return;
      }

      this.error = '';
      this.saving = true;

      try {
        const ports = this.formScanType === 'custom' && this.formPorts
          ? this.formPorts.split(',').map(p => parseInt(p.trim())).filter(p => !isNaN(p))
          : [];

        const profile: Partial<import('../core/types').ScanProfile> = {
          name: this.formName.trim(),
          scan_type: this.formScanType,
          ports: ports.length > 0 ? ports : undefined,
          enable_snmp: this.formEnableSNMP,
          enable_ssh: this.formEnableSSH,
          timeout_sec: this.formTimeoutSec,
          max_workers: this.formMaxWorkers,
          description: this.formDescription.trim() || undefined,
        };

        await api.createScanProfile(profile);
        this.closeModal();
        await this.loadProfiles();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to create scan profile';
      } finally {
        this.saving = false;
      }
    },

    async updateProfile(): Promise<void> {
      if (!this.selectedProfile || !this.formName.trim()) {
        return;
      }

      this.error = '';
      this.saving = true;

      try {
        const ports = this.formScanType === 'custom' && this.formPorts
          ? this.formPorts.split(',').map(p => parseInt(p.trim())).filter(p => !isNaN(p))
          : undefined;

        const profile: Partial<import('../core/types').ScanProfile> = {
          ...this.selectedProfile,
          name: this.formName.trim(),
          scan_type: this.formScanType,
          ports,
          enable_snmp: this.formEnableSNMP,
          enable_ssh: this.formEnableSSH,
          timeout_sec: this.formTimeoutSec,
          max_workers: this.formMaxWorkers,
          description: this.formDescription.trim() || undefined,
        };

        await api.updateScanProfile(this.selectedProfile.id, profile);
        this.closeModal();
        await this.loadProfiles();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to update scan profile';
      } finally {
        this.saving = false;
      }
    },

    closeEditModal(): void {
      this.closeModal();
    },

    async deleteProfile(id: string): Promise<void> {
      const profile = this.profiles.find((item) => item.id === id);
      if (!profile) return;
      this.openDeleteModal(profile);
    },

    async deleteConfirmed(): Promise<void> {
      if (!this.selectedProfile) return;
      this.error = '';
      this.deleting = true;
      try {
        await api.deleteScanProfile(this.selectedProfile.id);
        this.closeModal();
        await this.loadProfiles();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete scan profile';
      } finally {
        this.deleting = false;
      }
    },

    async save(): Promise<void> {
      if (this.modalType === 'edit') {
        await this.updateProfile();
        return;
      }
      await this.createProfile();
    },

    getPortsString(profile: ScanProfile): string {
      if (!profile.ports || profile.ports.length === 0) {
        return profile.scan_type === 'custom' ? 'Custom (no ports specified)' : 'Default ports';
      }
      if (profile.ports.length <= 3) {
        return profile.ports.join(', ');
      }
      return `${profile.ports.slice(0, 3).join(', ')} +${profile.ports.length - 3} more`;
    },
  };
}
