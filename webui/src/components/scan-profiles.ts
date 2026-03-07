// Scan Profiles Components for Rackd Web UI

import type { ScanProfile, Network } from '../core/types';
import { api, RackdAPIError } from '../core/api';

interface ScanProfilesData {
  profiles: ScanProfile[];
  networks: Network[];
  showCreateModal: boolean;
  showEditModal: boolean;
  editingProfile: ScanProfile | null;
  loading: boolean;
  error: string;
  init(): Promise<void>;
  loadProfiles(): Promise<void>;
  loadNetworks(): Promise<void>;
  openCreateModal(): void;
  openEditModal(profile: ScanProfile): void;
  createProfile(): Promise<void>;
  updateProfile(): Promise<void>;
  deleteProfile(id: string): Promise<void>;
  getPortsString(profile: ScanProfile): string;
}

export function scanProfilesList() {
  return {
    profiles: [] as ScanProfile[],
    networks: [] as Network[],
    showCreateModal: false,
    showEditModal: false,
    editingProfile: null as ScanProfile | null,
    loading: true,
    error: '',
    // Form fields
    formName: '',
    formScanType: 'quick' as ScanProfile['scan_type'],
    formPorts: '',
    formEnableSNMP: false,
    formEnableSSH: false,
    formTimeoutSec: 5,
    formMaxWorkers: 10,
    formDescription: '',

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
      this.formName = '';
      this.formScanType = 'quick';
      this.formPorts = '';
      this.formEnableSNMP = false;
      this.formEnableSSH = false;
      this.formTimeoutSec = 5;
      this.formMaxWorkers = 10;
      this.formDescription = '';
      this.showCreateModal = true;
    },

    openEditModal(profile: ScanProfile): void {
      this.editingProfile = profile;
      this.formName = profile.name;
      this.formScanType = profile.scan_type;
      this.formPorts = profile.ports?.join(', ') || '';
      this.formEnableSNMP = profile.enable_snmp || false;
      this.formEnableSSH = profile.enable_ssh || false;
      this.formTimeoutSec = profile.timeout_sec || 5;
      this.formMaxWorkers = profile.max_workers || 10;
      this.formDescription = profile.description || '';
      this.showEditModal = true;
    },

    async createProfile(): Promise<void> {
      if (!this.formName.trim()) {
        this.error = 'Profile name is required';
        return;
      }

      this.error = '';
      this.loading = true;

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
        this.showCreateModal = false;
        await this.loadProfiles();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to create scan profile';
      } finally {
        this.loading = false;
      }
    },

    async updateProfile(): Promise<void> {
      if (!this.editingProfile || !this.formName.trim()) {
        return;
      }

      this.error = '';
      this.loading = true;

      try {
        const ports = this.formScanType === 'custom' && this.formPorts
          ? this.formPorts.split(',').map(p => parseInt(p.trim())).filter(p => !isNaN(p))
          : undefined;

        const profile: Partial<import('../core/types').ScanProfile> = {
          ...this.editingProfile,
          name: this.formName.trim(),
          scan_type: this.formScanType,
          ports,
          enable_snmp: this.formEnableSNMP,
          enable_ssh: this.formEnableSSH,
          timeout_sec: this.formTimeoutSec,
          max_workers: this.formMaxWorkers,
          description: this.formDescription.trim() || undefined,
        };

        await api.updateScanProfile(this.editingProfile.id, profile);
        this.showEditModal = false;
        this.editingProfile = null;
        await this.loadProfiles();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to update scan profile';
      } finally {
        this.loading = false;
      }
    },

    closeEditModal(): void {
      this.showEditModal = false;
      this.editingProfile = null;
    },

    async deleteProfile(id: string): Promise<void> {
      if (!confirm('Delete this scan profile?')) return;
      this.error = '';
      try {
        await api.deleteScanProfile(id);
        await this.loadProfiles();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete scan profile';
      }
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
