// DNS Management Components

import type {
  DNSProvider, DNSProviderType, DNSProviderFilter,
  CreateDNSProviderRequest, UpdateDNSProviderRequest,
  DNSZone, DNSZoneFilter,
  CreateDNSZoneRequest, UpdateDNSZoneRequest,
  DNSRecord, DNSRecordFilter, UpdateDNSRecordRequest,
  SyncStatus, RecordSyncStatus, Network
} from '../core/types';
import { api, RackdAPIError } from '../core/api';

// ==================== DNS PROVIDERS COMPONENT ====================

interface ProviderValidationErrors {
  name?: string;
  endpoint?: string;
  token?: string;
}

type ProviderModalType = '' | 'create' | 'edit' | 'delete' | 'test';

interface DNSProvidersData {
  providers: DNSProvider[];
  loading: boolean;
  error: string;

  // Filters
  filterType: string;

  // Single modal state
  modalType: ProviderModalType;
  selectedProvider: DNSProvider | null;
  testingConnection: boolean;
  testResult: { success: boolean; message: string } | null;

  // Form data
  formData: {
    name: string;
    type: DNSProviderType;
    endpoint: string;
    token: string;
    description: string;
  };
  validationErrors: ProviderValidationErrors;
  saving: boolean;
  deleting: boolean;

  // Computed properties for template compatibility
  get showCreateModal(): boolean;
  get showEditModal(): boolean;
  get showDeleteModal(): boolean;

  init(): Promise<void>;
  loadProviders(): Promise<void>;
  applyFilters(): Promise<void>;

  // Modal management
  openCreateModal(): void;
  openEditModal(provider: DNSProvider): void;
  openDeleteModal(provider: DNSProvider): void;
  openTestModal(provider: DNSProvider): void;
  closeModal(): void;
  closeDeleteModal(): void;

  // CRUD operations
  saveProvider(): Promise<void>;
  doDeleteProvider(): Promise<void>;
  doTestConnection(): Promise<void>;

  // Form helpers
  validateForm(): boolean;

  // Utilities
  formatDate(dateStr: string): string;
  formatType(type: DNSProviderType): string;
}

export function dnsProvidersComponent(): DNSProvidersData {
  return {
    providers: [],
    loading: true,
    error: '',

    filterType: '',

    modalType: '',
    selectedProvider: null,
    testingConnection: false,
    testResult: null,

    formData: {
      name: '',
      type: 'technitium',
      endpoint: '',
      token: '',
      description: ''
    },
    validationErrors: {},
    saving: false,
    deleting: false,

    get showCreateModal(): boolean { return this.modalType === 'create'; },
    get showEditModal(): boolean { return this.modalType === 'edit'; },
    get showDeleteModal(): boolean { return this.modalType === 'delete'; },

    async init(): Promise<void> {
      await this.loadProviders();
    },

    async loadProviders(): Promise<void> {
      this.loading = true;
      try {
        const filter: DNSProviderFilter = {};
        if (this.filterType) filter.type = this.filterType as DNSProviderType;
        this.providers = await api.listDNSProviders(Object.keys(filter).length > 0 ? filter : undefined);
      } catch (e) {
        console.error('Failed to load DNS providers:', e);
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load DNS providers';
      } finally {
        this.loading = false;
      }
    },

    async applyFilters(): Promise<void> {
      await this.loadProviders();
    },

    openCreateModal(): void {
      this.modalType = '';
      this.selectedProvider = null;
      this.testResult = null;
      this.validationErrors = {};

      this.formData = {
        name: '',
        type: 'technitium',
        endpoint: '',
        token: '',
        description: ''
      };
      this.modalType = 'create';
    },

    openEditModal(provider: DNSProvider): void {
      this.modalType = '';
      this.validationErrors = {};
      this.testResult = null;

      this.selectedProvider = provider;
      this.formData = {
        name: provider.name,
        type: provider.type,
        endpoint: provider.endpoint,
        token: '', // Never populate token for security
        description: provider.description || ''
      };
      this.modalType = 'edit';
    },

    openDeleteModal(provider: DNSProvider): void {
      this.modalType = '';
      this.validationErrors = {};
      this.testResult = null;
      this.selectedProvider = provider;
      this.modalType = 'delete';
    },

    openTestModal(provider: DNSProvider): void {
      this.modalType = '';
      this.validationErrors = {};
      this.testResult = null;
      this.selectedProvider = provider;
      this.modalType = 'test';
    },

    closeModal(): void {
      this.modalType = '';
      this.selectedProvider = null;
      this.validationErrors = {};
      this.testResult = null;
    },

    closeDeleteModal(): void {
      this.modalType = '';
      this.selectedProvider = null;
    },

    validateForm(): boolean {
      this.validationErrors = {};

      if (!this.formData.name.trim()) {
        this.validationErrors.name = 'Name is required';
      }

      if (!this.formData.endpoint.trim()) {
        this.validationErrors.endpoint = 'Endpoint is required';
      }

      // Token required for create, optional for edit (user may leave blank to keep existing)
      if (!this.selectedProvider && !this.formData.token.trim()) {
        this.validationErrors.token = 'Token is required';
      }

      return Object.keys(this.validationErrors).length === 0;
    },

    async saveProvider(): Promise<void> {
      if (!this.validateForm()) {
        return;
      }

      this.saving = true;
      this.error = '';

      try {
        if (this.selectedProvider) {
          // Update existing provider
          const updateData: UpdateDNSProviderRequest = {
            name: this.formData.name,
            endpoint: this.formData.endpoint,
            description: this.formData.description || undefined
          };
          // Only include token if provided
          if (this.formData.token.trim()) {
            updateData.token = this.formData.token;
          }
          await api.updateDNSProvider(this.selectedProvider.id, updateData);
        } else {
          // Create new provider
          const createData: CreateDNSProviderRequest = {
            name: this.formData.name,
            type: this.formData.type,
            endpoint: this.formData.endpoint,
            token: this.formData.token,
            description: this.formData.description || undefined
          };
          await api.createDNSProvider(createData);
        }
        await this.loadProviders();
        this.closeModal();
      } catch (e) {
        console.error('Failed to save DNS provider:', e);
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to save DNS provider';
      } finally {
        this.saving = false;
      }
    },

    async doDeleteProvider(): Promise<void> {
      if (!this.selectedProvider) return;

      this.deleting = true;
      try {
        await api.deleteDNSProvider(this.selectedProvider.id);
        await this.loadProviders();
        this.closeDeleteModal();
      } catch (e) {
        console.error('Failed to delete DNS provider:', e);
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete DNS provider';
      } finally {
        this.deleting = false;
      }
    },

    async doTestConnection(): Promise<void> {
      if (!this.selectedProvider) return;

      this.testingConnection = true;
      this.testResult = null;

      try {
        await api.testDNSProvider(this.selectedProvider.id);
        this.testResult = { success: true, message: 'Connection successful!' };
      } catch (e) {
        console.error('Connection test failed:', e);
        this.testResult = {
          success: false,
          message: e instanceof RackdAPIError ? e.message : 'Connection failed'
        };
      } finally {
        this.testingConnection = false;
      }
    },

    formatDate(dateStr: string): string {
      if (!dateStr) return '';
      const date = new Date(dateStr);
      return date.toLocaleString();
    },

    formatType(type: DNSProviderType): string {
      const typeMap: Record<DNSProviderType, string> = {
        'technitium': 'Technitium',
        'powerdns': 'PowerDNS',
        'bind': 'BIND'
      };
      return typeMap[type] || type;
    }
  };
}

// ==================== DNS ZONES COMPONENT ====================

interface ZoneValidationErrors {
  name?: string;
  provider_id?: string;
  ttl?: string;
}

type ZoneModalType = '' | 'create' | 'edit' | 'delete' | 'sync' | 'import';

interface DNSZonesData {
  zones: DNSZone[];
  providers: DNSProvider[];
  networks: Network[];
  loading: boolean;
  error: string;

  // Filters
  filterProvider: string;
  filterNetwork: string;
  filterAutoSync: string;

  // Single modal state
  modalType: ZoneModalType;
  selectedZone: DNSZone | null;
  syncResult: { success: boolean; message: string; details?: string } | null;
  syncing: boolean;
  importing: boolean;

  // Form data
  formData: {
    name: string;
    provider_id: string;
    network_id: string;
    auto_sync: boolean;
    create_ptr: boolean;
    ptr_zone: string;
    ttl: number;
    description: string;
  };
  validationErrors: ZoneValidationErrors;
  saving: boolean;
  deleting: boolean;

  // Computed properties for template compatibility
  get showCreateModal(): boolean;
  get showEditModal(): boolean;
  get showDeleteModal(): boolean;
  get showSyncModal(): boolean;
  get showImportModal(): boolean;

  init(): Promise<void>;
  loadZones(): Promise<void>;
  loadProviders(): Promise<void>;
  loadNetworks(): Promise<void>;
  applyFilters(): Promise<void>;

  // Modal management
  openCreateModal(): void;
  openEditModal(zone: DNSZone): void;
  openDeleteModal(zone: DNSZone): void;
  openSyncModal(zone: DNSZone): void;
  openImportModal(zone: DNSZone): void;
  closeModal(): void;
  closeDeleteModal(): void;
  closeSyncModal(): void;
  closeImportModal(): void;

  // CRUD operations
  saveZone(): Promise<void>;
  doDeleteZone(): Promise<void>;
  doSyncZone(): Promise<void>;
  doImportZone(): Promise<void>;

  // Form helpers
  validateForm(): boolean;

  // Utilities
  formatDate(dateStr: string): string;
  getProviderName(id: string): string;
  getNetworkName(id: string): string;
  formatSyncStatus(status?: SyncStatus): string;
}

export function dnsZonesComponent(): DNSZonesData {
  return {
    zones: [],
    providers: [],
    networks: [],
    loading: true,
    error: '',

    filterProvider: '',
    filterNetwork: '',
    filterAutoSync: '',

    modalType: '',
    selectedZone: null,
    syncResult: null,
    syncing: false,
    importing: false,

    formData: {
      name: '',
      provider_id: '',
      network_id: '',
      auto_sync: false,
      create_ptr: false,
      ptr_zone: '',
      ttl: 3600,
      description: ''
    },
    validationErrors: {},
    saving: false,
    deleting: false,

    get showCreateModal(): boolean { return this.modalType === 'create'; },
    get showEditModal(): boolean { return this.modalType === 'edit'; },
    get showDeleteModal(): boolean { return this.modalType === 'delete'; },
    get showSyncModal(): boolean { return this.modalType === 'sync'; },
    get showImportModal(): boolean { return this.modalType === 'import'; },

    async init(): Promise<void> {
      await Promise.all([
        this.loadZones(),
        this.loadProviders(),
        this.loadNetworks()
      ]);
    },

    async loadZones(): Promise<void> {
      this.loading = true;
      try {
        const filter: DNSZoneFilter = {};
        if (this.filterProvider) filter.provider_id = this.filterProvider;
        if (this.filterNetwork) filter.network_id = this.filterNetwork;
        if (this.filterAutoSync !== '') filter.auto_sync = this.filterAutoSync === 'true';
        this.zones = await api.listDNSZones(Object.keys(filter).length > 0 ? filter : undefined);
      } catch (e) {
        console.error('Failed to load DNS zones:', e);
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load DNS zones';
      } finally {
        this.loading = false;
      }
    },

    async loadProviders(): Promise<void> {
      try {
        this.providers = await api.listDNSProviders();
      } catch (e) {
        console.error('Failed to load DNS providers:', e);
      }
    },

    async loadNetworks(): Promise<void> {
      try {
        this.networks = await api.listNetworks();
      } catch (e) {
        console.error('Failed to load networks:', e);
      }
    },

    async applyFilters(): Promise<void> {
      await this.loadZones();
    },

    openCreateModal(): void {
      this.modalType = '';
      this.selectedZone = null;
      this.syncResult = null;
      this.validationErrors = {};

      this.formData = {
        name: '',
        provider_id: '',
        network_id: '',
        auto_sync: false,
        create_ptr: false,
        ptr_zone: '',
        ttl: 3600,
        description: ''
      };
      this.modalType = 'create';
    },

    openEditModal(zone: DNSZone): void {
      this.modalType = '';
      this.validationErrors = {};
      this.syncResult = null;

      this.selectedZone = zone;
      this.formData = {
        name: zone.name,
        provider_id: zone.provider_id,
        network_id: zone.network_id || '',
        auto_sync: zone.auto_sync,
        create_ptr: zone.create_ptr,
        ptr_zone: zone.ptr_zone || '',
        ttl: zone.ttl,
        description: zone.description || ''
      };
      this.modalType = 'edit';
    },

    openDeleteModal(zone: DNSZone): void {
      this.modalType = '';
      this.validationErrors = {};
      this.syncResult = null;
      this.selectedZone = zone;
      this.modalType = 'delete';
    },

    openSyncModal(zone: DNSZone): void {
      this.modalType = '';
      this.validationErrors = {};
      this.syncResult = null;
      this.selectedZone = zone;
      this.modalType = 'sync';
    },

    openImportModal(zone: DNSZone): void {
      this.modalType = '';
      this.validationErrors = {};
      this.syncResult = null;
      this.selectedZone = zone;
      this.modalType = 'import';
    },

    closeModal(): void {
      this.modalType = '';
      this.selectedZone = null;
      this.validationErrors = {};
      this.syncResult = null;
    },

    closeDeleteModal(): void {
      this.modalType = '';
      this.selectedZone = null;
    },

    closeSyncModal(): void {
      this.modalType = '';
      this.selectedZone = null;
      this.syncResult = null;
    },

    closeImportModal(): void {
      this.modalType = '';
      this.selectedZone = null;
      this.syncResult = null;
    },

    validateForm(): boolean {
      this.validationErrors = {};

      if (!this.formData.name.trim()) {
        this.validationErrors.name = 'Zone name is required';
      }

      if (!this.formData.provider_id) {
        this.validationErrors.provider_id = 'Provider is required';
      }

      if (this.formData.ttl < 0) {
        this.validationErrors.ttl = 'TTL must be positive';
      }

      return Object.keys(this.validationErrors).length === 0;
    },

    async saveZone(): Promise<void> {
      if (!this.validateForm()) {
        return;
      }

      this.saving = true;
      this.error = '';

      try {
        if (this.selectedZone) {
          // Update existing zone
          const updateData: UpdateDNSZoneRequest = {
            name: this.formData.name,
            network_id: this.formData.network_id || undefined,
            auto_sync: this.formData.auto_sync,
            create_ptr: this.formData.create_ptr,
            ptr_zone: this.formData.ptr_zone || undefined,
            ttl: this.formData.ttl,
            description: this.formData.description || undefined
          };
          await api.updateDNSZone(this.selectedZone.id, updateData);
        } else {
          // Create new zone
          const createData: CreateDNSZoneRequest = {
            name: this.formData.name,
            provider_id: this.formData.provider_id,
            network_id: this.formData.network_id || undefined,
            auto_sync: this.formData.auto_sync,
            create_ptr: this.formData.create_ptr,
            ptr_zone: this.formData.ptr_zone || undefined,
            ttl: this.formData.ttl,
            description: this.formData.description || undefined
          };
          await api.createDNSZone(createData);
        }
        await this.loadZones();
        this.closeModal();
      } catch (e) {
        console.error('Failed to save DNS zone:', e);
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to save DNS zone';
      } finally {
        this.saving = false;
      }
    },

    async doDeleteZone(): Promise<void> {
      if (!this.selectedZone) return;

      this.deleting = true;
      try {
        await api.deleteDNSZone(this.selectedZone.id);
        await this.loadZones();
        this.closeDeleteModal();
      } catch (e) {
        console.error('Failed to delete DNS zone:', e);
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete DNS zone';
      } finally {
        this.deleting = false;
      }
    },

    async doSyncZone(): Promise<void> {
      if (!this.selectedZone) return;

      this.syncing = true;
      this.syncResult = null;

      try {
        const result = await api.syncDNSZone(this.selectedZone.id);
        this.syncResult = {
          success: result.success,
          message: result.success
            ? `Synced ${result.synced}/${result.total} records successfully`
            : `Sync completed with errors: ${result.failed} failed`,
          details: result.error
        };
      } catch (e) {
        console.error('Sync failed:', e);
        this.syncResult = {
          success: false,
          message: e instanceof RackdAPIError ? e.message : 'Sync failed'
        };
      } finally {
        this.syncing = false;
      }
    },

    async doImportZone(): Promise<void> {
      if (!this.selectedZone) return;

      this.importing = true;
      this.syncResult = null;

      try {
        const result = await api.importDNSZone(this.selectedZone.id);
        this.syncResult = {
          success: result.success || result.imported > 0,
          message: `Imported ${result.imported} records, skipped ${result.skipped}`,
          details: result.failed > 0 ? `${result.failed} records failed` : undefined
        };
        if (result.success) {
          await this.loadZones();
        }
      } catch (e) {
        console.error('Import failed:', e);
        this.syncResult = {
          success: false,
          message: e instanceof RackdAPIError ? e.message : 'Import failed'
        };
      } finally {
        this.importing = false;
      }
    },

    formatDate(dateStr: string): string {
      if (!dateStr) return '-';
      const date = new Date(dateStr);
      return date.toLocaleString();
    },

    getProviderName(id: string): string {
      if (!id) return '-';
      const provider = this.providers.find(p => p.id === id);
      return provider ? provider.name : id;
    },

    getNetworkName(id: string): string {
      if (!id) return '-';
      const network = this.networks.find(n => n.id === id);
      return network ? network.name : id;
    },

    formatSyncStatus(status?: SyncStatus): string {
      if (!status) return '-';
      const statusMap: Record<SyncStatus, string> = {
        'success': 'Success',
        'failed': 'Failed',
        'partial': 'Partial'
      };
      return statusMap[status] || status;
    }
  };
}

// ==================== DNS RECORDS COMPONENT ====================

interface RecordValidationErrors {
  name?: string;
  type?: string;
  value?: string;
  ttl?: string;
}

type RecordModalType = '' | 'edit' | 'delete';

interface DNSRecordsData {
  records: DNSRecord[];
  zoneId: string;
  zone: DNSZone | null;
  loading: boolean;
  error: string;

  // Filters
  filterType: string;
  filterSyncStatus: string;

  // Single modal state
  modalType: RecordModalType;
  selectedRecord: DNSRecord | null;

  // Form data
  formData: {
    name: string;
    type: string;
    value: string;
    ttl: number;
  };
  validationErrors: RecordValidationErrors;
  saving: boolean;
  deleting: boolean;

  // Computed properties for template compatibility
  get showEditModal(): boolean;
  get showDeleteModal(): boolean;

  init(): Promise<void>;
  loadRecords(): Promise<void>;
  loadZone(): Promise<void>;
  applyFilters(): Promise<void>;

  // Modal management
  openEditModal(record: DNSRecord): void;
  openDeleteModal(record: DNSRecord): void;
  closeModal(): void;
  closeDeleteModal(): void;

  // CRUD operations
  saveRecord(): Promise<void>;
  doDeleteRecord(): Promise<void>;

  // Form helpers
  validateForm(): boolean;

  // Utilities
  formatDate(dateStr: string): string;
  formatSyncStatus(status: RecordSyncStatus): string;
}

export function dnsRecordsComponent(): DNSRecordsData {
  return {
    records: [],
    zoneId: '',
    zone: null,
    loading: true,
    error: '',

    filterType: '',
    filterSyncStatus: '',

    modalType: '',
    selectedRecord: null,

    formData: {
      name: '',
      type: 'A',
      value: '',
      ttl: 3600
    },
    validationErrors: {},
    saving: false,
    deleting: false,

    get showEditModal(): boolean { return this.modalType === 'edit'; },
    get showDeleteModal(): boolean { return this.modalType === 'delete'; },

    async init(): Promise<void> {
      // Get zoneId from URL path: /dns/records/{zoneId}
      const pathParts = window.location.pathname.split('/');
      const recordsIndex = pathParts.indexOf('records');
      if (recordsIndex !== -1 && pathParts.length > recordsIndex + 1) {
        this.zoneId = pathParts[recordsIndex + 1];
      }
      if (!this.zoneId) {
        this.error = 'Zone ID not found in URL';
        return;
      }
      await Promise.all([
        this.loadRecords(),
        this.loadZone()
      ]);
    },

    async loadRecords(): Promise<void> {
      this.loading = true;
      try {
        const filter: DNSRecordFilter = { zone_id: this.zoneId };
        if (this.filterType) filter.type = this.filterType;
        if (this.filterSyncStatus) filter.sync_status = this.filterSyncStatus as RecordSyncStatus;
        this.records = await api.listDNSRecords(Object.keys(filter).length > 1 ? filter : { zone_id: this.zoneId });
      } catch (e) {
        console.error('Failed to load DNS records:', e);
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load DNS records';
      } finally {
        this.loading = false;
      }
    },

    async loadZone(): Promise<void> {
      try {
        this.zone = await api.getDNSZone(this.zoneId);
      } catch (e) {
        console.error('Failed to load zone:', e);
      }
    },

    async applyFilters(): Promise<void> {
      await this.loadRecords();
    },

    openEditModal(record: DNSRecord): void {
      this.modalType = '';
      this.validationErrors = {};

      this.selectedRecord = record;
      this.formData = {
        name: record.name,
        type: record.type,
        value: record.value,
        ttl: record.ttl
      };
      this.modalType = 'edit';
    },

    openDeleteModal(record: DNSRecord): void {
      this.modalType = '';
      this.validationErrors = {};
      this.selectedRecord = record;
      this.modalType = 'delete';
    },

    closeModal(): void {
      this.modalType = '';
      this.selectedRecord = null;
      this.validationErrors = {};
    },

    closeDeleteModal(): void {
      this.modalType = '';
      this.selectedRecord = null;
    },

    validateForm(): boolean {
      this.validationErrors = {};

      if (!this.formData.name.trim()) {
        this.validationErrors.name = 'Name is required';
      }

      if (!this.formData.type.trim()) {
        this.validationErrors.type = 'Type is required';
      }

      if (!this.formData.value.trim()) {
        this.validationErrors.value = 'Value is required';
      }

      if (this.formData.ttl < 0) {
        this.validationErrors.ttl = 'TTL must be positive';
      }

      return Object.keys(this.validationErrors).length === 0;
    },

    async saveRecord(): Promise<void> {
      if (!this.selectedRecord || !this.validateForm()) {
        return;
      }

      this.saving = true;
      this.error = '';

      try {
        const updateData: UpdateDNSRecordRequest = {
          name: this.formData.name,
          type: this.formData.type,
          value: this.formData.value,
          ttl: this.formData.ttl
        };
        await api.updateDNSRecord(this.selectedRecord.id, updateData);
        await this.loadRecords();
        this.closeModal();
      } catch (e) {
        console.error('Failed to save DNS record:', e);
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to save DNS record';
      } finally {
        this.saving = false;
      }
    },

    async doDeleteRecord(): Promise<void> {
      if (!this.selectedRecord) return;

      this.deleting = true;
      try {
        await api.deleteDNSRecord(this.selectedRecord.id);
        await this.loadRecords();
        this.closeDeleteModal();
      } catch (e) {
        console.error('Failed to delete DNS record:', e);
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete DNS record';
      } finally {
        this.deleting = false;
      }
    },

    formatDate(dateStr: string): string {
      if (!dateStr) return '-';
      const date = new Date(dateStr);
      return date.toLocaleString();
    },

    formatSyncStatus(status: RecordSyncStatus): string {
      const statusMap: Record<RecordSyncStatus, string> = {
        'synced': 'Synced',
        'pending': 'Pending',
        'failed': 'Failed'
      };
      return statusMap[status] || status;
    }
  };
}
