// Datacenter Components for Rackd Web UI

import type { Datacenter, Device } from '../core/types';
import { api, RackdAPIError } from '../core/api';
import { debounce } from '../core/utils';

interface DatacenterListData {
  datacenters: Datacenter[];
  allDatacenters: Datacenter[];
  loading: boolean;
  error: string;
  search: string;
  showDeleteModal: boolean;
  deleteTarget: Datacenter | null;
  deleting: boolean;
  showModal: boolean;
  isEditMode: boolean;
  editDatacenter: Partial<Datacenter>;
  saving: boolean;
  deleteModalTitle: string;
  deleteModalName: string;
  init(): Promise<void>;
  loadDatacenters(): Promise<void>;
  filterDatacenters(): void;
  applySearch(): void;
  clearFilters(): void;
  confirmDelete(dc: Datacenter): void;
  cancelDelete(): void;
  doDelete(): Promise<void>;
  openAddModal(): void;
  openEditModal(dc: Datacenter): void;
  closeModal(): void;
  saveDatacenter(): Promise<void>;
}

export function datacenterList() {
  return {
    datacenters: [] as Datacenter[],
    allDatacenters: [] as Datacenter[],
    loading: true,
    error: '',
    search: '',
    showDeleteModal: false,
    deleteTarget: null as Datacenter | null,
    deleting: false,
    // Unified add/edit modal
    showModal: false,
    isEditMode: false,
    editDatacenter: {} as Partial<Datacenter>,
    saving: false,

    get deleteModalTitle(): string {
      return 'Delete Datacenter';
    },

    get deleteModalName(): string {
      return this.deleteTarget?.name || '';
    },

    openAddModal(): void {
      this.isEditMode = false;
      this.editDatacenter = { name: '', location: '', description: '' };
      this.showModal = true;
      setTimeout(() => {
        (document.querySelector('[x-show="showModal"] input[type="text"]') as HTMLInputElement)?.focus();
      }, 50);
    },

    openEditModal(dc: Datacenter): void {
      this.isEditMode = true;
      this.editDatacenter = {
        id: dc.id,
        name: dc.name,
        location: dc.location || '',
        description: dc.description || '',
      };
      this.showModal = true;
      setTimeout(() => {
        (document.querySelector('[x-show="showModal"] input[type="text"]') as HTMLInputElement)?.focus();
      }, 50);
    },

    closeModal(): void {
      this.showModal = false;
    },

    async init(): Promise<void> {
      await this.loadDatacenters();
    },

    async loadDatacenters(): Promise<void> {
      this.loading = true;
      this.error = '';
      try {
        this.allDatacenters = (await api.listDatacenters()) || [];
        this.filterDatacenters();
      } catch (e) {
        this.allDatacenters = [];
        this.datacenters = [];
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load datacenters';
      } finally {
        this.loading = false;
      }
    },

    filterDatacenters(): void {
      if (!this.search.trim()) {
        this.datacenters = this.allDatacenters;
        return;
      }
      const q = this.search.trim().toLowerCase();
      this.datacenters = this.allDatacenters.filter((dc) => {
        const searchStr = [dc.name || '', dc.location || '', dc.description || ''].join(' ').toLowerCase();
        return searchStr.includes(q);
      });
    },

    applySearch: debounce(function (this: DatacenterListData) {
      this.filterDatacenters();
    }, 300),

    clearFilters(): void {
      this.search = '';
      this.loadDatacenters();
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

    async saveDatacenter(): Promise<void> {
      this.saving = true;
      this.error = '';
      try {
        if (this.isEditMode && this.editDatacenter.id) {
          await api.updateDatacenter(this.editDatacenter.id, this.editDatacenter);
        } else {
          await api.createDatacenter(this.editDatacenter);
        }
        this.showModal = false;
        await this.loadDatacenters();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : (this.isEditMode ? 'Failed to update datacenter' : 'Failed to create datacenter');
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
  deleteModalTitle: string;
  deleteModalName: string;
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

    get deleteModalTitle(): string {
      return 'Delete Datacenter';
    },

    get deleteModalName(): string {
      return this.datacenter?.name || '';
    },

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
