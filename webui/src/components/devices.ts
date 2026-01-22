// Device Components for Rackd Web UI

import type { Datacenter, Device, DeviceFilter, DeviceRelationship } from '../core/types';
import { RackdAPI, RackdAPIError } from '../core/api';
import { debounce, formatDate } from '../core/utils';

const api = new RackdAPI();

interface DeviceListData {
  devices: Device[];
  datacenters: Datacenter[];
  loading: boolean;
  error: string;
  search: string;
  filter: DeviceFilter;
  page: number;
  pageSize: number;
  showDeleteModal: boolean;
  deleteTarget: Device | null;
  deleting: boolean;
  totalPages: number;
  pagedDevices: Device[];
  init(): Promise<void>;
  loadDevices(): Promise<void>;
  loadDatacenters(): Promise<void>;
  applySearch(): void;
  setFilter(key: keyof DeviceFilter, value: string): void;
  clearFilters(): void;
  goToPage(p: number): void;
  confirmDelete(device: Device): void;
  cancelDelete(): void;
  doDelete(): Promise<void>;
}

export function deviceList() {
  return {
    devices: [] as Device[],
    datacenters: [] as Datacenter[],
    loading: true,
    error: '',
    search: '',
    filter: {} as DeviceFilter,
    page: 1,
    pageSize: 10,
    showDeleteModal: false,
    deleteTarget: null as Device | null,
    deleting: false,
    // Add modal
    showAddModal: false,
    newDevice: { name: '', make_model: '', description: '', datacenter_id: '', os: '' } as Partial<Device>,
    saving: false,

    get totalPages(): number {
      return Math.ceil(this.devices.length / this.pageSize) || 1;
    },

    get pagedDevices(): Device[] {
      const start = (this.page - 1) * this.pageSize;
      return this.devices.slice(start, start + this.pageSize);
    },

    async init(): Promise<void> {
      await Promise.all([this.loadDevices(), this.loadDatacenters()]);
    },

    async loadDatacenters(): Promise<void> {
      try {
        this.datacenters = await api.listDatacenters();
      } catch {
        // Non-critical
      }
    },

    async loadDevices(): Promise<void> {
      this.loading = true;
      this.error = '';
      try {
        if (this.search) {
          this.devices = await api.searchDevices(this.search);
        } else {
          this.devices = await api.listDevices(this.filter);
        }
        this.page = 1;
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load devices';
      } finally {
        this.loading = false;
      }
    },

    applySearch: debounce(function (this: DeviceListData) {
      this.loadDevices();
    }, 300),

    setFilter(key: keyof DeviceFilter, value: string): void {
      if (value) {
        if (key === 'tags') {
          this.filter.tags = value.split(',');
        } else {
          this.filter[key] = value;
        }
      } else {
        delete this.filter[key];
      }
      this.loadDevices();
    },

    clearFilters(): void {
      this.filter = {};
      this.search = '';
      this.loadDevices();
    },

    goToPage(p: number): void {
      if (p >= 1 && p <= this.totalPages) this.page = p;
    },

    confirmDelete(device: Device): void {
      this.deleteTarget = device;
      this.showDeleteModal = true;
    },

    cancelDelete(): void {
      this.showDeleteModal = false;
      this.deleteTarget = null;
    },

    async doDelete(): Promise<void> {
      if (!this.deleteTarget) return;
      this.deleting = true;
      try {
        await api.deleteDevice(this.deleteTarget.id);
        this.showDeleteModal = false;
        this.deleteTarget = null;
        await this.loadDevices();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete device';
      } finally {
        this.deleting = false;
      }
    },

    async saveNew(): Promise<void> {
      this.saving = true;
      this.error = '';
      try {
        await api.createDevice(this.newDevice);
        this.showAddModal = false;
        this.newDevice = { name: '', make_model: '', description: '', datacenter_id: '', os: '' };
        await this.loadDevices();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to create device';
      } finally {
        this.saving = false;
      }
    },
  };
}

interface DeviceDetailData {
  device: Device | null;
  datacenters: Datacenter[];
  relationships: DeviceRelationship[];
  relatedDevices: Map<string, Device>;
  loading: boolean;
  error: string;
  activeTab: 'details' | 'addresses' | 'relationships';
  showDeleteModal: boolean;
  deleting: boolean;
  init(): Promise<void>;
  loadDevice(): Promise<void>;
  loadDatacenters(): Promise<void>;
  loadRelationships(): Promise<void>;
  setTab(tab: 'details' | 'addresses' | 'relationships'): void;
  getDatacenterName(id?: string): string;
  confirmDelete(): void;
  cancelDelete(): void;
  doDelete(): Promise<void>;
}

export function deviceDetail(): DeviceDetailData {
  return {
    device: null,
    datacenters: [],
    relationships: [],
    relatedDevices: new Map(),
    loading: true,
    error: '',
    activeTab: 'details',
    showDeleteModal: false,
    deleting: false,

    async init(): Promise<void> {
      const id = new URLSearchParams(window.location.search).get('id');
      if (!id) {
        this.error = 'No device ID provided';
        this.loading = false;
        return;
      }
      await Promise.all([this.loadDevice(), this.loadDatacenters()]);
    },

    async loadDatacenters(): Promise<void> {
      try {
        this.datacenters = await api.listDatacenters();
      } catch {
        // Non-critical
      }
    },

    async loadDevice(): Promise<void> {
      const id = new URLSearchParams(window.location.search).get('id');
      if (!id) return;
      this.loading = true;
      try {
        this.device = await api.getDevice(id);
        await this.loadRelationships();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load device';
      } finally {
        this.loading = false;
      }
    },

    async loadRelationships(): Promise<void> {
      if (!this.device) return;
      try {
        this.relationships = await api.getRelationships(this.device.id);
        for (const rel of this.relationships) {
          const otherId = rel.parent_id === this.device.id ? rel.child_id : rel.parent_id;
          if (!this.relatedDevices.has(otherId)) {
            const d = await api.getDevice(otherId);
            this.relatedDevices.set(otherId, d);
          }
        }
      } catch {
        // Non-critical
      }
    },

    setTab(tab: 'details' | 'addresses' | 'relationships'): void {
      this.activeTab = tab;
    },

    getDatacenterName(id?: string): string {
      if (!id) return '-';
      return this.datacenters.find((d) => d.id === id)?.name ?? id;
    },

    confirmDelete(): void {
      this.showDeleteModal = true;
    },

    cancelDelete(): void {
      this.showDeleteModal = false;
    },

    async doDelete(): Promise<void> {
      if (!this.device) return;
      this.deleting = true;
      try {
        await api.deleteDevice(this.device.id);
        window.location.href = '/devices';
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete device';
        this.deleting = false;
      }
    },
  };
}

interface DeviceFormData {
  device: Partial<Device>;
  datacenters: Datacenter[];
  isEdit: boolean;
  loading: boolean;
  saving: boolean;
  error: string;
  tagInput: string;
  init(): Promise<void>;
  addTag(): void;
  removeTag(tag: string): void;
  addAddress(): void;
  removeAddress(index: number): void;
  save(): Promise<void>;
  cancel(): void;
}

export function deviceForm(): DeviceFormData {
  return {
    device: { tags: [], addresses: [], domains: [] },
    datacenters: [],
    isEdit: false,
    loading: true,
    saving: false,
    error: '',
    tagInput: '',

    async init(): Promise<void> {
      const id = new URLSearchParams(window.location.search).get('id');
      this.isEdit = !!id;
      try {
        this.datacenters = await api.listDatacenters();
        if (id) {
          this.device = await api.getDevice(id);
        }
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load data';
      } finally {
        this.loading = false;
      }
    },

    addTag(): void {
      const tag = this.tagInput.trim();
      if (tag && !this.device.tags?.includes(tag)) {
        this.device.tags = [...(this.device.tags ?? []), tag];
      }
      this.tagInput = '';
    },

    removeTag(tag: string): void {
      this.device.tags = this.device.tags?.filter((t) => t !== tag) ?? [];
    },

    addAddress(): void {
      this.device.addresses = [
        ...(this.device.addresses ?? []),
        { ip: '', port: 0, type: 'ipv4', label: '' },
      ];
    },

    removeAddress(index: number): void {
      this.device.addresses = this.device.addresses?.filter((_, i) => i !== index) ?? [];
    },

    async save(): Promise<void> {
      this.saving = true;
      this.error = '';
      try {
        if (this.isEdit && this.device.id) {
          await api.updateDevice(this.device.id, this.device);
        } else {
          await api.createDevice(this.device);
        }
        window.location.href = '/devices';
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to save device';
      } finally {
        this.saving = false;
      }
    },

    cancel(): void {
      window.location.href = '/devices';
    },
  };
}

export { formatDate };
