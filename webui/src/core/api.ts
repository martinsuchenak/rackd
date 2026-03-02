// API Client for Rackd - No DOM dependencies (mobile-ready)

import type {
  Address,
  APIError,
  ChangePasswordRequest,
  CreateUserRequest,
  CreateReservationRequest,
  CreateWebhookRequest,
  CreateCustomFieldDefinitionRequest,
  CurrentUser,
  CustomFieldDefinition,
  CustomFieldValue,
  CustomFieldWithDefinition,
  DashboardStats,
  Datacenter,
  DeliveryStatus,
  Device,
  DeviceFilter,
  DeviceRelationship,
  DeviceStatusCounts,
  DiscoveredDevice,
  DiscoveryRule,
  DiscoveryScan,
  EventTypeOption,
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
  UpdateWebhookRequest,
  UpdateCustomFieldDefinitionRequest,
  CreateRoleRequest,
  UpdateRoleRequest,
  User,
  UserFilter,
  UserInfo,
  Webhook,
  WebhookDelivery,
  Conflict,
  ConflictResolution,
  UtilizationTrendPoint,
  Circuit,
  CircuitFilter,
  CreateCircuitRequest,
  UpdateCircuitRequest,
  NATMapping,
  NATFilter,
  CreateNATRequest,
  UpdateNATRequest,
  DNSProvider,
  DNSProviderFilter,
  CreateDNSProviderRequest,
  UpdateDNSProviderRequest,
  DNSZone,
  DNSZoneFilter,
  CreateDNSZoneRequest,
  UpdateDNSZoneRequest,
  DNSRecord,
  DNSRecordFilter,
  UpdateDNSRecordRequest,
  SyncResult,
  ImportResult,
} from './types';

export type {
  Address,
  APIError,
  Datacenter,
  Device,
  DeviceFilter,
  DeviceRelationship,
  DeviceStatusCounts,
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
    if (filter?.pool_id) params.set('pool_id', filter.pool_id);
    if (filter?.status) params.set('status', filter.status);
    if (filter?.stale) params.set('stale', 'true');
    if (filter?.stale_days) params.set('stale_days', String(filter.stale_days));
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

  async getDeviceStatusCounts(): Promise<DeviceStatusCounts> {
    return this.request<DeviceStatusCounts>('GET', '/api/devices/status-counts');
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

  async resetPassword(id: string, newPassword: string): Promise<void> {
    return this.request<void>('POST', `/api/users/${id}/reset-password`, { new_password: newPassword });
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

  // Dashboard
  async getDashboardStats(staleDays?: number, recentLimit?: number): Promise<DashboardStats> {
    const params = new URLSearchParams();
    if (staleDays) params.set('stale_days', staleDays.toString());
    if (recentLimit) params.set('recent_limit', recentLimit.toString());
    const query = params.toString();
    return this.request<DashboardStats>('GET', `/api/dashboard${query ? `?${query}` : ''}`);
  }

  async getUtilizationTrend(type: 'network' | 'pool', resourceId: string, days?: number): Promise<UtilizationTrendPoint[]> {
    const params = new URLSearchParams();
    params.set('type', type);
    params.set('resource_id', resourceId);
    if (days) params.set('days', days.toString());
    return this.request<UtilizationTrendPoint[]>('GET', `/api/dashboard/trend?${params.toString()}`);
  }

  // Webhooks
  async listWebhooks(active?: boolean): Promise<Webhook[]> {
    const params = active !== undefined ? `?active=${active}` : '';
    return this.request<Webhook[]>('GET', `/api/webhooks${params}`);
  }

  async getWebhook(id: string): Promise<Webhook> {
    return this.request<Webhook>('GET', `/api/webhooks/${id}`);
  }

  async createWebhook(request: CreateWebhookRequest): Promise<Webhook> {
    return this.request<Webhook>('POST', '/api/webhooks', request);
  }

  async updateWebhook(id: string, request: UpdateWebhookRequest): Promise<Webhook> {
    return this.request<Webhook>('PUT', `/api/webhooks/${id}`, request);
  }

  async deleteWebhook(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/webhooks/${id}`);
  }

  async pingWebhook(id: string): Promise<{ message: string; delivery: WebhookDelivery }> {
    return this.request<{ message: string; delivery: WebhookDelivery }>('POST', `/api/webhooks/${id}/ping`);
  }

  async getWebhookDeliveries(webhookId: string, status?: DeliveryStatus): Promise<WebhookDelivery[]> {
    const params = status ? `?status=${status}` : '';
    return this.request<WebhookDelivery[]>('GET', `/api/webhooks/${webhookId}/deliveries${params}`);
  }

  async getWebhookDelivery(webhookId: string, deliveryId: string): Promise<WebhookDelivery> {
    return this.request<WebhookDelivery>('GET', `/api/webhooks/${webhookId}/deliveries/${deliveryId}`);
  }

  async getEventTypes(): Promise<EventTypeOption[]> {
    return this.request<EventTypeOption[]>('GET', '/api/webhooks/events');
  }

  // Custom Fields
  async listCustomFieldDefinitions(type?: string): Promise<CustomFieldDefinition[]> {
    const params = type ? `?type=${type}` : '';
    return this.request<CustomFieldDefinition[]>('GET', `/api/custom-fields${params}`);
  }

  async getCustomFieldDefinition(id: string): Promise<CustomFieldDefinition> {
    return this.request<CustomFieldDefinition>('GET', `/api/custom-fields/${id}`);
  }

  async createCustomFieldDefinition(request: CreateCustomFieldDefinitionRequest): Promise<CustomFieldDefinition> {
    return this.request<CustomFieldDefinition>('POST', '/api/custom-fields', request);
  }

  async updateCustomFieldDefinition(id: string, request: UpdateCustomFieldDefinitionRequest): Promise<CustomFieldDefinition> {
    return this.request<CustomFieldDefinition>('PUT', `/api/custom-fields/${id}`, request);
  }

  async deleteCustomFieldDefinition(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/custom-fields/${id}`);
  }

  async getCustomFieldTypes(): Promise<{ value: string; label: string }[]> {
    return this.request<{ value: string; label: string }[]>('GET', '/api/custom-fields/types');
  }

  // Circuit API methods
  async listCircuits(filter?: CircuitFilter): Promise<Circuit[]> {
    const params = new URLSearchParams();
    if (filter?.provider) params.append('provider', filter.provider);
    if (filter?.status) params.append('status', filter.status);
    if (filter?.datacenter_id) params.append('datacenter_id', filter.datacenter_id);
    if (filter?.type) params.append('type', filter.type);
    if (filter?.tags) {
      filter.tags.forEach(tag => params.append('tags', tag));
    }
    const query = params.toString() ? `?${params.toString()}` : '';
    return this.request<Circuit[]>('GET', `/api/circuits${query}`);
  }

  async getCircuit(id: string): Promise<Circuit> {
    return this.request<Circuit>('GET', `/api/circuits/${id}`);
  }

  async createCircuit(data: CreateCircuitRequest): Promise<Circuit> {
    return this.request<Circuit>('POST', '/api/circuits', data);
  }

  async updateCircuit(id: string, data: UpdateCircuitRequest): Promise<Circuit> {
    return this.request<Circuit>('PUT', `/api/circuits/${id}`, data);
  }

  async deleteCircuit(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/circuits/${id}`);
  }

  // NAT API methods
  async listNATMappings(filter?: NATFilter): Promise<NATMapping[]> {
    const params = new URLSearchParams();
    if (filter?.external_ip) params.append('external_ip', filter.external_ip);
    if (filter?.internal_ip) params.append('internal_ip', filter.internal_ip);
    if (filter?.protocol) params.append('protocol', filter.protocol);
    if (filter?.device_id) params.append('device_id', filter.device_id);
    if (filter?.datacenter_id) params.append('datacenter_id', filter.datacenter_id);
    if (filter?.network_id) params.append('network_id', filter.network_id);
    if (filter?.enabled !== undefined) params.append('enabled', String(filter.enabled));
    const query = params.toString() ? `?${params.toString()}` : '';
    return this.request<NATMapping[]>('GET', `/api/nat${query}`);
  }

  async getNATMapping(id: string): Promise<NATMapping> {
    return this.request<NATMapping>('GET', `/api/nat/${id}`);
  }

  async createNATMapping(data: CreateNATRequest): Promise<NATMapping> {
    return this.request<NATMapping>('POST', '/api/nat', data);
  }

  async updateNATMapping(id: string, data: UpdateNATRequest): Promise<NATMapping> {
    return this.request<NATMapping>('PUT', `/api/nat/${id}`, data);
  }

  async deleteNATMapping(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/nat/${id}`);
  }

  // DNS Providers
  async listDNSProviders(filter?: DNSProviderFilter): Promise<DNSProvider[]> {
    const params = new URLSearchParams();
    if (filter?.type) params.set('type', filter.type);
    const query = params.toString();
    return this.request<DNSProvider[]>('GET', `/api/dns/providers${query ? `?${query}` : ''}`);
  }

  async getDNSProvider(id: string): Promise<DNSProvider> {
    return this.request<DNSProvider>('GET', `/api/dns/providers/${id}`);
  }

  async createDNSProvider(req: CreateDNSProviderRequest): Promise<DNSProvider> {
    return this.request<DNSProvider>('POST', '/api/dns/providers', req);
  }

  async updateDNSProvider(id: string, req: UpdateDNSProviderRequest): Promise<DNSProvider> {
    return this.request<DNSProvider>('PUT', `/api/dns/providers/${id}`, req);
  }

  async deleteDNSProvider(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/dns/providers/${id}`);
  }

  async testDNSProvider(id: string): Promise<void> {
    return this.request<void>('POST', `/api/dns/providers/${id}/test`);
  }

  // DNS Zones
  async listDNSZones(filter?: DNSZoneFilter): Promise<DNSZone[]> {
    const params = new URLSearchParams();
    if (filter?.provider_id) params.set('provider_id', filter.provider_id);
    if (filter?.network_id) params.set('network_id', filter.network_id);
    if (filter?.auto_sync !== undefined) params.set('auto_sync', filter.auto_sync.toString());
    const query = params.toString();
    return this.request<DNSZone[]>('GET', `/api/dns/zones${query ? `?${query}` : ''}`);
  }

  async getDNSZone(id: string): Promise<DNSZone> {
    return this.request<DNSZone>('GET', `/api/dns/zones/${id}`);
  }

  async createDNSZone(req: CreateDNSZoneRequest): Promise<DNSZone> {
    return this.request<DNSZone>('POST', '/api/dns/zones', req);
  }

  async updateDNSZone(id: string, req: UpdateDNSZoneRequest): Promise<DNSZone> {
    return this.request<DNSZone>('PUT', `/api/dns/zones/${id}`, req);
  }

  async deleteDNSZone(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/dns/zones/${id}`);
  }

  async syncDNSZone(id: string): Promise<SyncResult> {
    return this.request<SyncResult>('POST', `/api/dns/zones/${id}/sync`);
  }

  async importDNSZone(id: string): Promise<ImportResult> {
    return this.request<ImportResult>('POST', `/api/dns/zones/${id}/import`);
  }

  // DNS Records
  async listDNSRecords(filter?: DNSRecordFilter): Promise<DNSRecord[]> {
    if (filter?.zone_id) {
      // Use zone-specific endpoint when zone_id is provided
      const params = new URLSearchParams();
      if (filter?.type) params.set('type', filter.type);
      if (filter?.sync_status) params.set('sync_status', filter.sync_status);
      const query = params.toString();
      return this.request<DNSRecord[]>('GET', `/api/dns/zones/${filter.zone_id}/records${query ? `?${query}` : ''}`);
    }
    const params = new URLSearchParams();
    if (filter?.device_id) params.set('device_id', filter.device_id);
    if (filter?.type) params.set('type', filter.type);
    if (filter?.sync_status) params.set('sync_status', filter.sync_status);
    const query = params.toString();
    return this.request<DNSRecord[]>('GET', `/api/dns/records${query ? `?${query}` : ''}`);
  }

  async getDNSRecord(id: string): Promise<DNSRecord> {
    return this.request<DNSRecord>('GET', `/api/dns/records/${id}`);
  }

  async updateDNSRecord(id: string, req: UpdateDNSRecordRequest): Promise<DNSRecord> {
    return this.request<DNSRecord>('PUT', `/api/dns/records/${id}`, req);
  }

  async deleteDNSRecord(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/dns/records/${id}`);
  }
}

// Singleton instance for request deduplication across components
export const api = new RackdAPI();
