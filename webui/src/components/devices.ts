// Device Components for Rackd Web UI

import type { Address, Datacenter, Device, DeviceFilter, DeviceRelationship, Network, NetworkPool } from '../core/types';
import { api, RackdAPIError } from '../core/api';
import { debounce, formatDate, createFocusTrap, isValidIP } from '../core/utils';

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
    networks: [] as Network[],
    pools: [] as NetworkPool[],
    poolsCache: {} as Record<string, NetworkPool[]>,
    allPools: [] as NetworkPool[],
    networkFilter: '',
    poolFilter: '',
    statusFilter: '',
    staleFilter: false,
    staleDays: 7,
    loading: true,
    error: '',
    search: '',
    filter: {} as DeviceFilter,
    page: 1,
    pageSize: 10,
    showDeleteModal: false,
    deleteTarget: null as Device | null,
    deleting: false,
    // Device modal (unified for add/edit)
    showDeviceModal: false,
    isEditMode: false,
    modalTab: 'general' as 'general' | 'addresses' | 'tags',
    editDevice: {
      name: '',
      hostname: '',
      make_model: '',
      description: '',
      os: '',
      datacenter_id: '',
      username: '',
      location: '',
      status: 'active' as 'planned' | 'active' | 'maintenance' | 'decommissioned',
      tags: [] as string[],
      addresses: [] as Address[],
      domains: [] as string[],
    } as Partial<Device>,
    tagInput: '',
    domainInput: '',
    saving: false,
    focusTrapCleanup: null as (() => void) | null,
    validationErrors: {} as Record<string, string>,
    showHostnameHelp: false,

    get totalPages(): number {
      return Math.ceil(this.devices.length / this.pageSize) || 1;
    },

    get pagedDevices(): Device[] {
      const start = (this.page - 1) * this.pageSize;
      return this.devices.slice(start, start + this.pageSize);
    },

    get hasMultipleDatacenters(): boolean {
      return this.datacenters.length > 1;
    },

    get deleteModalTitle(): string {
      return 'Delete Device';
    },

    get deleteModalName(): string {
      return this.deleteTarget?.name || '';
    },

    get singleDatacenterId(): string {
      return this.datacenters.length === 1 ? this.datacenters[0].id : '';
    },

    async init(): Promise<void> {
      // Read URL parameters for pre-filtering
      const params = new URLSearchParams(window.location.search);
      const networkParam = params.get('network');
      const poolParam = params.get('pool');
      const statusParam = params.get('status');
      const staleParam = params.get('stale');
      const staleDaysParam = params.get('stale_days');

      if (networkParam) this.networkFilter = networkParam;
      if (poolParam) this.poolFilter = poolParam;
      if (statusParam) this.statusFilter = statusParam;
      if (staleParam === 'true') {
        this.staleFilter = true;
        if (staleDaysParam) this.staleDays = parseInt(staleDaysParam, 10) || 7;
      }

      await Promise.all([this.loadDevices(), this.loadDatacenters(), this.loadNetworks()]);
      await this.loadAllPools();
      
      // Watch for modal open/close to manage focus trap
      this.$watch('showDeviceModal', (show: boolean) => {
        if (show) {
          setTimeout(() => {
            const modal = document.querySelector('[role="dialog"]') as HTMLElement;
            if (modal) this.focusTrapCleanup = createFocusTrap(modal);
          }, 50);
        } else {
          this.focusTrapCleanup?.();
          this.focusTrapCleanup = null;
        }
      });
    },

    async loadDatacenters(): Promise<void> {
      try {
        this.datacenters = (await api.listDatacenters()) || [];
      } catch {
        this.datacenters = [];
      }
    },

    async loadNetworks(): Promise<void> {
      try {
        this.networks = (await api.listNetworks()) || [];
      } catch {
        this.networks = [];
      }
    },

    async loadPoolsForNetwork(networkId: string): Promise<void> {
      if (!networkId) {
        this.pools = [];
        return;
      }
      // Check cache first
      if (this.poolsCache[networkId]) {
        this.pools = this.poolsCache[networkId];
        return;
      }
      try {
        const pools = (await api.listNetworkPools(networkId)) || [];
        this.poolsCache[networkId] = pools;
        this.pools = pools;
      } catch {
        this.pools = [];
      }
    },

    getPoolsForNetwork(networkId: string): NetworkPool[] {
      if (!networkId) return [];
      // Load pools async if not cached
      if (!this.poolsCache[networkId]) {
        this.loadPoolsForNetwork(networkId);
        return [];
      }
      return this.poolsCache[networkId];
    },

    async fetchNextIPForAddress(index: number): Promise<void> {
      const addr = this.editDevice.addresses?.[index];
      if (!addr?.pool_id) return;
      try {
        const result = await api.getNextIP(addr.pool_id);
        if (result?.ip && this.editDevice.addresses?.[index]) {
          this.editDevice.addresses[index].ip = result.ip;
        }
      } catch {
        // Silently fail
      }
    },

    async loadDevices(): Promise<void> {
      this.loading = true;
      this.error = '';
      try {
        let devices: Device[] = [];
        if (this.search) {
          devices = (await api.searchDevices(this.search)) || [];
        } else {
          // Build filter with status and stale
          const filter: DeviceFilter = { ...this.filter };
          if (this.statusFilter) {
            filter.status = this.statusFilter as any;
          }
          if (this.staleFilter) {
            filter.stale = true;
            filter.stale_days = this.staleDays;
          }
          devices = (await api.listDevices(filter)) || [];
        }

        // Apply network filter
        if (this.networkFilter) {
          devices = devices.filter(d =>
            d.addresses?.some(a => a.network_id === this.networkFilter)
          );
        }

        // Apply pool filter
        if (this.poolFilter) {
          devices = devices.filter(d =>
            d.addresses?.some(a => a.pool_id === this.poolFilter)
          );
        }

        // Apply status filter (for search results)
        if (this.search && this.statusFilter) {
          devices = devices.filter(d => d.status === this.statusFilter);
        }

        this.devices = devices;
        this.page = 1;
      } catch (e) {
        this.devices = [];
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
      this.networkFilter = '';
      this.poolFilter = '';
      this.statusFilter = '';
      // Update URL to remove query parameters
      if (window.location.search) {
        window.history.pushState({}, '', '/devices');
      }
      this.loadDevices();
    },

    applyFilters(): void {
      this.page = 1;
      this.loadDevices();
    },

    getNetworkName(networkId: string): string {
      const network = this.networks.find(n => n.id === networkId);
      return network?.name || networkId;
    },

    getPoolName(poolId: string): string {
      const pool = this.allPools.find(p => p.id === poolId);
      return pool?.name || poolId;
    },

    async loadAllPools(): Promise<void> {
      try {
        const pools: NetworkPool[] = [];
        for (const network of this.networks) {
          const networkPools = await api.listNetworkPools(network.id);
          pools.push(...networkPools);
        }
        this.allPools = pools;
      } catch {
        this.allPools = [];
      }
    },

    goToPage(p: number): void {
      if (p >= 1 && p <= this.totalPages) this.page = p;
    },

    confirmDelete(device: Device): void {
      this.deleteTarget = device;
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

    openAddModal(): void {
      this.isEditMode = false;
      this.modalTab = 'general';
      this.editDevice = {
        name: '',
        hostname: '',
        make_model: '',
        description: '',
        os: '',
        datacenter_id: this.datacenters.length === 1 ? this.datacenters[0].id : '',
        username: '',
        location: '',
        status: 'active',
        tags: [],
        addresses: [],
        domains: [],
      };
      this.tagInput = '';
      this.domainInput = '';
      this.pools = [];
      this.showHostnameHelp = false;
      this.showDeviceModal = true;
      setTimeout(() => {
        (document.querySelector('[x-show="showDeviceModal"] input[type="text"]') as HTMLInputElement)?.focus();
      }, 50);
    },

    async openEditModal(device: Device): Promise<void> {
      this.isEditMode = true;
      this.modalTab = 'general';
      this.editDevice = {
        id: device.id,
        name: device.name,
        hostname: device.hostname || '',
        make_model: device.make_model,
        description: device.description,
        os: device.os,
        datacenter_id: device.datacenter_id || '',
        username: device.username || '',
        location: device.location || '',
        status: device.status || 'active',
        tags: [...(device.tags || [])],
        addresses: (device.addresses || []).map((a) => ({ ...a })),
        domains: [...(device.domains || [])],
      };
      this.tagInput = '';
      this.domainInput = '';
      this.pools = [];
      // Preload pools for networks used in existing addresses before showing modal
      const networkIds = [...new Set(
        (device.addresses || []).map((a) => a.network_id).filter((id): id is string => !!id)
      )];
      await Promise.all(networkIds.map((networkId) => this.loadPoolsForNetwork(networkId)));
      this.showDeviceModal = true;
      setTimeout(() => {
        (document.querySelector('[x-show="showDeviceModal"] input[type="text"]') as HTMLInputElement)?.focus();
      }, 50);
    },

    closeModal(): void {
      this.focusTrapCleanup?.();
      this.focusTrapCleanup = null;
      this.showDeviceModal = false;
    },

    addTag(): void {
      const tag = this.tagInput.trim();
      if (tag && !this.editDevice.tags?.includes(tag)) {
        this.editDevice.tags = [...(this.editDevice.tags ?? []), tag];
      }
      this.tagInput = '';
    },

    removeTag(tag: string): void {
      this.editDevice.tags = this.editDevice.tags?.filter((t) => t !== tag) ?? [];
    },

    addDomain(): void {
      const domain = this.domainInput.trim();
      if (domain && !this.editDevice.domains?.includes(domain)) {
        this.editDevice.domains = [...(this.editDevice.domains ?? []), domain];
      }
      this.domainInput = '';
    },

    removeDomain(domain: string): void {
      this.editDevice.domains = this.editDevice.domains?.filter((d) => d !== domain) ?? [];
    },

    addAddress(): void {
      this.editDevice.addresses = [
        ...(this.editDevice.addresses ?? []),
        { ip: '', type: 'ipv4', label: '', network_id: '', switch_port: '', pool_id: '' },
      ];
    },

    removeAddress(index: number): void {
      this.editDevice.addresses = this.editDevice.addresses?.filter((_, i) => i !== index) ?? [];
    },

    validateDevice(): boolean {
      this.validationErrors = {};
      
      if (!this.editDevice.name?.trim()) {
        this.validationErrors.name = 'Device name is required';
      }
      
      this.editDevice.addresses?.forEach((addr, i) => {
        if (!addr.ip?.trim()) {
          this.validationErrors[`addr_${i}_ip`] = 'IP address is required';
        } else if (!isValidIP(addr.ip)) {
          this.validationErrors[`addr_${i}_ip`] = 'Invalid IP address format';
        }
      });
      
      return Object.keys(this.validationErrors).length === 0;
    },

    async saveDevice(): Promise<void> {
      if (!this.validateDevice()) return;
      
      this.saving = true;
      this.error = '';
      try {
        if (this.isEditMode && this.editDevice.id) {
          await api.updateDevice(this.editDevice.id, this.editDevice);
        } else {
          await api.createDevice(this.editDevice);
        }
        this.showDeviceModal = false;
        await this.loadDevices();
      } catch (e) {
        this.error = e instanceof RackdAPIError
          ? e.message
          : this.isEditMode
            ? 'Failed to update device'
            : 'Failed to create device';
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

export function deviceDetail() {
  return {
    device: null as Device | null,
    datacenters: [] as Datacenter[],
    networks: [] as Network[],
    pools: [] as NetworkPool[],
    poolsCache: {} as Record<string, NetworkPool[]>,
    relationships: [] as DeviceRelationship[],
    relatedDevices: new Map() as Map<string, Device>,
    loading: true,
    error: '',
    activeTab: 'details' as 'details' | 'addresses' | 'relationships',
    showDeleteModal: false,
    deleting: false,
    // Edit modal
    showEditModal: false,
    modalTab: 'general' as 'general' | 'addresses' | 'tags',
    editDevice: {} as Partial<Device>,
    tagInput: '',
    domainInput: '',
    saving: false,
    // Relationship modal
    showRelationshipModal: false,
    newRelationship: { type: '', device: null as Device | null, notes: '' },
    relationshipSearch: '',
    relationshipSearchResults: [] as Device[],
    showRelationshipDropdown: false,
    // Relationship filtering/sorting
    relationshipFilter: 'all' as 'all' | 'contains' | 'connected_to' | 'depends_on',
    relationshipSort: 'type' as 'type' | 'date' | 'name',
    // Edit relationship notes
    editingRelationship: null as DeviceRelationship | null,
    editNotes: '',

    async init(): Promise<void> {
      // Wait for next tick to ensure URL is updated after SPA navigation
      await new Promise((resolve) => setTimeout(resolve, 0));
      const id = new URLSearchParams(window.location.search).get('id');
      if (!id) {
        this.error = 'No device ID provided';
        this.loading = false;
        return;
      }
      await Promise.all([this.loadDevice(), this.loadDatacenters(), this.loadNetworks()]);
      
      // Watch for URL changes
      const checkURL = () => {
        const newId = new URLSearchParams(window.location.search).get('id');
        if (newId && newId !== this.device?.id) {
          this.loading = true;
          this.loadDevice();
        }
      };
      window.addEventListener('popstate', checkURL);
      
      // Also check periodically for pushState changes
      const interval = setInterval(() => {
        if (window.location.pathname !== '/devices/detail') {
          clearInterval(interval);
          return;
        }
        checkURL();
      }, 100);
    },

    async loadDatacenters(): Promise<void> {
      try {
        this.datacenters = (await api.listDatacenters()) || [];
      } catch {
        this.datacenters = [];
      }
    },

    async loadNetworks(): Promise<void> {
      try {
        this.networks = (await api.listNetworks()) || [];
      } catch {
        this.networks = [];
      }
    },

    async loadPoolsForNetwork(networkId: string): Promise<void> {
      if (!networkId) {
        this.pools = [];
        return;
      }
      // Check cache first
      if (this.poolsCache[networkId]) {
        this.pools = this.poolsCache[networkId];
        return;
      }
      try {
        const pools = (await api.listNetworkPools(networkId)) || [];
        this.poolsCache[networkId] = pools;
        this.pools = pools;
      } catch {
        this.pools = [];
      }
    },

    getPoolsForNetwork(networkId: string): NetworkPool[] {
      if (!networkId) return [];
      // Load pools async if not cached
      if (!this.poolsCache[networkId]) {
        this.loadPoolsForNetwork(networkId);
        return [];
      }
      return this.poolsCache[networkId];
    },

    async fetchNextIPForAddress(index: number): Promise<void> {
      const addr = this.editDevice.addresses?.[index];
      if (!addr?.pool_id) return;
      try {
        const result = await api.getNextIP(addr.pool_id);
        if (result?.ip && this.editDevice.addresses?.[index]) {
          this.editDevice.addresses[index].ip = result.ip;
        }
      } catch {
        // Silently fail
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
        this.relationships = (await api.getRelationships(this.device.id)) || [];
        for (const rel of this.relationships) {
          const otherId = rel.parent_id === this.device.id ? rel.child_id : rel.parent_id;
          if (!this.relatedDevices.has(otherId)) {
            const d = await api.getDevice(otherId);
            this.relatedDevices.set(otherId, d);
          }
        }
      } catch {
        this.relationships = [];
      }
    },

    setTab(tab: 'details' | 'addresses' | 'relationships'): void {
      this.activeTab = tab;
    },

    getDatacenterName(id?: string): string {
      if (!id) return '-';
      return this.datacenters.find((d) => d.id === id)?.name ?? id;
    },

    get deleteModalTitle(): string {
      return 'Delete Device';
    },

    get deleteModalName(): string {
      return this.device?.name || '';
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

    async openEditModal(): Promise<void> {
      if (!this.device) return;
      this.modalTab = 'general';
      this.editDevice = {
        id: this.device.id,
        name: this.device.name,
        hostname: this.device.hostname || '',
        make_model: this.device.make_model,
        description: this.device.description,
        os: this.device.os,
        datacenter_id: this.device.datacenter_id || '',
        username: this.device.username || '',
        location: this.device.location || '',
        status: this.device.status || 'active',
        tags: [...(this.device.tags || [])],
        addresses: (this.device.addresses || []).map((a) => ({ ...a })),
        domains: [...(this.device.domains || [])],
      };
      this.tagInput = '';
      this.domainInput = '';
      this.pools = [];
      // Preload pools for networks used in existing addresses before showing modal
      const networkIds = [...new Set(
        (this.device.addresses || []).map((a) => a.network_id).filter((id): id is string => !!id)
      )];
      await Promise.all(networkIds.map((networkId) => this.loadPoolsForNetwork(networkId)));
      this.showEditModal = true;
    },

    closeEditModal(): void {
      this.showEditModal = false;
    },

    addTag(): void {
      const tag = this.tagInput.trim();
      if (tag && !this.editDevice.tags?.includes(tag)) {
        this.editDevice.tags = [...(this.editDevice.tags ?? []), tag];
      }
      this.tagInput = '';
    },

    removeTag(tag: string): void {
      this.editDevice.tags = this.editDevice.tags?.filter((t) => t !== tag) ?? [];
    },

    addDomain(): void {
      const domain = this.domainInput.trim();
      if (domain && !this.editDevice.domains?.includes(domain)) {
        this.editDevice.domains = [...(this.editDevice.domains ?? []), domain];
      }
      this.domainInput = '';
    },

    removeDomain(domain: string): void {
      this.editDevice.domains = this.editDevice.domains?.filter((d) => d !== domain) ?? [];
    },

    addAddress(): void {
      this.editDevice.addresses = [
        ...(this.editDevice.addresses ?? []),
        { ip: '', type: 'ipv4', label: '', network_id: '', switch_port: '', pool_id: '' },
      ];
    },

    removeAddress(index: number): void {
      this.editDevice.addresses = this.editDevice.addresses?.filter((_, i) => i !== index) ?? [];
    },

    async saveDevice(): Promise<void> {
      if (!this.editDevice.id) return;
      this.saving = true;
      this.error = '';
      try {
        await api.updateDevice(this.editDevice.id, this.editDevice);
        this.showEditModal = false;
        await this.loadDevice();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to update device';
      } finally {
        this.saving = false;
      }
    },

    formatDate(dateStr?: string): string {
      if (!dateStr) return '-';
      return formatDate(dateStr);
    },

    get filteredRelationships(): DeviceRelationship[] {
      let filtered = this.relationships;
      if (this.relationshipFilter !== 'all') {
        filtered = filtered.filter(r => r.type === this.relationshipFilter);
      }
      
      // Sort
      const sorted = [...filtered];
      if (this.relationshipSort === 'type') {
        sorted.sort((a, b) => a.type.localeCompare(b.type));
      } else if (this.relationshipSort === 'date') {
        sorted.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
      } else if (this.relationshipSort === 'name') {
        sorted.sort((a, b) => {
          const aId = a.parent_id === this.device?.id ? a.child_id : a.parent_id;
          const bId = b.parent_id === this.device?.id ? b.child_id : b.parent_id;
          const aName = this.relatedDevices.get(aId)?.name || '';
          const bName = this.relatedDevices.get(bId)?.name || '';
          return aName.localeCompare(bName);
        });
      }
      return sorted;
    },

    openRelationshipModal(): void {
      this.newRelationship = { type: '', device: null, notes: '' };
      this.relationshipSearch = '';
      this.relationshipSearchResults = [];
      this.showRelationshipDropdown = false;
      this.showRelationshipModal = true;
    },

    closeRelationshipModal(): void {
      this.showRelationshipModal = false;
    },

    async searchDevicesForRelationship(): Promise<void> {
      const query = this.relationshipSearch.trim();
      if (!query || query.length < 2) {
        this.relationshipSearchResults = [];
        return;
      }
      try {
        const results = await api.searchDevices(query);
        this.relationshipSearchResults = results.filter(d => d.id !== this.device?.id);
        this.showRelationshipDropdown = true;
      } catch {
        this.relationshipSearchResults = [];
      }
    },

    selectRelationshipDevice(device: Device): void {
      this.newRelationship.device = device;
      this.relationshipSearch = device.name;
      this.showRelationshipDropdown = false;
    },

    async saveRelationship(): Promise<void> {
      if (!this.device || !this.newRelationship.type || !this.newRelationship.device) return;
      this.saving = true;
      this.error = '';
      try {
        await api.addRelationship(this.device.id, this.newRelationship.device.id, this.newRelationship.type as DeviceRelationship['type'], this.newRelationship.notes);
        await this.loadRelationships();
        this.closeRelationshipModal();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to add relationship';
      } finally {
        this.saving = false;
      }
    },

    startEditNotes(rel: DeviceRelationship): void {
      this.editingRelationship = rel;
      this.editNotes = rel.notes;
    },

    cancelEditNotes(): void {
      this.editingRelationship = null;
      this.editNotes = '';
    },

    async saveNotes(rel: DeviceRelationship): Promise<void> {
      if (!this.device) return;
      try {
        await api.updateRelationshipNotes(rel.parent_id, rel.child_id, rel.type, this.editNotes);
        await this.loadRelationships();
        this.editingRelationship = null;
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to update notes';
      }
    },

    async removeRelationship(rel: DeviceRelationship): Promise<void> {
      if (!this.device || !confirm('Remove this relationship?')) return;
      try {
        await api.removeRelationship(rel.parent_id, rel.child_id, rel.type);
        await this.loadRelationships();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to remove relationship';
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
        this.datacenters = (await api.listDatacenters()) || [];
        if (id) {
          this.device = await api.getDevice(id);
        }
      } catch (e) {
        this.datacenters = [];
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
        { ip: '', type: 'ipv4', label: '' },
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
