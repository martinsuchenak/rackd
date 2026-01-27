// Datacenter Components for Rackd Web UI

import type { Datacenter, Device } from '../core/types';
import { RackdAPI, RackdAPIError } from '../core/api';

const api = new RackdAPI();

interface DatacenterListData {
  datacenters: Datacenter[];
  loading: boolean;
  error: string;
  showDeleteModal: boolean;
  deleteTarget: Datacenter | null;
  deleting: boolean;
  init(): Promise<void>;
  loadDatacenters(): Promise<void>;
  confirmDelete(dc: Datacenter): void;
  cancelDelete(): void;
  doDelete(): Promise<void>;
}

export function datacenterList() {
  return {
    datacenters: [] as Datacenter[],
    loading: true,
    error: '',
    showDeleteModal: false,
    deleteTarget: null as Datacenter | null,
    deleting: false,
    showAddModal: false,
    newDatacenter: { name: '', location: '', description: '' } as Partial<Datacenter>,
    saving: false,

    openAddModal(): void {
      this.newDatacenter = { name: '', location: '', description: '' };
      this.showAddModal = true;
      setTimeout(() => {
        (document.querySelector('[x-show="showAddModal"] input[type="text"]') as HTMLInputElement)?.focus();
      }, 50);
    },

    async init(): Promise<void> {
      await this.loadDatacenters();
    },

    async loadDatacenters(): Promise<void> {
      this.loading = true;
      this.error = '';
      try {
        this.datacenters = (await api.listDatacenters()) || [];
      } catch (e) {
        this.datacenters = [];
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load datacenters';
      } finally {
        this.loading = false;
      }
    },

    confirmDelete(dc: Datacenter): void {
      this.deleteTarget = dc;
      this.showDeleteModal = true;
      setTimeout(() => {
        const modal = document.querySelector('[x-show="showDeleteModal"]');
        if (modal) {
          const cancelBtn = modal.querySelector('button[type="button"]') as HTMLButtonElement;
          cancelBtn?.focus();
        }
      }, 50);
    },

    cancelDelete(): void {
      this.showDeleteModal = false;
      this.deleteTarget = null;
    },

    async doDelete(): Promise<void> {
      if (!this.deleteTarget) return;
      this.deleting = true;
      try {
        await api.deleteDatacenter(this.deleteTarget.id);
        this.showDeleteModal = false;
        this.deleteTarget = null;
        await this.loadDatacenters();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete datacenter';
      } finally {
        this.deleting = false;
      }
    },

    async saveNew(): Promise<void> {
      this.saving = true;
      this.error = '';
      try {
        await api.createDatacenter(this.newDatacenter);
        this.showAddModal = false;
        this.newDatacenter = { name: '', location: '', description: '' };
        await this.loadDatacenters();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to create datacenter';
      } finally {
        this.saving = false;
      }
    },
  };
}

interface DatacenterDetailData {
  datacenter: Datacenter | null;
  devices: Device[];
  loading: boolean;
  error: string;
  showDeleteModal: boolean;
  deleting: boolean;
  showEditModal: boolean;
  editDatacenter: Partial<Datacenter>;
  saving: boolean;
  init(): Promise<void>;
  loadDatacenter(): Promise<void>;
  loadDevices(): Promise<void>;
  confirmDelete(): void;
  cancelDelete(): void;
  doDelete(): Promise<void>;
  openEditModal(): void;
  closeEditModal(): void;
  saveEdit(): Promise<void>;
}

export function datacenterDetail(): DatacenterDetailData {
  return {
    datacenter: null,
    devices: [],
    loading: true,
    error: '',
    showDeleteModal: false,
    deleting: false,
    showEditModal: false,
    editDatacenter: {},
    saving: false,

    async init(): Promise<void> {
      // Wait for next tick to ensure URL is updated after SPA navigation
      await new Promise((resolve) => setTimeout(resolve, 0));
      const id = new URLSearchParams(window.location.search).get('id');
      if (!id) {
        this.error = 'No datacenter ID provided';
        this.loading = false;
        return;
      }
      await this.loadDatacenter();
    },

    async loadDatacenter(): Promise<void> {
      const id = new URLSearchParams(window.location.search).get('id');
      if (!id) return;
      this.loading = true;
      try {
        this.datacenter = await api.getDatacenter(id);
        await this.loadDevices();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load datacenter';
      } finally {
        this.loading = false;
      }
    },

    async loadDevices(): Promise<void> {
      if (!this.datacenter) return;
      try {
        this.devices = (await api.getDatacenterDevices(this.datacenter.id)) || [];
      } catch {
        this.devices = [];
      }
    },

    confirmDelete(): void {
      this.showDeleteModal = true;
      setTimeout(() => {
        const modal = document.querySelector('[x-show="showDeleteModal"]');
        if (modal) {
          const cancelBtn = modal.querySelector('button[type="button"]') as HTMLButtonElement;
          cancelBtn?.focus();
        }
      }, 50);
    },

    cancelDelete(): void {
      this.showDeleteModal = false;
    },

    async doDelete(): Promise<void> {
      if (!this.datacenter) return;
      this.deleting = true;
      try {
        await api.deleteDatacenter(this.datacenter.id);
        window.location.href = '/datacenters';
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete datacenter';
        this.deleting = false;
      }
    },

    openEditModal(): void {
      if (!this.datacenter) return;
      this.editDatacenter = {
        id: this.datacenter.id,
        name: this.datacenter.name,
        location: this.datacenter.location || '',
        description: this.datacenter.description || '',
      };
      this.showEditModal = true;
      setTimeout(() => {
        (document.querySelector('[x-show="showEditModal"] input[type="text"]') as HTMLInputElement)?.focus();
      }, 50);
    },

    closeEditModal(): void {
      this.showEditModal = false;
      this.editDatacenter = {};
    },

    async saveEdit(): Promise<void> {
      if (!this.datacenter || !this.editDatacenter.id) return;
      this.saving = true;
      try {
        await api.updateDatacenter(this.editDatacenter.id, this.editDatacenter);
        this.showEditModal = false;
        this.editDatacenter = {};
        await this.loadDatacenter();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to update datacenter';
      } finally {
        this.saving = false;
      }
    },
  };
}

interface DatacenterFormData {
  datacenter: Partial<Datacenter>;
  isEdit: boolean;
  loading: boolean;
  saving: boolean;
  error: string;
  init(): Promise<void>;
  save(): Promise<void>;
  cancel(): void;
}

export function datacenterForm(): DatacenterFormData {
  return {
    datacenter: {},
    isEdit: false,
    loading: true,
    saving: false,
    error: '',

    async init(): Promise<void> {
      const id = new URLSearchParams(window.location.search).get('id');
      this.isEdit = !!id;
      try {
        if (id) {
          this.datacenter = await api.getDatacenter(id);
        }
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load datacenter';
      } finally {
        this.loading = false;
      }
    },

    async save(): Promise<void> {
      this.saving = true;
      this.error = '';
      try {
        if (this.isEdit && this.datacenter.id) {
          await api.updateDatacenter(this.datacenter.id, this.datacenter);
        } else {
          await api.createDatacenter(this.datacenter);
        }
        window.location.href = '/datacenters';
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to save datacenter';
      } finally {
        this.saving = false;
      }
    },

    cancel(): void {
      window.location.href = '/datacenters';
    },
  };
}
