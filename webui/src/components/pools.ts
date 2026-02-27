// Pool Components for Rackd Web UI

import type { IPStatus, Network, NetworkPool, Device } from '../core/types';
import { api, RackdAPIError } from '../core/api';

interface PoolDetailData {
  pool: NetworkPool | null;
  network: Network | null;
  heatmap: IPStatus[];
  nextIP: string;
  poolDevices: Device[];
  loadingDevices: boolean;
  loading: boolean;
  error: string;
  showDeleteModal: boolean;
  deleting: boolean;
  fetchingNextIP: boolean;
  deleteModalTitle: string;
  deleteModalName: string;
  init(): Promise<void>;
  loadPool(): Promise<void>;
  loadPoolDevices(): Promise<void>;
  loadNetwork(): Promise<void>;
  loadHeatmap(): Promise<void>;
  fetchNextIP(): Promise<void>;
  getStatusColor(status: IPStatus['status']): string;
  confirmDelete(): void;
  cancelDelete(): void;
  doDelete(): Promise<void>;
}

export function poolDetail(): PoolDetailData {
  return {
    pool: null,
    network: null,
    heatmap: [],
    nextIP: '',
    poolDevices: [] as Device[],
    loadingDevices: false,
    loading: true,
    error: '',
    showDeleteModal: false,
    deleting: false,
    fetchingNextIP: false,

    get deleteModalTitle(): string {
      return 'Delete Pool';
    },

    get deleteModalName(): string {
      return this.pool?.name || '';
    },

    async init(): Promise<void> {
      // Wait for next tick to ensure URL is updated after SPA navigation
      await new Promise((resolve) => setTimeout(resolve, 0));
      const id = new URLSearchParams(window.location.search).get('id');
      if (!id) {
        this.error = 'No pool ID provided';
        this.loading = false;
        return;
      }
      await this.loadPool();
      await this.loadPoolDevices();
    },

    async loadPoolDevices(): Promise<void> {
      const id = new URLSearchParams(window.location.search).get('id');
      if (!id) return;
      this.loadingDevices = true;
      try {
        // Get all devices and filter by pool_id
        const allDevices = await api.listDevices({});
        this.poolDevices = allDevices.filter(d =>
          d.addresses?.some(a => a.pool_id === id)
        );
      } catch {
        this.poolDevices = [];
      } finally {
        this.loadingDevices = false;
      }
    },

    async loadPool(): Promise<void> {
      const id = new URLSearchParams(window.location.search).get('id');
      if (!id) return;
      this.loading = true;
      try {
        this.pool = await api.getNetworkPool(id);
        await Promise.all([this.loadNetwork(), this.loadHeatmap()]);
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load pool';
      } finally {
        this.loading = false;
      }
    },

    async loadNetwork(): Promise<void> {
      if (!this.pool) return;
      try {
        this.network = await api.getNetwork(this.pool.network_id);
      } catch {
        // Non-critical
      }
    },

    async loadHeatmap(): Promise<void> {
      if (!this.pool) return;
      try {
        this.heatmap = (await api.getPoolHeatmap(this.pool.id)) || [];
      } catch {
        this.heatmap = [];
      }
    },

    async fetchNextIP(): Promise<void> {
      if (!this.pool) return;
      this.fetchingNextIP = true;
      try {
        const result = await api.getNextIP(this.pool.id);
        this.nextIP = result.ip;
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to get next IP';
      } finally {
        this.fetchingNextIP = false;
      }
    },

    getStatusColor(status: IPStatus['status']): string {
      switch (status) {
        case 'available': return 'bg-green-500';
        case 'used': return 'bg-red-500';
        case 'reserved': return 'bg-yellow-500';
        case 'conflicted': return 'bg-orange-500';
        default: return 'bg-gray-300';
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
      if (!this.pool) return;
      this.deleting = true;
      try {
        await api.deleteNetworkPool(this.pool.id);
        window.location.href = this.network
          ? `/networks/detail?id=${this.network.id}`
          : '/networks';
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete pool';
        this.deleting = false;
      }
    },
  };
}

interface PoolFormData {
  pool: Partial<NetworkPool>;
  networkId: string;
  isEdit: boolean;
  loading: boolean;
  saving: boolean;
  error: string;
  tagInput: string;
  init(): Promise<void>;
  addTag(): void;
  removeTag(tag: string): void;
  save(): Promise<void>;
  cancel(): void;
}

export function poolForm(): PoolFormData {
  return {
    pool: { tags: [] },
    networkId: '',
    isEdit: false,
    loading: true,
    saving: false,
    error: '',
    tagInput: '',

    async init(): Promise<void> {
      const params = new URLSearchParams(window.location.search);
      const id = params.get('id');
      this.networkId = params.get('network_id') || '';
      this.isEdit = !!id;
      try {
        if (id) {
          this.pool = await api.getNetworkPool(id);
          this.networkId = this.pool.network_id || this.networkId;
        }
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load pool';
      } finally {
        this.loading = false;
      }
    },

    addTag(): void {
      const tag = this.tagInput.trim();
      if (tag && !this.pool.tags?.includes(tag)) {
        this.pool.tags = [...(this.pool.tags ?? []), tag];
      }
      this.tagInput = '';
    },

    removeTag(tag: string): void {
      this.pool.tags = this.pool.tags?.filter((t) => t !== tag) ?? [];
    },

    async save(): Promise<void> {
      this.saving = true;
      this.error = '';
      try {
        if (this.isEdit && this.pool.id) {
          await api.updateNetworkPool(this.pool.id, this.pool);
        } else {
          await api.createNetworkPool(this.networkId, this.pool);
        }
        window.location.href = `/networks/detail?id=${this.networkId}`;
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to save pool';
      } finally {
        this.saving = false;
      }
    },

    cancel(): void {
      window.location.href = this.networkId
        ? `/networks/detail?id=${this.networkId}`
        : '/networks';
    },
  };
}
