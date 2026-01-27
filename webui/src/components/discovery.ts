// Discovery Components for Rackd Web UI

import type { DiscoveredDevice, DiscoveryScan, Network } from '../core/types';
import { RackdAPI, RackdAPIError } from '../core/api';

const api = new RackdAPI();

interface DiscoveryListData {
  networks: Network[];
  scans: DiscoveryScan[];
  discoveredDevices: DiscoveredDevice[];
  selectedNetworkId: string;
  loading: boolean;
  error: string;
  pollInterval: ReturnType<typeof setInterval> | null;
  init(): Promise<void>;
  loadNetworks(): Promise<void>;
  loadScans(): Promise<void>;
  loadDiscoveredDevices(): Promise<void>;
  selectNetwork(id: string): void;
  hasActiveScan(): boolean;
  startPolling(): void;
  stopPolling(): void;
  destroy(): void;
}

export function discoveryList() {
  return {
    networks: [] as Network[],
    scans: [] as DiscoveryScan[],
    discoveredDevices: [] as DiscoveredDevice[],
    selectedNetworkId: '',
    loading: true,
    error: '',
    pollInterval: null as ReturnType<typeof setInterval> | null,
    // Scan modal
    showScanModal: false,
    scanNetworkId: '',
    scanType: 'quick' as DiscoveryScan['scan_type'],
    scanning: false,
    // Promote modal
    showPromoteModal: false,
    promoteDevice: null as DiscoveredDevice | null,
    promoteName: '',
    promoting: false,

    openScanModal(): void {
      this.showScanModal = true;
      setTimeout(() => {
        (document.querySelector('[x-show="showScanModal"] select') as HTMLSelectElement)?.focus();
      }, 50);
    },

    openPromoteModal(device: DiscoveredDevice): void {
      this.promoteDevice = device;
      this.promoteName = device.hostname || device.ip;
      this.showPromoteModal = true;
      setTimeout(() => {
        (document.querySelector('[x-show="showPromoteModal"] input[type="text"]') as HTMLInputElement)?.focus();
      }, 50);
    },

    async init(): Promise<void> {
      await this.loadNetworks();
      await Promise.all([this.loadScans(), this.loadDiscoveredDevices()]);
      if (this.hasActiveScan()) this.startPolling();
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
        await api.promoteDevice(this.promoteDevice.id, this.promoteName.trim());
        this.showPromoteModal = false;
        this.promoteDevice = null;
        this.promoteName = '';
        await this.loadDiscoveredDevices();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to promote device';
      } finally {
        this.promoting = false;
      }
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
