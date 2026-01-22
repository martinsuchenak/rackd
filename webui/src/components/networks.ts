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
  init(): Promise<void>;
  loadNetworks(): Promise<void>;
  loadDatacenters(): Promise<void>;
  setFilter(datacenterId: string): void;
  getDatacenterName(id: string): string;
  confirmDelete(network: Network): void;
  cancelDelete(): void;
  doDelete(): Promise<void>;
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

    async init(): Promise<void> {
      await Promise.all([this.loadNetworks(), this.loadDatacenters()]);
    },

    async loadDatacenters(): Promise<void> {
      try {
        this.datacenters = await api.listDatacenters();
      } catch {
        // Non-critical
      }
    },

    async loadNetworks(): Promise<void> {
      this.loading = true;
      this.error = '';
      try {
        this.networks = await api.listNetworks(this.filter.datacenter_id);
      } catch (e) {
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

    async init(): Promise<void> {
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
        this.datacenters = await api.listDatacenters();
      } catch {
        // Non-critical
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
        this.pools = await api.listNetworkPools(this.network.id);
      } catch {
        // Non-critical
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
        this.datacenters = await api.listDatacenters();
        if (id) {
          this.network = await api.getNetwork(id);
        }
      } catch (e) {
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
