// Discovery Components for Rackd Web UI

import type { DiscoveredDevice, DiscoveryScan, Network, Datacenter, Device } from '../core/types';
import { api, RackdAPIError } from '../core/api';

interface DiscoveryListData {
  networks: Network[];
  scans: DiscoveryScan[];
  discoveredDevices: DiscoveredDevice[];
  datacenters: Datacenter[];
  selectedNetworkId: string;
  loading: boolean;
  error: string;
  pollInterval: ReturnType<typeof setInterval> | null;
  init(): Promise<void>;
  loadNetworks(): Promise<void>;
  loadScans(): Promise<void>;
  loadDiscoveredDevices(): Promise<void>;
  loadDatacenters(): Promise<void>;
  selectNetwork(id: string): void;
  hasActiveScan(): boolean;
  startPolling(): void;
  stopPolling(): void;
  destroy(): void;
  formatDate(date: string): string;
  getPromoteVendor(): string;
  getPromoteIP(): string;
  getPromoteMAC(): string;
  getPromoteHostname(): string;
  getPromoteOS(): string;
  hasMAC(): boolean;
  hasHostname(): boolean;
  hasOS(): boolean;
  hasVendor(): boolean;
  getOpenPorts(): number[];
  getServices(): any[];
  getConfidenceValue(): number;
  hasScans(): boolean;
  hasDiscoveredDevices(): boolean;
  hasScanProgress(scan: any): boolean;
  hasScanHosts(scan: any): boolean;
  hasFoundHosts(scan: any): boolean;
  hasPortCount(device: any, min: number): boolean;
  getPortRemainingCount(device: any, limit: number): number;
  hasServiceCount(device: any, min: number): boolean;
  getServiceRemainingCount(device: any, limit: number): number;
  getConfidenceBadgeClass(confidence: number): any;
  getScanStatusClass(status: string): any;
  getDiscoveredDevicesCountLabel(): string;
  hasOldScans(): boolean;
  getPromoteVendorPlaceholder(): string;
  getNetworkOptionLabel(network: Network): string;
  getScanProgressLabel(scan: DiscoveryScan): string;
  getScanHostsLabel(scan: DiscoveryScan): string;
  getFoundHostsLabel(scan: DiscoveryScan): string;
  getHostnameLabel(device: DiscoveredDevice): string;
  getMacAddressLabel(device: DiscoveredDevice): string;
  getOsGuessLabel(device: DiscoveredDevice): string;
  getVendorLabel(device: DiscoveredDevice): string;
  getLimitedPorts(device: DiscoveredDevice, limit: number): number[];
  getLimitedServices(device: DiscoveredDevice, limit: number): any[];
  getServiceTitle(service: any): string;
  getConfidenceLabel(device: DiscoveredDevice | null): string;
  getServiceLabel(service: any): string;
}

export function discoveryList() {
  return {
    networks: [] as Network[],
    scans: [] as DiscoveryScan[],
    discoveredDevices: [] as DiscoveredDevice[],
    datacenters: [] as Datacenter[],
    selectedNetworkId: '',
    loading: true,
    error: '',
    pollInterval: null as ReturnType<typeof setInterval> | null,
    deviceFilter: '',
    // Scan modal
    showScanModal: false,
    scanNetworkId: '',
    scanType: 'quick' as DiscoveryScan['scan_type'],
    scanning: false,
    // Get scan types
    get scanTypes() {
      return [
        { value: 'quick', label: 'Quick', description: 'ICMP ping' },
        { value: 'full', label: 'Full', description: 'TCP port scan' },
        { value: 'deep', label: 'Deep', description: 'Comprehensive scan with SNMP/SSH' },
      ];
    },
    // Promote modal
    showPromoteModal: false,
    promoteDevice: null as DiscoveredDevice | null,
    promoteName: '',
    promoteDatacenterId: '',
    promoteMakeModel: '',
    promoting: false,
    // Cancel scan state
    cancellingScans: new Set<string>(),
    // Computed property for filtered devices
    get filteredDevices() {
      const filter = this.deviceFilter.toLowerCase().trim();
      if (!filter) {
        return this.discoveredDevices;
      }
      return this.discoveredDevices.filter(d =>
        d.ip.toLowerCase().includes(filter) ||
        (d.hostname && d.hostname.toLowerCase().includes(filter)) ||
        (d.mac_address && d.mac_address.toLowerCase().includes(filter)) ||
        (d.os_guess && d.os_guess.toLowerCase().includes(filter)) ||
        (d.vendor && d.vendor.toLowerCase().includes(filter))
      );
    },

    openScanModal(): void {
      this.showScanModal = true;
      setTimeout(() => {
        (document.querySelector('[x-show="showScanModal"] select') as HTMLSelectElement)?.focus();
      }, 50);
    },

    openPromoteModal(device: DiscoveredDevice): void {
      this.promoteDevice = device;
      this.promoteName = device.hostname || device.ip;
      this.promoteDatacenterId = '';
      this.promoteMakeModel = device.vendor || '';
      this.showPromoteModal = true;
      setTimeout(() => {
        (document.querySelector('[x-show="showPromoteModal"] input[type="text"]') as HTMLInputElement)?.focus();
      }, 50);
    },

    async init(): Promise<void> {
      await Promise.all([this.loadNetworks(), this.loadDatacenters()]);
      await Promise.all([this.loadScans(), this.loadDiscoveredDevices()]);
      if (this.hasActiveScan()) this.startPolling();
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
      } catch (e) {
        this.networks = [];
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load networks';
      } finally {
        this.loading = false;
      }
    },

    async loadScans(): Promise<void> {
      try {
        this.scans = (await api.listScans(this.selectedNetworkId || undefined)) || [];
      } catch {
        this.scans = [];
      }
    },

    async loadDiscoveredDevices(): Promise<void> {
      try {
        this.discoveredDevices = (await api.listDiscoveredDevices(this.selectedNetworkId || undefined)) || [];
      } catch {
        this.discoveredDevices = [];
      }
    },

    selectNetwork(id: string): void {
      this.selectedNetworkId = id;
      this.loadScans();
      this.loadDiscoveredDevices();
    },

    hasActiveScan(): boolean {
      return this.scans.some((s) => s.status === 'pending' || s.status === 'running');
    },

    startPolling(): void {
      if (this.pollInterval) return;
      this.pollInterval = setInterval(async () => {
        await this.loadScans();
        await this.loadDiscoveredDevices();
        if (!this.hasActiveScan()) this.stopPolling();
      }, 3000);
    },

    stopPolling(): void {
      if (this.pollInterval) {
        clearInterval(this.pollInterval);
        this.pollInterval = null;
      }
    },

    destroy(): void {
      this.stopPolling();
    },

    async startScan(): Promise<void> {
      if (!this.scanNetworkId) {
        this.error = 'Please select a network';
        return;
      }
      this.scanning = true;
      this.error = '';
      try {
        await api.startScan(this.scanNetworkId, this.scanType);
        this.showScanModal = false;
        this.scanNetworkId = '';
        await this.loadScans();
        if (this.hasActiveScan()) this.startPolling();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to start scan';
      } finally {
        this.scanning = false;
      }
    },

    async doPromote(): Promise<void> {
      if (!this.promoteDevice || !this.promoteName.trim()) {
        this.error = 'Name is required';
        return;
      }
      this.promoting = true;
      this.error = '';
      try {
        await api.promoteDevice(
          this.promoteDevice.id,
          this.promoteName.trim(),
          this.promoteDatacenterId || undefined,
          this.promoteMakeModel || undefined
        );
        this.showPromoteModal = false;
        this.promoteDevice = null;
        this.promoteName = '';
        this.promoteDatacenterId = '';
        this.promoteMakeModel = '';
        await this.loadDiscoveredDevices();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to promote device';
      } finally {
        this.promoting = false;
      }
    },

    formatDate(date: string | undefined): string {
      if (!date) return '-';
      const d = new Date(date);
      const now = new Date();
      const diffMs = now.getTime() - d.getTime();
      const diffMins = Math.floor(diffMs / 60000);
      const diffHours = Math.floor(diffMins / 60);
      const diffDays = Math.floor(diffHours / 24);

      if (diffMins < 1) return 'Just now';
      if (diffMins < 60) return `${diffMins}m ago`;
      if (diffHours < 24) return `${diffHours}h ago`;
      if (diffDays < 7) return `${diffDays}d ago`;
      return d.toLocaleDateString();
    },

    async deleteDevice(deviceId: string): Promise<void> {
      if (!confirm('Delete this discovered device?')) return;
      this.error = '';
      try {
        await api.deleteDiscoveredDevice(deviceId);
        await this.loadDiscoveredDevices();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete device';
      }
    },

    async deleteScan(scanId: string): Promise<void> {
      this.error = '';
      try {
        await api.deleteScan(scanId);
        await this.loadScans();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete scan';
      }
    },

    async deleteOldScans(): Promise<void> {
      const oldScans = this.scans.filter((s) => s.status === 'completed' || s.status === 'failed');
      if (!confirm(`Delete ${oldScans.length} completed/failed scan(s)?`)) return;
      this.error = '';
      try {
        await Promise.all(oldScans.map((s) => api.deleteScan(s.id)));
        await this.loadScans();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete scans';
      }
    },

    async deleteAllDevices(): Promise<void> {
      const message = this.selectedNetworkId
        ? `Delete all ${this.discoveredDevices.length} discovered devices${this.selectedNetworkId ? ' in this network' : ''}?`
        : `Delete all ${this.discoveredDevices.length} discovered devices globally?`;

      if (!confirm(message)) return;
      this.error = '';
      try {
        // If network selected, pass network_id. If not, no query param = delete all
        await api.deleteDiscoveredDevicesByNetwork(this.selectedNetworkId || '');
        await this.loadDiscoveredDevices();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete devices';
      }
    },

    isCancelling(scanId: string): boolean {
      return this.cancellingScans.has(scanId);
    },

    async cancelScan(scanId: string): Promise<void> {
      if (!confirm('Stop this scan? Progress will be lost.')) return;
      this.error = '';
      this.cancellingScans.add(scanId);
      try {
        await api.cancelScan(scanId);
        await this.loadScans();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to stop scan';
      } finally {
        this.cancellingScans.delete(scanId);
      }
    },

    hasOpenPorts(): boolean {
      return !!(this.promoteDevice && this.promoteDevice.open_ports && this.promoteDevice.open_ports.length > 0);
    },
    hasServices(): boolean {
      return !!(this.promoteDevice && this.promoteDevice.services && this.promoteDevice.services.length > 0);
    },
    hasPromoted(): boolean {
      return !!(this.promoteDevice && this.promoteDevice.promoted_to_device_id);
    },
    getConfidenceStyle(): string {
      const conf = this.promoteDevice?.confidence || 0;
      return `width: ${conf * 10}%`;
    },
    getConfidenceClass(): string {
      const conf = this.promoteDevice?.confidence || 0;
      if (conf >= 8) return 'bg-green-500';
      if (conf >= 5) return 'bg-yellow-500';
      return 'bg-gray-500';
    },
    getPromoteVendorPlaceholder(): string {
      return (this.promoteDevice && this.promoteDevice.vendor) || 'e.g., Dell PowerEdge R640';
    },

    getPromoteVendor(): string {
      return (this.promoteDevice && this.promoteDevice.vendor) || '-';
    },

    getPromoteIP(): string {
      return this.promoteDevice?.ip || '-';
    },

    getPromoteMAC(): string {
      return this.promoteDevice?.mac_address || '-';
    },

    getPromoteHostname(): string {
      return this.promoteDevice?.hostname || '-';
    },

    getPromoteOS(): string {
      return this.promoteDevice?.os_guess || '-';
    },

    getPromoteFirstSeen(): string | undefined {
      return this.promoteDevice?.first_seen;
    },

    getPromoteLastSeen(): string | undefined {
      return this.promoteDevice?.last_seen;
    },

    hasMAC(): boolean {
      return !!(this.promoteDevice && this.promoteDevice.mac_address);
    },

    hasHostname(): boolean {
      return !!(this.promoteDevice && this.promoteDevice.hostname);
    },

    hasOS(): boolean {
      return !!(this.promoteDevice && this.promoteDevice.os_guess);
    },

    hasVendor(): boolean {
      return !!(this.promoteDevice && this.promoteDevice.vendor);
    },

    getOpenPorts(): number[] {
      return this.promoteDevice?.open_ports || [];
    },

    getServices(): any[] {
      return this.promoteDevice?.services || [];
    },

    getConfidenceValue(): number {
      return this.promoteDevice?.confidence || 0;
    },
    hasScans(): boolean {
      return this.scans.length > 0;
    },
    hasDiscoveredDevices(): boolean {
      return this.discoveredDevices.length > 0;
    },
    hasScanProgress(scan: any): boolean {
      return !!(scan && scan.progress_percent > 0);
    },
    hasScanHosts(scan: any): boolean {
      return !!(scan && scan.scanned_hosts > 0);
    },
    hasFoundHosts(scan: any): boolean {
      return !!(scan && scan.found_hosts > 0);
    },
    hasPortCount(device: any, min: number): boolean {
      return !!(device && device.open_ports && device.open_ports.length > min);
    },
    getPortRemainingCount(device: any, limit: number): number {
      if (!device || !device.open_ports) return 0;
      return device.open_ports.length - limit;
    },
    hasServiceCount(device: any, min: number): boolean {
      return !!(device && device.services && device.services.length > min);
    },
    getServiceRemainingCount(device: any, limit: number): number {
      if (!device || !device.services) return 0;
      return device.services.length - limit;
    },
    getConfidenceBadgeClass(conf: number): any {
      return {
        'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400': conf >= 8,
        'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400': conf >= 5 && conf < 8,
        'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400': conf < 5
      };
    },
    getScanStatusClass(status: string): any {
      return {
        'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400': status === 'pending',
        'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400': status === 'running',
        'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400': status === 'completed',
        'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400': status === 'failed'
      };
    },
    getDiscoveredDevicesCountLabel(): string {
      return this.discoveredDevices.length + ' devices found';
    },
    hasOldScans(): boolean {
      return this.scans.some((s) => s.status === 'completed' || s.status === 'failed');
    },
    getNetworkOptionLabel(n: Network): string {
      return `${n.name} (${n.subnet})`;
    },
    getScanProgressLabel(s: DiscoveryScan): string {
      return `${Math.round(s.progress_percent)}%`;
    },
    getScanHostsLabel(s: DiscoveryScan): string {
      return `${s.scanned_hosts}/${s.total_hosts} hosts`;
    },
    getFoundHostsLabel(s: DiscoveryScan): string {
      return `${s.found_hosts} found`;
    },
    getHostnameLabel(d: DiscoveredDevice): string {
      return d.hostname || '-';
    },
    getMacAddressLabel(d: DiscoveredDevice): string {
      return d.mac_address || '-';
    },
    getOsGuessLabel(d: DiscoveredDevice): string {
      return d.os_guess || '-';
    },
    getVendorLabel(d: DiscoveredDevice): string {
      return d.vendor || '-';
    },
    getLimitedPorts(d: DiscoveredDevice, limit: number): number[] {
      return (d.open_ports || []).slice(0, limit);
    },
    getLimitedServices(d: DiscoveredDevice, limit: number): any[] {
      return (d.services || []).slice(0, limit);
    },
    getServiceTitle(svc: any): string {
      if (!svc) return '';
      return `${svc.protocol}/${svc.port}${svc.version ? ' ' + svc.version : ''}`;
    },
    getConfidenceLabel(d: DiscoveredDevice | null): string {
      const conf = d ? d.confidence : 0;
      return `${conf}/10`;
    },
    getServiceLabel(svc: any): string {
      if (!svc) return '';
      return `${svc.protocol}/${svc.port}`;
    },
  };
}

interface ScanFormData {
  networkId: string;
  scanType: DiscoveryScan['scan_type'];
  starting: boolean;
  error: string;
  init(): void;
  start(): Promise<void>;
}

export function scanForm(): ScanFormData {
  return {
    networkId: '',
    scanType: 'quick',
    starting: false,
    error: '',

    init(): void {
      this.networkId = new URLSearchParams(window.location.search).get('network_id') || '';
    },

    async start(): Promise<void> {
      if (!this.networkId) {
        this.error = 'Please select a network';
        return;
      }
      this.starting = true;
      this.error = '';
      try {
        await api.startScan(this.networkId, this.scanType);
        window.location.href = '/discovery';
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to start scan';
      } finally {
        this.starting = false;
      }
    },
  };
}

interface ScanDetailData {
  scan: DiscoveryScan | null;
  network: Network | null;
  loading: boolean;
  error: string;
  pollInterval: ReturnType<typeof setInterval> | null;
  init(): Promise<void>;
  loadScan(): Promise<void>;
  loadNetwork(): Promise<void>;
  startPolling(): void;
  stopPolling(): void;
  destroy(): void;
}

export function scanDetail(): ScanDetailData {
  return {
    scan: null,
    network: null,
    loading: true,
    error: '',
    pollInterval: null,

    async init(): Promise<void> {
      const id = new URLSearchParams(window.location.search).get('id');
      if (!id) {
        this.error = 'No scan ID provided';
        this.loading = false;
        return;
      }
      await this.loadScan();
      if (this.scan && (this.scan.status === 'pending' || this.scan.status === 'running')) {
        this.startPolling();
      }
    },

    async loadScan(): Promise<void> {
      const id = new URLSearchParams(window.location.search).get('id');
      if (!id) return;
      this.loading = true;
      try {
        this.scan = await api.getScan(id);
        await this.loadNetwork();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load scan';
      } finally {
        this.loading = false;
      }
    },

    async loadNetwork(): Promise<void> {
      if (!this.scan) return;
      try {
        this.network = await api.getNetwork(this.scan.network_id);
      } catch {
        // Non-critical
      }
    },

    startPolling(): void {
      if (this.pollInterval) return;
      this.pollInterval = setInterval(async () => {
        await this.loadScan();
        if (this.scan && this.scan.status !== 'pending' && this.scan.status !== 'running') {
          this.stopPolling();
        }
      }, 2000);
    },

    stopPolling(): void {
      if (this.pollInterval) {
        clearInterval(this.pollInterval);
        this.pollInterval = null;
      }
    },

    destroy(): void {
      this.stopPolling();
    },
  };
}

interface PromoteFormData {
  device: DiscoveredDevice | null;
  name: string;
  loading: boolean;
  promoting: boolean;
  error: string;
  init(): Promise<void>;
  promote(): Promise<void>;
  cancel(): void;
}

export function promoteForm(): PromoteFormData {
  return {
    device: null,
    name: '',
    loading: true,
    promoting: false,
    error: '',

    async init(): Promise<void> {
      const id = new URLSearchParams(window.location.search).get('id');
      if (!id) {
        this.error = 'No device ID provided';
        this.loading = false;
        return;
      }
      try {
        const devices = (await api.listDiscoveredDevices()) || [];
        this.device = devices.find((d) => d.id === id) || null;
        if (this.device) {
          this.name = this.device.hostname || this.device.ip;
        }
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load device';
      } finally {
        this.loading = false;
      }
    },

    async promote(): Promise<void> {
      if (!this.device || !this.name.trim()) {
        this.error = 'Name is required';
        return;
      }
      this.promoting = true;
      this.error = '';
      try {
        await api.promoteDevice(this.device.id, this.name.trim());
        window.location.href = '/discovery';
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to promote device';
      } finally {
        this.promoting = false;
      }
    },

    cancel(): void {
      window.location.href = '/discovery';
    },
  };
}
