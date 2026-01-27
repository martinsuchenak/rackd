// Network Components for Rackd Web UI

import type { Datacenter, Network, NetworkPool, NetworkUtilization } from '../core/types';
import { RackdAPI, RackdAPIError } from '../core/api';

const api = new RackdAPI();

interface NetworkListData {
  networks: Network[];
  datacenters: Datacenter[];
  loading: boolean;
  error: string;
  filter: { datacenter_id?: string };
  showDeleteModal: boolean;
  deleteTarget: Network | null;
  deleting: boolean;
  hasMultipleDatacenters: boolean;
  init(): Promise<void>;
  loadNetworks(): Promise<void>;
  loadDatacenters(): Promise<void>;
  setFilter(datacenterId: string): void;
  getDatacenterName(id: string): string;
  confirmDelete(network: Network): void;
  cancelDelete(): void;
  doDelete(): Promise<void>;
  openAddModal(): void;
  saveNew(): Promise<void>;
}

export function networkList() {
  return {
    networks: [] as Network[],
    datacenters: [] as Datacenter[],
    loading: true,
    error: '',
    filter: {} as { datacenter_id?: string },
    showDeleteModal: false,
    deleteTarget: null as Network | null,
    deleting: false,
    // Add modal
    showAddModal: false,
    newNetwork: { name: '', subnet: '', vlan_id: 0, datacenter_id: '', description: '' } as Partial<Network>,
    saving: false,

    get hasMultipleDatacenters(): boolean {
      return this.datacenters.length > 1;
    },

    async init(): Promise<void> {
      await Promise.all([this.loadNetworks(), this.loadDatacenters()]);
    },

    async loadDatacenters(): Promise<void> {
      try {
        this.datacenters = (await api.listDatacenters()) || [];
      } catch {
        this.datacenters = [];
      }
    },

    async loadNetworks(): Promise<void> {
      this.loading = true;
      this.error = '';
      try {
        this.networks = (await api.listNetworks(this.filter.datacenter_id)) || [];
      } catch (e) {
        this.networks = [];
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load networks';
      } finally {
        this.loading = false;
      }
    },

    setFilter(datacenterId: string): void {
      this.filter.datacenter_id = datacenterId || undefined;
      this.loadNetworks();
    },

    getDatacenterName(id: string): string {
      return this.datacenters.find((d) => d.id === id)?.name ?? id;
    },

    confirmDelete(network: Network): void {
      this.deleteTarget = network;
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
        await api.deleteNetwork(this.deleteTarget.id);
        this.showDeleteModal = false;
        this.deleteTarget = null;
        await this.loadNetworks();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete network';
      } finally {
        this.deleting = false;
      }
    },

    openAddModal(): void {
      this.showAddModal = true;
      this.newNetwork = {
        name: '',
        subnet: '',
        vlan_id: 0,
        datacenter_id: this.datacenters.length === 1 ? this.datacenters[0].id : '',
        description: '',
      };
      setTimeout(() => {
        (document.querySelector('[x-show="showAddModal"] input[type="text"]') as HTMLInputElement)?.focus();
      }, 50);
    },

    async saveNew(): Promise<void> {
      this.saving = true;
      this.error = '';
      try {
        await api.createNetwork(this.newNetwork);
        this.showAddModal = false;
        this.newNetwork = { name: '', subnet: '', vlan_id: 0, datacenter_id: '', description: '' };
        await this.loadNetworks();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to create network';
      } finally {
        this.saving = false;
      }
    },
  };
}

interface NetworkDetailData {
  network: Network | null;
  datacenters: Datacenter[];
  pools: NetworkPool[];
  utilization: NetworkUtilization | null;
  loading: boolean;
  error: string;
  showDeleteModal: boolean;
  deleting: boolean;
  hasMultipleDatacenters: boolean;
  // Edit network
  showEditModal: boolean;
  editNetwork: Partial<Network>;
  saving: boolean;
  // Pool management
  showPoolModal: boolean;
  editPool: Partial<NetworkPool>;
  isEditPool: boolean;
  savingPool: boolean;
  showDeletePoolModal: boolean;
  deletePoolTarget: NetworkPool | null;
  deletingPool: boolean;
  init(): Promise<void>;
  loadNetwork(): Promise<void>;
  loadDatacenters(): Promise<void>;
  loadPools(): Promise<void>;
  loadUtilization(): Promise<void>;
  getDatacenterName(id: string): string;
  utilizationPercent(): string;
  confirmDelete(): void;
  cancelDelete(): void;
  doDelete(): Promise<void>;
  // Network edit methods
  openEditModal(): void;
  closeEditModal(): void;
  saveNetwork(): Promise<void>;
  // Pool methods
  openAddPoolModal(): void;
  openEditPoolModal(pool: NetworkPool): void;
  closePoolModal(): void;
  savePool(): Promise<void>;
  confirmDeletePool(pool: NetworkPool): void;
  cancelDeletePool(): void;
  doDeletePool(): Promise<void>;
}

export function networkDetail(): NetworkDetailData {
  return {
    network: null,
    datacenters: [],
    pools: [],
    utilization: null,
    loading: true,
    error: '',
    showDeleteModal: false,
    deleting: false,
    // Edit network modal
    showEditModal: false,
    editNetwork: {} as Partial<Network>,
    saving: false,
    // Pool modal
    showPoolModal: false,
    editPool: {} as Partial<NetworkPool>,
    isEditPool: false,
    savingPool: false,
    showDeletePoolModal: false,
    deletePoolTarget: null as NetworkPool | null,
    deletingPool: false,

    get hasMultipleDatacenters(): boolean {
      return this.datacenters.length > 1;
    },

    async init(): Promise<void> {
      // Wait for next tick to ensure URL is updated after SPA navigation
      await new Promise((resolve) => setTimeout(resolve, 0));
      const id = new URLSearchParams(window.location.search).get('id');
      if (!id) {
        this.error = 'No network ID provided';
        this.loading = false;
        return;
      }
      await Promise.all([this.loadNetwork(), this.loadDatacenters()]);
    },

    async loadDatacenters(): Promise<void> {
      try {
        this.datacenters = (await api.listDatacenters()) || [];
      } catch {
        this.datacenters = [];
      }
    },

    async loadNetwork(): Promise<void> {
      const id = new URLSearchParams(window.location.search).get('id');
      if (!id) return;
      this.loading = true;
      try {
        this.network = await api.getNetwork(id);
        await Promise.all([this.loadPools(), this.loadUtilization()]);
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load network';
      } finally {
        this.loading = false;
      }
    },

    async loadPools(): Promise<void> {
      if (!this.network) return;
      try {
        this.pools = (await api.listNetworkPools(this.network.id)) || [];
      } catch {
        this.pools = [];
      }
    },

    async loadUtilization(): Promise<void> {
      if (!this.network) return;
      try {
        this.utilization = await api.getNetworkUtilization(this.network.id);
      } catch {
        // Non-critical
      }
    },

    getDatacenterName(id: string): string {
      return this.datacenters.find((d) => d.id === id)?.name ?? id;
    },

    utilizationPercent(): string {
      if (!this.utilization) return '0';
      return this.utilization.utilization.toFixed(1);
    },

    confirmDelete(): void {
      this.showDeleteModal = true;
    },

    cancelDelete(): void {
      this.showDeleteModal = false;
    },

    async doDelete(): Promise<void> {
      if (!this.network) return;
      this.deleting = true;
      try {
        await api.deleteNetwork(this.network.id);
        window.location.href = '/networks';
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete network';
        this.deleting = false;
      }
    },

    // Network edit methods
    openEditModal(): void {
      if (!this.network) return;
      this.editNetwork = {
        ...this.network,
        datacenter_id: this.datacenters.length === 1 && this.network.datacenter_id ? this.datacenters[0].id : this.network.datacenter_id || '',
      };
      this.showEditModal = true;
    },

    closeEditModal(): void {
      this.showEditModal = false;
      this.editNetwork = {};
    },

    async saveNetwork(): Promise<void> {
      if (!this.network || !this.editNetwork.id) return;
      this.saving = true;
      this.error = '';
      try {
        await api.updateNetwork(this.editNetwork.id, this.editNetwork);
        this.showEditModal = false;
        await this.loadNetwork();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to update network';
      } finally {
        this.saving = false;
      }
    },

    // Pool methods
    openAddPoolModal(): void {
      this.editPool = { tags: [] };
      this.isEditPool = false;
      this.showPoolModal = true;
      setTimeout(() => {
        (document.querySelector('[x-show="showPoolModal"] input[type="text"]') as HTMLInputElement)?.focus();
      }, 50);
    },

    openEditPoolModal(pool: NetworkPool): void {
      this.editPool = { ...pool, tags: [...(pool.tags || [])] };
      this.isEditPool = true;
      this.showPoolModal = true;
      setTimeout(() => {
        (document.querySelector('[x-show="showPoolModal"] input[type="text"]') as HTMLInputElement)?.focus();
      }, 50);
    },

    closePoolModal(): void {
      this.showPoolModal = false;
      this.editPool = {};
      this.isEditPool = false;
    },

    async savePool(): Promise<void> {
      if (!this.network) return;
      this.savingPool = true;
      this.error = '';
      try {
        if (this.isEditPool && this.editPool.id) {
          await api.updateNetworkPool(this.editPool.id, this.editPool);
        } else {
          await api.createNetworkPool(this.network.id, this.editPool);
        }
        this.showPoolModal = false;
        this.editPool = {};
        await this.loadPools();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to save pool';
      } finally {
        this.savingPool = false;
      }
    },

    confirmDeletePool(pool: NetworkPool): void {
      this.deletePoolTarget = pool;
      this.showDeletePoolModal = true;
      setTimeout(() => {
        const modal = document.querySelector('[x-show="showDeletePoolModal"]');
        if (modal) {
          const cancelBtn = modal.querySelector('button[type="button"]') as HTMLButtonElement;
          cancelBtn?.focus();
        }
      }, 50);
    },

    cancelDeletePool(): void {
      this.showDeletePoolModal = false;
      this.deletePoolTarget = null;
    },

    async doDeletePool(): Promise<void> {
      if (!this.deletePoolTarget) return;
      this.deletingPool = true;
      try {
        await api.deleteNetworkPool(this.deletePoolTarget.id);
        this.showDeletePoolModal = false;
        this.deletePoolTarget = null;
        await this.loadPools();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete pool';
      } finally {
        this.deletingPool = false;
      }
    },
  };
}

interface NetworkFormData {
  network: Partial<Network>;
  datacenters: Datacenter[];
  isEdit: boolean;
  loading: boolean;
  saving: boolean;
  error: string;
  init(): Promise<void>;
  save(): Promise<void>;
  cancel(): void;
}

export function networkForm(): NetworkFormData {
  return {
    network: { vlan_id: 0 },
    datacenters: [],
    isEdit: false,
    loading: true,
    saving: false,
    error: '',

    async init(): Promise<void> {
      const id = new URLSearchParams(window.location.search).get('id');
      this.isEdit = !!id;
      try {
        this.datacenters = (await api.listDatacenters()) || [];
        if (id) {
          this.network = await api.getNetwork(id);
        }
      } catch (e) {
        this.datacenters = [];
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load data';
      } finally {
        this.loading = false;
      }
    },

    async save(): Promise<void> {
      this.saving = true;
      this.error = '';
      try {
        if (this.isEdit && this.network.id) {
          await api.updateNetwork(this.network.id, this.network);
        } else {
          await api.createNetwork(this.network);
        }
        window.location.href = '/networks';
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to save network';
      } finally {
        this.saving = false;
      }
    },

    cancel(): void {
      window.location.href = '/networks';
    },
  };
}
