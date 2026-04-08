// DNS Management Components

import type {
  DNSProvider, DNSProviderType, DNSProviderFilter,
  CreateDNSProviderRequest, UpdateDNSProviderRequest,
  DNSZone, DNSZoneFilter,
  CreateDNSZoneRequest, UpdateDNSZoneRequest,
  DNSRecord, DNSRecordFilter, UpdateDNSRecordRequest,
  SyncStatus, RecordSyncStatus, Network, Device, Datacenter
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

  // Form data - flat properties for CSP compatibility
  providerName: string;
  providerType: DNSProviderType;
  providerEndpoint: string;
  providerToken: string;
  providerDescription: string;
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
  getInputClass(hasError: boolean): string;
  getAriaDescribedBy(errorId: string, hasError: boolean): string | undefined;

  // Utilities
  formatDate(dateStr: string): string;
  formatType(type: DNSProviderType): string;
  hasProviders(): boolean;
  getProviderAriaLabel(providerName: string, action: string): string;
  getProviderDescription(provider: DNSProvider): string;
  getEditInputId(id: string, prefix: string): string;
  getSelectedProviderName(): string;
  getTestResultMessage(): string;
  isTestResultSuccess(): boolean;
  getTestResultClass(): string;
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

    providerName: '',
    providerType: 'technitium',
    providerEndpoint: '',
    providerToken: '',
    providerDescription: '',
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

      this.providerName = '';
      this.providerType = 'technitium';
      this.providerEndpoint = '';
      this.providerToken = '';
      this.providerDescription = '';
      this.modalType = 'create';
    },

    openEditModal(provider: DNSProvider): void {
      this.modalType = '';
      this.validationErrors = {};
      this.testResult = null;

      this.selectedProvider = provider;
      this.providerName = provider.name;
      this.providerType = provider.type;
      this.providerEndpoint = provider.endpoint;
      this.providerToken = ''; // Never populate token for security
      this.providerDescription = provider.description || '';
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
      this.providerName = '';
      this.providerType = 'technitium';
      this.providerEndpoint = '';
      this.providerToken = '';
      this.providerDescription = '';
    },

    closeDeleteModal(): void {
      this.modalType = '';
      this.selectedProvider = null;
    },

    getProviderAriaLabel(providerName: string, action: string): string {
      return action + ' ' + providerName;
    },

    getProviderDescription(provider: DNSProvider): string {
      return provider.description || '-';
    },

    getEditInputId(id: string, prefix: string): string {
      return prefix + '-' + id;
    },

    getInputClass(hasError: boolean): string {
      const base = 'w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-50 focus:outline-none focus:ring-[3px] focus:ring-blue-500 min-h-[44px]';
      return hasError ? `${base} border-red-600 dark:border-red-400` : `${base} border-gray-300 dark:border-gray-600`;
    },

    getAriaDescribedBy(errorId: string, hasError: boolean): string | undefined {
      return hasError ? errorId : undefined;
    },

    validateForm(): boolean {
      this.validationErrors = {};

      if (!this.providerName.trim()) {
        this.validationErrors.name = 'Name is required';
      }

      if (!this.providerEndpoint.trim()) {
        this.validationErrors.endpoint = 'Endpoint is required';
      }

      // Token required for create, optional for edit (user may leave blank to keep existing)
      if (!this.selectedProvider && !this.providerToken.trim()) {
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
            name: this.providerName,
            endpoint: this.providerEndpoint,
            description: this.providerDescription || undefined
          };
          // Only include token if provided
          if (this.providerToken.trim()) {
            updateData.token = this.providerToken;
          }
          await api.updateDNSProvider(this.selectedProvider.id, updateData);
        } else {
          // Create new provider
          const createData: CreateDNSProviderRequest = {
            name: this.providerName,
            type: this.providerType,
            endpoint: this.providerEndpoint,
            token: this.providerToken,
            description: this.providerDescription || undefined
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
    },

    hasProviders(): boolean {
      return this.providers.length > 0;
    },

    getSelectedProviderName(): string {
      return this.selectedProvider ? this.selectedProvider.name : '';
    },

    getTestResultMessage(): string {
      return this.testResult ? this.testResult.message : '';
    },

    isTestResultSuccess(): boolean {
      return this.testResult ? this.testResult.success : false;
    },

    getTestResultClass(): string {
      return this.isTestResultSuccess()
        ? 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-200'
        : 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-200';
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
  getInputClass(hasError: boolean): string;
  getAriaDescribedBy(errorId: string, hasError: boolean): string | undefined;

  // Utilities
  formatDate(dateStr: string): string;
  getProviderName(id: string): string;
  getNetworkName(id: string): string;
  formatSyncStatus(status?: SyncStatus): string;
  hasZones(): boolean;
  getAutoSyncClass(autoSync: boolean): string;
  getSyncStatusClass(status?: SyncStatus): string;
  getRecordsUrl(zoneId: string): string;
  getZoneAriaLabel(zoneName: string, action: string): string;
  getEditInputId(id: string, prefix: string): string;
  getSelectedZoneName(): string;
  getSyncResultMessage(): string;
  getSyncResultDetails(): string;
  isSyncResultSuccess(): boolean;
  getSyncResultClass(): string;
  getAutoSyncLabel(autoSync: boolean): string;
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
          message: `Imported: ${result.imported}, Linked: ${result.linked || 0}, Skipped: ${result.skipped}`,
          details: result.failed > 0 ? `Failed: ${result.failed}` : undefined
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
    },

    hasZones(): boolean {
      return this.zones.length > 0;
    },

    getAutoSyncClass(autoSync: boolean): string {
      return autoSync
        ? 'bg-green-100 text-green-900 dark:bg-green-900 dark:text-green-200'
        : 'bg-gray-100 text-gray-900 dark:bg-gray-700 dark:text-gray-200';
    },

    getSyncStatusClass(status?: SyncStatus): string {
      if (!status) return 'bg-gray-100 text-gray-900 dark:bg-gray-700 dark:text-gray-200';
      const classes: Record<SyncStatus, string> = {
        'success': 'bg-green-100 text-green-900 dark:bg-green-900 dark:text-green-200',
        'failed': 'bg-red-100 text-red-900 dark:bg-red-900 dark:text-red-200',
        'partial': 'bg-yellow-100 text-yellow-900 dark:bg-yellow-900 dark:text-yellow-200'
      };
      return classes[status] || 'bg-gray-100 text-gray-900 dark:bg-gray-700 dark:text-gray-200';
    },

    getRecordsUrl(zoneId: string): string {
      return '/dns/records/' + zoneId;
    },

    getZoneAriaLabel(zoneName: string, action: string): string {
      return action + ' ' + zoneName;
    },

    getEditInputId(id: string, prefix: string): string {
      return prefix + '-' + id;
    },

    getInputClass(hasError: boolean): string {
      const base = 'w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-50 focus:outline-none focus:ring-[3px] focus:ring-blue-500 min-h-[44px]';
      return hasError ? `${base} border-red-600 dark:border-red-400` : `${base} border-gray-300 dark:border-gray-600`;
    },

    getAriaDescribedBy(errorId: string, hasError: boolean): string | undefined {
      return hasError ? errorId : undefined;
    },

    getSelectedZoneName(): string {
      return this.selectedZone ? this.selectedZone.name : '';
    },

    getSyncResultMessage(): string {
      return this.syncResult ? this.syncResult.message : '';
    },

    getSyncResultDetails(): string {
      return this.syncResult?.details || '';
    },

    isSyncResultSuccess(): boolean {
      return this.syncResult ? this.syncResult.success : false;
    },

    getSyncResultClass(): string {
      return (this.syncResult && this.syncResult.success)
        ? 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-200'
        : 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-200';
    },

    getAutoSyncLabel(autoSync: boolean): string {
      return autoSync ? 'Enabled' : 'Disabled';
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

type RecordModalType = '' | 'edit' | 'delete' | 'link' | 'promote';

interface DNSRecordsData {
  records: DNSRecord[];
  zoneId: string;
  zone: DNSZone | null;
  loading: boolean;
  error: string;

  // Filters
  filterType: string;
  filterSyncStatus: string;
  filterLinkStatus: string;

  // Device data for resolving names
  devices: Device[];

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

  // Link modal state
  linkDeviceSearch: string;
  linkSelectedDeviceId: string;
  linkSelectedAddressId: string;
  linkAddToDomains: boolean;
  linkError: string;
  linking: boolean;

  // Promote modal state
  promoteDeviceName: string;
  promoteDatacenterId: string;
  promoteTags: string;
  promoteError: string;
  promoting: boolean;
  datacenters: Datacenter[];

  // Computed properties for template compatibility
  get showEditModal(): boolean;
  get showDeleteModal(): boolean;
  get showLinkModal(): boolean;
  get showPromoteModal(): boolean;

  init(): Promise<void>;
  loadRecords(): Promise<void>;
  loadZone(): Promise<void>;
  loadDevices(): Promise<void>;
  loadDatacenters(): Promise<void>;
  applyFilters(): Promise<void>;

  // Modal management
  openEditModal(record: DNSRecord): void;
  openDeleteModal(record: DNSRecord): void;
  openLinkModal(record: DNSRecord): void;
  openPromoteModal(record: DNSRecord): void;
  closeModal(): void;
  closeDeleteModal(): void;
  closeLinkModal(): void;
  closePromoteModal(): void;

  // CRUD operations
  saveRecord(): Promise<void>;
  doDeleteRecord(): Promise<void>;
  doLinkRecord(): Promise<void>;
  doPromoteRecord(): Promise<void>;

  // Link modal helpers
  get filteredDevices(): Device[];
  get selectedDeviceAddresses(): { id: string; ip: string }[];
  get showAddressDropdown(): boolean;
  get showAddToDomains(): boolean;

  // Form helpers
  validateForm(): boolean;
  getInputClass(hasError: boolean, additionalClasses?: string): string;
  getAriaDescribedBy(errorId: string, hasError: boolean): string | undefined;

  // Utilities
  formatDate(dateStr: string): string;
  formatSyncStatus(status: RecordSyncStatus): string;
  getDeviceName(deviceId: string): string;
  getAddressValue(deviceId: string, addressId: string): string;
  hasRecords(): boolean;
  getSyncStatusClass(status: RecordSyncStatus): string;
  getDeviceUrl(deviceId: string): string;
  getRecordAriaLabel(recordName: string, action: string): string;
  getEditInputId(id: string, prefix: string): string;
  getZoneName(): string;
  getAddressValueSafe(deviceId: string, addressId: string): string;
  getSelectedRecordName(): string;
  getSelectedRecordType(): string;
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
    filterLinkStatus: '',

    devices: [],

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

    linkDeviceSearch: '',
    linkSelectedDeviceId: '',
    linkSelectedAddressId: '',
    linkAddToDomains: false,
    linkError: '',
    linking: false,

    promoteDeviceName: '',
    promoteDatacenterId: '',
    promoteTags: '',
    promoteError: '',
    promoting: false,
    datacenters: [],

    get showEditModal(): boolean { return this.modalType === 'edit'; },
    get showDeleteModal(): boolean { return this.modalType === 'delete'; },
    get showLinkModal(): boolean { return this.modalType === 'link'; },
    get showPromoteModal(): boolean { return this.modalType === 'promote'; },

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
        this.loadZone(),
        this.loadDevices(),
        this.loadDatacenters()
      ]);
    },

    async loadRecords(): Promise<void> {
      this.loading = true;
      try {
        const filter: DNSRecordFilter = { zone_id: this.zoneId };
        if (this.filterType) filter.type = this.filterType;
        if (this.filterSyncStatus) filter.sync_status = this.filterSyncStatus as RecordSyncStatus;
        if (this.filterLinkStatus) filter.link_status = this.filterLinkStatus;
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

    async loadDevices(): Promise<void> {
      try {
        this.devices = await api.listDevices();
      } catch (e) {
        console.error('Failed to load devices:', e);
      }
    },

    async loadDatacenters(): Promise<void> {
      try {
        this.datacenters = await api.listDatacenters();
      } catch (e) {
        console.error('Failed to load datacenters:', e);
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

    openLinkModal(record: DNSRecord): void {
      this.modalType = '';
      this.validationErrors = {};
      this.linkError = '';
      this.linkDeviceSearch = '';
      this.linkSelectedDeviceId = '';
      this.linkSelectedAddressId = '';
      this.linkAddToDomains = false;
      this.selectedRecord = record;
      this.modalType = 'link';
    },

    closeLinkModal(): void {
      this.modalType = '';
      this.selectedRecord = null;
      this.linkError = '';
      this.linkDeviceSearch = '';
      this.linkSelectedDeviceId = '';
      this.linkSelectedAddressId = '';
      this.linkAddToDomains = false;
    },

    openPromoteModal(record: DNSRecord): void {
      this.modalType = '';
      this.promoteError = '';
      this.promoteDeviceName = record.name + '.' + (this.zone?.name || '');
      this.promoteDatacenterId = '';
      this.promoteTags = '';
      this.selectedRecord = record;
      this.modalType = 'promote';
    },

    closePromoteModal(): void {
      this.modalType = '';
      this.selectedRecord = null;
      this.promoteError = '';
      this.promoteDeviceName = '';
      this.promoteDatacenterId = '';
      this.promoteTags = '';
    },

    hasRecords(): boolean {
      return this.records.length > 0;
    },

    getSyncStatusClass(status: RecordSyncStatus): string {
      const classes: Record<RecordSyncStatus, string> = {
        'synced': 'bg-green-100 text-green-900 dark:bg-green-900 dark:text-green-200',
        'pending': 'bg-yellow-100 text-yellow-900 dark:bg-yellow-900 dark:text-yellow-200',
        'failed': 'bg-red-100 text-red-900 dark:bg-red-900 dark:text-red-200'
      };
      return classes[status] || 'bg-gray-100 text-gray-900 dark:bg-gray-700 dark:text-gray-200';
    },

    getDeviceUrl(deviceId: string): string {
      return '/devices/detail?id=' + deviceId;
    },

    getRecordAriaLabel(recordName: string, action: string): string {
      return action + ' record ' + recordName;
    },

    getEditInputId(id: string, prefix: string): string {
      return prefix + '-' + id;
    },

    getInputClass(hasError: boolean, additionalClasses?: string): string {
      const base = 'w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-50 focus:outline-none focus:ring-[3px] focus:ring-blue-500 min-h-[44px]';
      const classes = hasError ? `${base} border-red-600 dark:border-red-400` : `${base} border-gray-300 dark:border-gray-600`;
      return additionalClasses ? `${classes} ${additionalClasses}` : classes;
    },

    getAriaDescribedBy(errorId: string, hasError: boolean): string | undefined {
      return hasError ? errorId : undefined;
    },

    get filteredDevices(): Device[] {
      if (!this.linkDeviceSearch.trim()) return this.devices;
      const search = this.linkDeviceSearch.toLowerCase();
      return this.devices.filter(d => d.name.toLowerCase().includes(search));
    },

    get selectedDeviceAddresses(): { id: string; ip: string }[] {
      if (!this.linkSelectedDeviceId) return [];
      const device = this.devices.find(d => d.id === this.linkSelectedDeviceId);
      if (!device || !device.addresses) return [];
      return device.addresses.filter(a => a.id).map(a => ({ id: a.id!, ip: a.ip }));
    },

    get showAddressDropdown(): boolean {
      if (!this.selectedRecord) return false;
      const type = this.selectedRecord.type;
      return (type === 'A' || type === 'AAAA' || type === 'PTR') && this.linkSelectedDeviceId !== '';
    },

    get showAddToDomains(): boolean {
      if (!this.selectedRecord) return false;
      return this.selectedRecord.type === 'CNAME';
    },

    async doLinkRecord(): Promise<void> {
      if (!this.selectedRecord || !this.linkSelectedDeviceId) return;

      this.linking = true;
      this.linkError = '';

      try {
        const req: { device_id: string; address_id?: string; add_to_domains?: boolean } = {
          device_id: this.linkSelectedDeviceId
        };
        if (this.linkSelectedAddressId) {
          req.address_id = this.linkSelectedAddressId;
        }
        if (this.selectedRecord.type === 'CNAME' && this.linkAddToDomains) {
          req.add_to_domains = true;
        }
        await api.linkDNSRecord(this.selectedRecord.id, req);
        await this.loadRecords();
        this.closeLinkModal();
      } catch (e) {
        console.error('Failed to link DNS record:', e);
        this.linkError = e instanceof RackdAPIError ? e.message : 'Failed to link DNS record';
      } finally {
        this.linking = false;
      }
    },

    async doPromoteRecord(): Promise<void> {
      if (!this.selectedRecord) return;

      this.promoting = true;
      this.promoteError = '';

      try {
        const req: { name?: string; datacenter_id?: string; tags?: string[] } = {};
        if (this.promoteDeviceName.trim()) {
          req.name = this.promoteDeviceName.trim();
        }
        if (this.promoteDatacenterId) {
          req.datacenter_id = this.promoteDatacenterId;
        }
        if (this.promoteTags.trim()) {
          req.tags = this.promoteTags.split(',').map(t => t.trim()).filter(t => t);
        }
        await api.promoteDNSRecord(this.selectedRecord.id, req);
        await this.loadRecords();
        this.closePromoteModal();
      } catch (e) {
        console.error('Failed to promote DNS record:', e);
        this.promoteError = e instanceof RackdAPIError ? e.message : 'Failed to promote DNS record';
      } finally {
        this.promoting = false;
      }
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
    },

    getDeviceName(deviceId: string): string {
      if (!deviceId) return '';
      const device = this.devices.find(d => d.id === deviceId);
      return device ? device.name : deviceId;
    },

    getAddressValue(deviceId: string, addressId: string): string {
      if (!deviceId || !addressId) return '';
      const device = this.devices.find(d => d.id === deviceId);
      if (!device || !device.addresses) return '';
      const address = device.addresses.find(a => a.id === addressId);
      return address ? address.ip : '';
    },

    getZoneName(): string {
      return this.zone ? (this.zone as DNSZone).name : '';
    },

    getAddressValueSafe(deviceId: string, addressId: string): string {
      return this.getAddressValue(deviceId, addressId) || '';
    },

    getSelectedRecordName(): string {
      return this.selectedRecord ? this.selectedRecord.name : '';
    },

    getSelectedRecordType(): string {
      return this.selectedRecord ? this.selectedRecord.type : '';
    }
  };
}
