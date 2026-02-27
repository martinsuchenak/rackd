// API Client for Rackd - No DOM dependencies (mobile-ready)

import type {
  Address,
  APIError,
  ChangePasswordRequest,
  CreateUserRequest,
  CreateReservationRequest,
  CurrentUser,
  Datacenter,
  Device,
  DeviceFilter,
  DeviceRelationship,
  DiscoveredDevice,
  DiscoveryRule,
  DiscoveryScan,
  IPStatus,
  LoginRequest,
  LoginResponse,
  NavItem,
  Network,
  NetworkPool,
  NetworkUtilization,
  Permission,
  Reservation,
  ReservationFilter,
  Role,
  RoleFilter,
  ScanProfile,
  SearchResult,
  ServiceInfo,
  UIConfig,
  UpdateReservationRequest,
  UpdateUserRequest,
  CreateRoleRequest,
  UpdateRoleRequest,
  User,
  UserFilter,
  UserInfo,
  Conflict,
  ConflictResolution,
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
  Permission,
  Role,
  ScanProfile,
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
}

export class RackdAPI {
  private baseURL: string;
  private inFlightRequests: Map<string, Promise<unknown>> = new Map();

  constructor(options: RackdAPIOptions = {}) {
    this.baseURL = options.baseURL || '';
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    // Only deduplicate GET requests
    const cacheKey = method === 'GET' ? `${method}:${path}` : null;

    if (cacheKey && this.inFlightRequests.has(cacheKey)) {
      return this.inFlightRequests.get(cacheKey) as Promise<T>;
    }

    const headers: Record<string, string> = { 'Content-Type': 'application/json' };

    const requestPromise = (async () => {
      const response = await fetch(`${this.baseURL}${path}`, {
        method,
        headers,
        credentials: 'same-origin',
        body: body !== undefined ? JSON.stringify(body) : undefined,
      });

      if (!response.ok) {
        let error: APIError = { code: 'UNKNOWN_ERROR', message: response.statusText };
        try {
          error = await response.json();
        } catch {
          // Use default error
        }

        // Handle 403 Forbidden with user-friendly toast message
        if (response.status === 403) {
          const message = "You don't have permission to perform this action";
          // Dispatch event for toast notification
          window.dispatchEvent(new CustomEvent('toast:permission-denied', { detail: { message } }));
          throw new RackdAPIError('FORBIDDEN', message, error.details);
        }

        // Handle 401 Unauthorized
        if (response.status === 401) {
          // Redirect to login if not already there
          if (window.location.pathname !== '/login') {
            window.location.href = '/login';
          }
        }

        throw new RackdAPIError(error.code, error.message, error.details);
      }

      if (response.status === 204 || response.status === 201 && response.headers.get('content-length') === '0') {
        return undefined as T;
      }
      const text = await response.text();
      if (!text) {
        return undefined as T;
      }
      return JSON.parse(text);
    })();

    if (cacheKey) {
      this.inFlightRequests.set(cacheKey, requestPromise);
      requestPromise.finally(() => {
        this.inFlightRequests.delete(cacheKey);
      });
    }

    return requestPromise;
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

  async search(query: string): Promise<SearchResult[]> {
    const response = await this.request<{ results: SearchResult[] }>('GET', `/api/search?q=${encodeURIComponent(query)}`);
    return response.results;
  }

  // Relationships
  async addRelationship(deviceId: string, childId: string, type: DeviceRelationship['type'], notes?: string): Promise<void> {
    return this.request<void>('POST', `/api/devices/${deviceId}/relationships`, { child_id: childId, type, notes: notes || '' });
  }

  async getRelationships(deviceId: string): Promise<DeviceRelationship[]> {
    return this.request<DeviceRelationship[]>('GET', `/api/devices/${deviceId}/relationships`);
  }

  async getAllRelationships(): Promise<DeviceRelationship[]> {
    return this.request<DeviceRelationship[]>('GET', '/api/relationships');
  }

  async getRelatedDevices(deviceId: string, type: DeviceRelationship['type']): Promise<Device[]> {
    return this.request<Device[]>('GET', `/api/devices/${deviceId}/related?type=${type}`);
  }

  async updateRelationshipNotes(deviceId: string, childId: string, type: DeviceRelationship['type'], notes: string): Promise<void> {
    return this.request<void>('PATCH', `/api/devices/${deviceId}/relationships/${childId}/${type}`, { notes });
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

  async getNetworkDevices(id: string): Promise<Device[]> {
    return this.request<Device[]>('GET', `/api/networks/${id}/devices`);
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

  async cancelScan(id: string): Promise<DiscoveryScan> {
    return this.request<DiscoveryScan>('POST', `/api/discovery/scans/${id}/cancel`);
  }

  async listDiscoveredDevices(networkId?: string): Promise<DiscoveredDevice[]> {
    const query = networkId ? `?network_id=${networkId}` : '';
    return this.request<DiscoveredDevice[]>('GET', `/api/discovery/devices${query}`);
  }

  async deleteDiscoveredDevice(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/discovery/devices/${id}`);
  }

  async promoteDevice(id: string, name: string, datacenterId?: string, makeModel?: string): Promise<Device> {
    const body: any = { name };
    if (datacenterId) body.datacenter_id = datacenterId;
    if (makeModel) body.make_model = makeModel;
    return this.request<Device>('POST', `/api/discovery/devices/${id}/promote`, body);
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

  // Scan Profiles
  async listScanProfiles(): Promise<ScanProfile[]> {
    return this.request<ScanProfile[]>('GET', '/api/scan-profiles');
  }

  async getScanProfile(id: string): Promise<ScanProfile> {
    return this.request<ScanProfile>('GET', `/api/scan-profiles/${id}`);
  }

  async createScanProfile(profile: Partial<ScanProfile>): Promise<ScanProfile> {
    return this.request<ScanProfile>('POST', '/api/scan-profiles', profile);
  }

  async updateScanProfile(id: string, profile: Partial<ScanProfile>): Promise<ScanProfile> {
    return this.request<ScanProfile>('PUT', `/api/scan-profiles/${id}`, profile);
  }

  async deleteScanProfile(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/scan-profiles/${id}`);
  }

  // Auth
  async login(username: string, password: string): Promise<LoginResponse> {
    return this.request<LoginResponse>('POST', '/api/auth/login', { username, password });
  }

  async logout(): Promise<void> {
    return this.request<void>('POST', '/api/auth/logout');
  }

  async getCurrentUser(): Promise<CurrentUser> {
    return this.request<CurrentUser>('GET', '/api/auth/me');
  }

  // Users
  async listUsers(filter?: UserFilter): Promise<User[]> {
    const params = new URLSearchParams();
    if (filter?.username) params.set('username', filter.username);
    if (filter?.email) params.set('email', filter.email);
    if (filter?.is_active !== undefined) params.set('is_active', filter.is_active.toString());
    if (filter?.is_admin !== undefined) params.set('is_admin', filter.is_admin.toString());
    const query = params.toString();
    return this.request<User[]>('GET', `/api/users${query ? `?${query}` : ''}`);
  }

  async getUser(id: string): Promise<User> {
    return this.request<User>('GET', `/api/users/${id}`);
  }

  async createUser(user: CreateUserRequest): Promise<User> {
    return this.request<User>('POST', '/api/users', user);
  }

  async updateUser(id: string, updates: UpdateUserRequest): Promise<User> {
    return this.request<User>('PUT', `/api/users/${id}`, updates);
  }

  async deleteUser(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/users/${id}`);
  }

  async changePassword(id: string, request: ChangePasswordRequest): Promise<void> {
    return this.request<void>('POST', `/api/users/${id}/password`, request);
  }

  async grantRole(userId: string, roleId: string): Promise<void> {
    return this.request<void>('POST', '/api/users/grant-role', { user_id: userId, role_id: roleId });
  }

  async revokeRole(userId: string, roleId: string): Promise<void> {
    return this.request<void>('POST', '/api/users/revoke-role', { user_id: userId, role_id: roleId });
  }

  // Roles
  async listRoles(filter?: RoleFilter): Promise<Role[]> {
    const params = new URLSearchParams();
    if (filter?.name) params.set('name', filter.name);
    if (filter?.is_system !== undefined) params.set('is_system', filter.is_system.toString());
    const query = params.toString();
    return this.request<Role[]>('GET', `/api/roles${query ? `?${query}` : ''}`);
  }

  async getRole(id: string): Promise<Role> {
    return this.request<Role>('GET', `/api/roles/${id}`);
  }

  async createRole(role: CreateRoleRequest): Promise<Role> {
    return this.request<Role>('POST', '/api/roles', role);
  }

  async updateRole(id: string, role: UpdateRoleRequest): Promise<Role> {
    return this.request<Role>('PUT', `/api/roles/${id}`, role);
  }

  async deleteRole(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/roles/${id}`);
  }

  async getRolePermissions(id: string): Promise<{ role: Role; permissions: Permission[] }> {
    return this.request<{ role: Role; permissions: Permission[] }>('GET', `/api/roles/${id}/permissions`);
  }

  async listPermissions(): Promise<Permission[]> {
    return this.request<Permission[]>('GET', '/api/permissions');
  }

  async getUserRoles(id: string): Promise<Role[]> {
    return this.request<Role[]>('GET', `/api/users/${id}/roles`);
  }

  async getUserPermissions(id: string): Promise<Permission[]> {
    return this.request<Permission[]>('GET', `/api/users/${id}/permissions`);
  }

  // Conflicts
  async listConflicts(): Promise<Conflict[]> {
    return this.request<Conflict[]>('GET', '/api/conflicts');
  }

  async getConflictSummary(): Promise<{ duplicate_ips: number; overlapping_subnets: number } | null> {
    return this.request<{ duplicate_ips: number; overlapping_subnets: number } | null>('GET', '/api/conflicts/summary');
  }

  async detectConflicts(type?: string): Promise<{ conflicts: Conflict[] }> {
    const query = type ? `?type=${type}` : '';
    return this.request<{ conflicts: Conflict[] }>('POST', `/api/conflicts/detect${query}`);
  }

  async getConflict(id: string): Promise<Conflict> {
    return this.request<Conflict>('GET', `/api/conflicts/${id}`);
  }

  async resolveConflict(resolution: ConflictResolution): Promise<void> {
    return this.request<void>('POST', '/api/conflicts/resolve', resolution);
  }

  async deleteConflict(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/conflicts/${id}`);
  }

  // Reservations
  async listReservations(filter?: ReservationFilter): Promise<Reservation[]> {
    const params = new URLSearchParams();
    if (filter?.pool_id) params.set('pool_id', filter.pool_id);
    if (filter?.status) params.set('status', filter.status);
    if (filter?.reserved_by) params.set('reserved_by', filter.reserved_by);
    if (filter?.ip) params.set('ip', filter.ip);
    const query = params.toString();
    return this.request<Reservation[]>('GET', `/api/reservations${query ? `?${query}` : ''}`);
  }

  async getReservation(id: string): Promise<Reservation> {
    return this.request<Reservation>('GET', `/api/reservations/${id}`);
  }

  async createReservation(request: CreateReservationRequest): Promise<Reservation> {
    return this.request<Reservation>('POST', '/api/reservations', request);
  }

  async updateReservation(id: string, request: UpdateReservationRequest): Promise<Reservation> {
    return this.request<Reservation>('PUT', `/api/reservations/${id}`, request);
  }

  async deleteReservation(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/reservations/${id}`);
  }

  async releaseReservation(id: string): Promise<void> {
    return this.request<void>('POST', `/api/reservations/${id}/release`);
  }

  async getPoolReservations(poolId: string): Promise<Reservation[]> {
    return this.request<Reservation[]>('GET', `/api/pools/${poolId}/reservations`);
  }
}

// Singleton instance for request deduplication across components
export const api = new RackdAPI();
