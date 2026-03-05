// Pool Components for Rackd Web UI

import type { IPStatus, Network, NetworkPool, Device, Reservation, CreateReservationRequest } from '../core/types';
import { api, RackdAPIError } from '../core/api';

interface PoolDetailData {
  pool: NetworkPool | null;
  network: Network | null;
  heatmap: IPStatus[];
  nextIP: string;
  poolDevices: Device[];
  reservations: Reservation[];
  loadingDevices: boolean;
  loadingReservations: boolean;
  loading: boolean;
  error: string;
  showDeleteModal: boolean;
  showReserveModal: boolean;
  showDeleteReservationModal: boolean;
  deleteReservationItem: Reservation | null;
  deleting: boolean;
  deletingReservation: boolean;
  fetchingNextIP: boolean;
  savingReservation: boolean;
  deleteModalTitle: string;
  deleteModalName: string;
  reservationForm: { ip_address: string; hostname: string; purpose: string; expires_in_days: number; notes: string };
  init(): Promise<void>;
  loadPool(): Promise<void>;
  loadPoolDevices(): Promise<void>;
  loadNetwork(): Promise<void>;
  loadHeatmap(): Promise<void>;
  loadReservations(): Promise<void>;
  fetchNextIP(): Promise<void>;
  getStatusColor(status: IPStatus['status']): string;
  confirmDelete(): void;
  cancelDelete(): void;
  doDelete(): Promise<void>;
  openReserveModal(ip?: string): void;
  closeReserveModal(): void;
  createReservation(): Promise<void>;
  confirmDeleteReservation(reservation: Reservation): void;
  cancelDeleteReservation(): void;
  doDeleteReservation(): Promise<void>;
  getDeviceIP(device: Device): string;
  formatReservationExpires(expiresAt: string | undefined): string;
  firstPoolDevices(): Device[];
  hasMorePoolDevices(): boolean;
  hasTags(): boolean;
  hasHeatmap(): boolean;
  hasReservations(): boolean;
  hasPoolDevices(): boolean;
}

export function poolDetail(): PoolDetailData {
  return {
    pool: null,
    network: null,
    heatmap: [],
    nextIP: '',
    poolDevices: [] as Device[],
    reservations: [] as Reservation[],
    loadingDevices: false,
    loadingReservations: false,
    loading: true,
    error: '',
    showDeleteModal: false,
    showReserveModal: false,
    showDeleteReservationModal: false,
    deleteReservationItem: null as Reservation | null,
    deleting: false,
    deletingReservation: false,
    fetchingNextIP: false,
    savingReservation: false,
    reservationForm: { ip_address: '', hostname: '', purpose: '', expires_in_days: 0, notes: '' },

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
      await this.loadReservations();
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

    async loadReservations(): Promise<void> {
      if (!this.pool) return;
      this.loadingReservations = true;
      try {
        this.reservations = await api.getPoolReservations(this.pool.id);
      } catch {
        this.reservations = [];
      } finally {
        this.loadingReservations = false;
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

    openReserveModal(ip?: string): void {
      this.reservationForm = {
        ip_address: ip || this.nextIP || '',
        hostname: '',
        purpose: '',
        expires_in_days: 0,
        notes: ''
      };
      this.showReserveModal = true;
    },

    closeReserveModal(): void {
      this.showReserveModal = false;
    },

    async createReservation(): Promise<void> {
      if (!this.pool) return;
      this.savingReservation = true;
      this.error = '';
      try {
        const req: CreateReservationRequest = {
          pool_id: this.pool.id,
          ip_address: this.reservationForm.ip_address || undefined,
          hostname: this.reservationForm.hostname || undefined,
          purpose: this.reservationForm.purpose || undefined,
          expires_in_days: this.reservationForm.expires_in_days || undefined,
          notes: this.reservationForm.notes || undefined
        };
        await api.createReservation(req);
        this.closeReserveModal();
        await Promise.all([this.loadHeatmap(), this.loadReservations()]);
        this.nextIP = ''; // Clear cached next IP
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to create reservation';
      } finally {
        this.savingReservation = false;
      }
    },

    confirmDeleteReservation(reservation: Reservation): void {
      this.deleteReservationItem = reservation;
      this.showDeleteReservationModal = true;
    },

    cancelDeleteReservation(): void {
      this.showDeleteReservationModal = false;
      this.deleteReservationItem = null;
    },

    async doDeleteReservation(): Promise<void> {
      if (!this.deleteReservationItem) return;
      this.deletingReservation = true;
      try {
        await api.deleteReservation(this.deleteReservationItem.id);
        this.showDeleteReservationModal = false;
        this.deleteReservationItem = null;
        await Promise.all([this.loadHeatmap(), this.loadReservations()]);
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete reservation';
      } finally {
        this.deletingReservation = false;
      }
    },

    getDeviceIP(device: Device): string {
      return (device.addresses && device.addresses[0]) ? device.addresses[0].ip : '-';
    },

    formatReservationExpires(expiresAt: string | undefined): string {
      if (!expiresAt) return 'Never';
      try {
        return new Date(expiresAt).toLocaleDateString();
      } catch {
        return expiresAt;
      }
    },

    firstPoolDevices(): Device[] {
      return this.poolDevices.slice(0, 5);
    },

    hasMorePoolDevices(): boolean {
      return this.poolDevices.length > 5;
    },

    hasTags(): boolean {
      return !!(this.pool && this.pool.tags && this.pool.tags.length > 0);
    },

    hasHeatmap(): boolean {
      return this.heatmap.length > 0;
    },

    hasReservations(): boolean {
      return this.reservations.length > 0;
    },

    hasPoolDevices(): boolean {
      return this.poolDevices.length > 0;
    }
  };
}

interface PoolFormData {
  editPool: Partial<NetworkPool>;
  networkId: string;
  isEdit: boolean;
  loading: boolean;
  saving: boolean;
  error: string;
  poolTagInput: string;
  init(): Promise<void>;
  addTag(): void;
  removeTag(idx: number): void;
  save(): Promise<void>;
  cancel(): void;
}

export function poolForm(): PoolFormData {
  return {
    editPool: { tags: [] },
    networkId: '',
    isEdit: false,
    loading: true,
    saving: false,
    error: '',
    poolTagInput: '',

    async init(): Promise<void> {
      const params = new URLSearchParams(window.location.search);
      const id = params.get('id');
      this.networkId = params.get('network_id') || '';
      this.isEdit = !!id;
      try {
        if (id) {
          this.editPool = await api.getNetworkPool(id);
          this.networkId = this.editPool.network_id || this.networkId;
        }
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load pool';
      } finally {
        this.loading = false;
      }
    },

    addTag(): void {
      const tag = this.poolTagInput.trim();
      if (tag && !this.editPool.tags?.includes(tag)) {
        this.editPool.tags = [...(this.editPool.tags ?? []), tag];
      }
      this.poolTagInput = '';
    },

    removeTag(idx: number): void {
      this.editPool.tags = this.editPool.tags?.filter((_, i) => i !== idx) ?? [];
    },

    async save(): Promise<void> {
      this.saving = true;
      this.error = '';
      try {
        if (this.isEdit && this.editPool.id) {
          await api.updateNetworkPool(this.editPool.id, this.editPool);
        } else {
          await api.createNetworkPool(this.networkId, this.editPool);
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
