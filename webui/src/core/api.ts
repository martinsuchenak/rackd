// API Client for Rackd - No DOM dependencies (mobile-ready)

import type {
  Address,
  APIError,
  Datacenter,
  Device,
  DeviceFilter,
  DeviceRelationship,
  DiscoveredDevice,
  DiscoveryRule,
  DiscoveryScan,
  IPStatus,
  Network,
  NetworkPool,
  NetworkUtilization,
  UIConfig,
} from './types';

export type {
  Address,
  APIError,
  Datacenter,
  Device,
  DeviceFilter,
  DeviceRelationship,
  DiscoveredDevice,
  DiscoveryRule,
  DiscoveryScan,
  IPStatus,
  NavItem,
  Network,
  NetworkPool,
  NetworkUtilization,
  ServiceInfo,
  UIConfig,
  UserInfo,
} from './types';

export class RackdAPIError extends Error {
  constructor(
    public code: string,
    message: string,
    public details?: Record<string, unknown>
  ) {
    super(message);
    this.name = 'RackdAPIError';
  }
}

export interface RackdAPIOptions {
  baseURL?: string;
  token?: string;
}

export class RackdAPI {
  private baseURL: string;
  private token?: string;

  constructor(options: RackdAPIOptions = {}) {
    this.baseURL = options.baseURL || '';
    this.token = options.token;
  }

  setToken(token: string): void {
    this.token = token;
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const headers: Record<string, string> = { 'Content-Type': 'application/json' };
    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const response = await fetch(`${this.baseURL}${path}`, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });

    if (!response.ok) {
      let error: APIError = { code: 'UNKNOWN_ERROR', message: response.statusText };
      try {
        error = await response.json();
      } catch {
        // Use default error
      }
      throw new RackdAPIError(error.code, error.message, error.details);
    }

    if (response.status === 204) {
      return undefined as T;
    }
    return response.json();
  }

  // Config
  async getConfig(): Promise<UIConfig> {
    return this.request<UIConfig>('GET', '/api/config');
  }

  // Devices
  async listDevices(filter?: DeviceFilter): Promise<Device[]> {
    const params = new URLSearchParams();
    if (filter?.tags?.length) params.set('tags', filter.tags.join(','));
    if (filter?.datacenter_id) params.set('datacenter_id', filter.datacenter_id);
    if (filter?.network_id) params.set('network_id', filter.network_id);
    const query = params.toString();
    return this.request<Device[]>('GET', `/api/devices${query ? `?${query}` : ''}`);
  }

  async getDevice(id: string): Promise<Device> {
    return this.request<Device>('GET', `/api/devices/${id}`);
  }

  async createDevice(device: Partial<Device>): Promise<Device> {
    return this.request<Device>('POST', '/api/devices', device);
  }

  async updateDevice(id: string, updates: Partial<Device>): Promise<Device> {
    return this.request<Device>('PUT', `/api/devices/${id}`, updates);
  }

  async deleteDevice(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/devices/${id}`);
  }

  async searchDevices(query: string): Promise<Device[]> {
    return this.request<Device[]>('GET', `/api/devices/search?q=${encodeURIComponent(query)}`);
  }

  // Relationships
  async addRelationship(deviceId: string, childId: string, type: DeviceRelationship['type']): Promise<void> {
    return this.request<void>('POST', `/api/devices/${deviceId}/relationships`, { child_id: childId, type });
  }

  async getRelationships(deviceId: string): Promise<DeviceRelationship[]> {
    return this.request<DeviceRelationship[]>('GET', `/api/devices/${deviceId}/relationships`);
  }

  async getRelatedDevices(deviceId: string, type: DeviceRelationship['type']): Promise<Device[]> {
    return this.request<Device[]>('GET', `/api/devices/${deviceId}/related?type=${type}`);
  }

  async removeRelationship(deviceId: string, childId: string, type: DeviceRelationship['type']): Promise<void> {
    return this.request<void>('DELETE', `/api/devices/${deviceId}/relationships/${childId}/${type}`);
  }

  // Datacenters
  async listDatacenters(): Promise<Datacenter[]> {
    return this.request<Datacenter[]>('GET', '/api/datacenters');
  }

  async getDatacenter(id: string): Promise<Datacenter> {
    return this.request<Datacenter>('GET', `/api/datacenters/${id}`);
  }

  async createDatacenter(datacenter: Partial<Datacenter>): Promise<Datacenter> {
    return this.request<Datacenter>('POST', '/api/datacenters', datacenter);
  }

  async updateDatacenter(id: string, updates: Partial<Datacenter>): Promise<Datacenter> {
    return this.request<Datacenter>('PUT', `/api/datacenters/${id}`, updates);
  }

  async deleteDatacenter(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/datacenters/${id}`);
  }

  async getDatacenterDevices(id: string): Promise<Device[]> {
    return this.request<Device[]>('GET', `/api/datacenters/${id}/devices`);
  }

  // Networks
  async listNetworks(datacenterId?: string): Promise<Network[]> {
    const query = datacenterId ? `?datacenter_id=${datacenterId}` : '';
    return this.request<Network[]>('GET', `/api/networks${query}`);
  }

  async getNetwork(id: string): Promise<Network> {
    return this.request<Network>('GET', `/api/networks/${id}`);
  }

  async createNetwork(network: Partial<Network>): Promise<Network> {
    return this.request<Network>('POST', '/api/networks', network);
  }

  async updateNetwork(id: string, updates: Partial<Network>): Promise<Network> {
    return this.request<Network>('PUT', `/api/networks/${id}`, updates);
  }

  async deleteNetwork(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/networks/${id}`);
  }

  async getNetworkUtilization(id: string): Promise<NetworkUtilization> {
    return this.request<NetworkUtilization>('GET', `/api/networks/${id}/utilization`);
  }

  // Network Pools
  async listNetworkPools(networkId: string): Promise<NetworkPool[]> {
    return this.request<NetworkPool[]>('GET', `/api/networks/${networkId}/pools`);
  }

  async createNetworkPool(networkId: string, pool: Partial<NetworkPool>): Promise<NetworkPool> {
    return this.request<NetworkPool>('POST', `/api/networks/${networkId}/pools`, pool);
  }

  async getNetworkPool(id: string): Promise<NetworkPool> {
    return this.request<NetworkPool>('GET', `/api/pools/${id}`);
  }

  async updateNetworkPool(id: string, updates: Partial<NetworkPool>): Promise<NetworkPool> {
    return this.request<NetworkPool>('PUT', `/api/pools/${id}`, updates);
  }

  async deleteNetworkPool(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/pools/${id}`);
  }

  async getNextIP(poolId: string): Promise<{ ip: string }> {
    return this.request<{ ip: string }>('GET', `/api/pools/${poolId}/next-ip`);
  }

  async getPoolHeatmap(poolId: string): Promise<IPStatus[]> {
    return this.request<IPStatus[]>('GET', `/api/pools/${poolId}/heatmap`);
  }

  // Discovery
  async startScan(networkId: string, scanType: DiscoveryScan['scan_type']): Promise<DiscoveryScan> {
    return this.request<DiscoveryScan>('POST', `/api/discovery/networks/${networkId}/scan`, { scan_type: scanType });
  }

  async listScans(networkId?: string): Promise<DiscoveryScan[]> {
    const query = networkId ? `?network_id=${networkId}` : '';
    return this.request<DiscoveryScan[]>('GET', `/api/discovery/scans${query}`);
  }

  async getScan(id: string): Promise<DiscoveryScan> {
    return this.request<DiscoveryScan>('GET', `/api/discovery/scans/${id}`);
  }

  async deleteScan(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/discovery/scans/${id}`);
  }

  async listDiscoveredDevices(networkId?: string): Promise<DiscoveredDevice[]> {
    const query = networkId ? `?network_id=${networkId}` : '';
    return this.request<DiscoveredDevice[]>('GET', `/api/discovery/devices${query}`);
  }

  async deleteDiscoveredDevice(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/discovery/devices/${id}`);
  }

  async deleteDiscoveredDevicesByNetwork(networkId: string): Promise<void> {
    return this.request<void>('DELETE', `/api/discovery/devices?network_id=${networkId}`);
  }

  async getDiscoveryRules(networkId: string): Promise<DiscoveryRule[]> {
    return this.request<DiscoveryRule[]>('GET', `/api/discovery/networks/${networkId}/rules`);
  }

  async saveDiscoveryRule(networkId: string, rule: Partial<DiscoveryRule>): Promise<DiscoveryRule> {
    return this.request<DiscoveryRule>('POST', `/api/discovery/networks/${networkId}/rules`, rule);
  }
}
