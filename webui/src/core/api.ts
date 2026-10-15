// API Client for Rackd - No DOM dependencies (mobile-ready)

import type {
  Address,
  APIError,
  AuditFilter,
  AuditLog,
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
  LogEntry,
  LogFilter,
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
  APIKey,
  CreateAPIKeyRequest,
  Credential,
  CredentialInput,
  ScheduledScan,
  ScheduledScanInput,
  OAuthClient,
} from './types';

export type {
  Address,
  APIError,
  AuditLog,
  Datacenter,
  Device,
  DeviceFilter,
  DeviceRelationship,
  DeviceStatusCounts,
  DiscoveredDevice,
  DiscoveryRule,
  DiscoveryScan,
  IPStatus,
  LogEntry,
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
  Credential,
  ScheduledScan,
  OAuthClient,
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

/**
 * Escapes a value for one path segment or query value. Every interpolated
 * value in an API path goes through it, so an id like "../users/<id>" or
 * "<id>?x" can't change which route a request reaches.
 */
function seg(value: string | number | boolean): string {
  return encodeURIComponent(String(value));
}

/** Whether a path's route part contains a "." or ".." segment, escaped or not. */
export function hasDotSegment(path: string): boolean {
  const route = path.split(/[?#]/)[0];
  return route.split('/').some((part) => {
    let decoded = part;
    try {
      decoded = decodeURIComponent(part);
    } catch {
      // A malformed escape isn't a dot segment.
    }
    return decoded === '.' || decoded === '..';
  });
}

// The API's largest page.
const DEVICE_PAGE_SIZE = 1000;

export class RackdAPI {
  private baseURL: string;
  private inFlightRequests: Map<string, Promise<unknown>> = new Map();

  constructor(options: RackdAPIOptions = {}) {
    this.baseURL = options.baseURL || '';
  }

  /** Calls any API route; extensions use it through window.rackd.request. */
  async call<T>(method: string, path: string, body?: unknown): Promise<T> {
    return this.request<T>(method.toUpperCase(), path, body);
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    // The browser resolves dot segments before sending, which would send the
    // request to a different route than the one the caller built.
    if (hasDotSegment(path)) {
      throw new RackdAPIError('INVALID_PATH', 'Refusing a request with a "." or ".." path segment');
    }

    // Only deduplicate GET requests
    const cacheKey = method === 'GET' ? `${method}:${path}` : null;

    if (cacheKey && this.inFlightRequests.has(cacheKey)) {
      return this.inFlightRequests.get(cacheKey) as Promise<T>;
    }

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'X-Requested-With': 'XMLHttpRequest'
    };

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
          const parsed = await response.json();
          // Backend error envelope is {"error": "...", "code": "..."}; some
          // endpoints use "message". Normalize so callers always get a message.
          error = {
            code: parsed?.code ?? 'UNKNOWN_ERROR',
            message: parsed?.error ?? parsed?.message ?? response.statusText,
            details: parsed?.details,
          };
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

  private async requestOptionalJSON<T>(method: string, path: string): Promise<T | null> {
    if (hasDotSegment(path)) {
      throw new RackdAPIError('INVALID_PATH', 'Refusing a request with a "." or ".." path segment');
    }
    const response = await fetch(`${this.baseURL}${path}`, {
      method,
      headers: {
        'Content-Type': 'application/json',
        'X-Requested-With': 'XMLHttpRequest'
      },
      credentials: 'same-origin',
    });

    const contentType = response.headers.get('content-type') || '';
    if (!contentType.includes('application/json')) {
      return null;
    }

    if (!response.ok) {
      let error: APIError = { code: 'UNKNOWN_ERROR', message: response.statusText };
      try {
        error = await response.json();
      } catch {
        // Use default error
      }

      if (response.status === 403) {
        const message = "You don't have permission to perform this action";
        window.dispatchEvent(new CustomEvent('toast:permission-denied', { detail: { message } }));
        throw new RackdAPIError('FORBIDDEN', message, error.details);
      }

      if (response.status === 401 && window.location.pathname !== '/login') {
        window.location.href = '/login';
      }

      throw new RackdAPIError(error.code, error.message, error.details);
    }

    const text = await response.text();
    return text ? JSON.parse(text) as T : null;
  }

  private buildQuery(params: Record<string, string | number | boolean | undefined>): string {
    const search = new URLSearchParams();
    for (const [key, value] of Object.entries(params)) {
      if (value !== undefined && value !== '') {
        search.set(key, String(value));
      }
    }
    const query = search.toString();
    return query ? `?${query}` : '';
  }

  // Config
  async getConfig(): Promise<UIConfig> {
    return this.request<UIConfig>('GET', '/api/v1/config');
  }

  async listAuditLogs(filter: AuditFilter = {}): Promise<AuditLog[]> {
    return this.request<AuditLog[]>('GET', `/api/v1/audit${this.buildQuery(filter as Record<string, string | number | boolean | undefined>)}`);
  }

  async getAuditLog(id: string): Promise<AuditLog> {
    return this.request<AuditLog>('GET', `/api/v1/audit/${seg(id)}`);
  }

  getAuditExportURL(filter: AuditFilter = {}, format: 'json' | 'csv' = 'json'): string {
    return `${this.baseURL}/api/v1/audit/export${this.buildQuery({ ...filter, format } as Record<string, string | number | boolean | undefined>)}`;
  }

  async listLogs(filter: LogFilter = {}): Promise<LogEntry[]> {
    return this.request<LogEntry[]>('GET', `/api/v1/logs${this.buildQuery(filter as Record<string, string | number | boolean | undefined>)}`);
  }

  async getLogEntry(id: string): Promise<LogEntry> {
    return this.request<LogEntry>('GET', `/api/v1/logs/${seg(id)}`);
  }

  getLogsExportURL(filter: LogFilter = {}, format: 'json' | 'csv' = 'json'): string {
    return `${this.baseURL}/api/v1/logs/export${this.buildQuery({ ...filter, format } as Record<string, string | number | boolean | undefined>)}`;
  }

  // Devices
  /**
   * Lists every device that matches the filter. The API returns devices a page
   * at a time (100 by default), so this asks for the largest pages and keeps
   * going until a short page, instead of silently stopping at the first.
   */
  async listDevices(filter?: DeviceFilter): Promise<Device[]> {
    const params = new URLSearchParams();
    for (const tag of filter?.tags ?? []) params.append('tags', tag);
    if (filter?.datacenter_id) params.set('datacenter_id', filter.datacenter_id);
    if (filter?.network_id) params.set('network_id', filter.network_id);
    if (filter?.pool_id) params.set('pool_id', filter.pool_id);
    if (filter?.status) params.set('status', filter.status);
    if (filter?.stale) params.set('stale', 'true');
    if (filter?.stale_days) params.set('stale_days', String(filter.stale_days));

    const devices: Device[] = [];
    for (let offset = 0; ; offset += DEVICE_PAGE_SIZE) {
      params.set('limit', String(DEVICE_PAGE_SIZE));
      params.set('offset', String(offset));
      const page = (await this.request<Device[]>('GET', `/api/v1/devices?${params}`)) ?? [];
      devices.push(...page);
      if (page.length < DEVICE_PAGE_SIZE) return devices;
    }
  }


  async getDevice(id: string): Promise<Device> {
    return this.request<Device>('GET', `/api/v1/devices/${seg(id)}`);
  }

  async createDevice(device: Partial<Device>): Promise<Device> {
    return this.request<Device>('POST', '/api/v1/devices', device);
  }

  async updateDevice(id: string, updates: Partial<Device>): Promise<Device> {
    return this.request<Device>('PUT', `/api/v1/devices/${seg(id)}`, updates);
  }

  async deleteDevice(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/devices/${seg(id)}`);
  }

  async getDeviceStatusCounts(): Promise<DeviceStatusCounts> {
    return this.request<DeviceStatusCounts>('GET', '/api/v1/devices/status-counts');
  }

  async searchDevices(query: string): Promise<Device[]> {
    return this.request<Device[]>('GET', `/api/v1/devices/search?q=${encodeURIComponent(query)}`);
  }

  async search(query: string): Promise<SearchResult[]> {
    const response = await this.request<{ results: SearchResult[] }>('GET', `/api/v1/search?q=${encodeURIComponent(query)}`);
    return response.results;
  }

  // Relationships
  async addRelationship(deviceId: string, childId: string, type: DeviceRelationship['type'], notes?: string): Promise<void> {
    return this.request<void>('POST', `/api/v1/devices/${seg(deviceId)}/relationships`, { child_id: childId, type, notes: notes || '' });
  }

  async getRelationships(deviceId: string): Promise<DeviceRelationship[]> {
    return this.request<DeviceRelationship[]>('GET', `/api/v1/devices/${seg(deviceId)}/relationships`);
  }

  async getAllRelationships(): Promise<DeviceRelationship[]> {
    return this.request<DeviceRelationship[]>('GET', '/api/v1/relationships');
  }

  async getRelatedDevices(deviceId: string, type: DeviceRelationship['type']): Promise<Device[]> {
    return this.request<Device[]>('GET', `/api/v1/devices/${seg(deviceId)}/related?type=${seg(type)}`);
  }

  async updateRelationshipNotes(deviceId: string, childId: string, type: DeviceRelationship['type'], notes: string): Promise<void> {
    return this.request<void>('PATCH', `/api/v1/devices/${seg(deviceId)}/relationships/${seg(childId)}/${seg(type)}`, { notes });
  }

  async removeRelationship(deviceId: string, childId: string, type: DeviceRelationship['type']): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/devices/${seg(deviceId)}/relationships/${seg(childId)}/${seg(type)}`);
  }

  // Datacenters
  async listDatacenters(): Promise<Datacenter[]> {
    return this.request<Datacenter[]>('GET', '/api/v1/datacenters');
  }

  async getDatacenter(id: string): Promise<Datacenter> {
    return this.request<Datacenter>('GET', `/api/v1/datacenters/${seg(id)}`);
  }

  async createDatacenter(datacenter: Partial<Datacenter>): Promise<Datacenter> {
    return this.request<Datacenter>('POST', '/api/v1/datacenters', datacenter);
  }

  async updateDatacenter(id: string, updates: Partial<Datacenter>): Promise<Datacenter> {
    return this.request<Datacenter>('PUT', `/api/v1/datacenters/${seg(id)}`, updates);
  }

  async deleteDatacenter(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/datacenters/${seg(id)}`);
  }

  async getDatacenterDevices(id: string): Promise<Device[]> {
    return this.request<Device[]>('GET', `/api/v1/datacenters/${seg(id)}/devices`);
  }

  // Networks
  async listNetworks(datacenterId?: string): Promise<Network[]> {
    const query = datacenterId ? `?datacenter_id=${encodeURIComponent(datacenterId)}` : '';
    return this.request<Network[]>('GET', `/api/v1/networks${query}`);
  }

  async getNetwork(id: string): Promise<Network> {
    return this.request<Network>('GET', `/api/v1/networks/${seg(id)}`);
  }

  async createNetwork(network: Partial<Network>): Promise<Network> {
    return this.request<Network>('POST', '/api/v1/networks', network);
  }

  async updateNetwork(id: string, updates: Partial<Network>): Promise<Network> {
    return this.request<Network>('PUT', `/api/v1/networks/${seg(id)}`, updates);
  }

  async deleteNetwork(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/networks/${seg(id)}`);
  }

  async getNetworkUtilization(id: string): Promise<NetworkUtilization> {
    return this.request<NetworkUtilization>('GET', `/api/v1/networks/${seg(id)}/utilization`);
  }

  async getNetworkDevices(id: string): Promise<Device[]> {
    return this.request<Device[]>('GET', `/api/v1/networks/${seg(id)}/devices`);
  }

  // Network Pools
  async listNetworkPools(networkId: string): Promise<NetworkPool[]> {
    return this.request<NetworkPool[]>('GET', `/api/v1/networks/${seg(networkId)}/pools`);
  }

  async createNetworkPool(networkId: string, pool: Partial<NetworkPool>): Promise<NetworkPool> {
    return this.request<NetworkPool>('POST', `/api/v1/networks/${seg(networkId)}/pools`, pool);
  }

  async getNetworkPool(id: string): Promise<NetworkPool> {
    return this.request<NetworkPool>('GET', `/api/v1/pools/${seg(id)}`);
  }

  async updateNetworkPool(id: string, updates: Partial<NetworkPool>): Promise<NetworkPool> {
    return this.request<NetworkPool>('PUT', `/api/v1/pools/${seg(id)}`, updates);
  }

  async deleteNetworkPool(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/pools/${seg(id)}`);
  }

  async getNextIP(poolId: string): Promise<{ ip: string }> {
    return this.request<{ ip: string }>('GET', `/api/v1/pools/${seg(poolId)}/next-ip`);
  }

  async getPoolHeatmap(poolId: string): Promise<IPStatus[]> {
    return this.request<IPStatus[]>('GET', `/api/v1/pools/${seg(poolId)}/heatmap`);
  }

  // Discovery
  async startScan(networkId: string, scanType: DiscoveryScan['scan_type']): Promise<DiscoveryScan> {
    return this.request<DiscoveryScan>('POST', `/api/v1/discovery/networks/${seg(networkId)}/scan`, { scan_type: scanType });
  }

  async listScans(networkId?: string): Promise<DiscoveryScan[]> {
    const query = networkId ? `?network_id=${encodeURIComponent(networkId)}` : '';
    return this.request<DiscoveryScan[]>('GET', `/api/v1/discovery/scans${query}`);
  }

  async getScan(id: string): Promise<DiscoveryScan> {
    return this.request<DiscoveryScan>('GET', `/api/v1/discovery/scans/${seg(id)}`);
  }

  async deleteScan(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/discovery/scans/${seg(id)}`);
  }

  async cancelScan(id: string): Promise<DiscoveryScan> {
    return this.request<DiscoveryScan>('POST', `/api/v1/discovery/scans/${seg(id)}/cancel`);
  }

  async listDiscoveredDevices(networkId?: string): Promise<DiscoveredDevice[]> {
    const query = networkId ? `?network_id=${encodeURIComponent(networkId)}` : '';
    return this.request<DiscoveredDevice[]>('GET', `/api/v1/discovery/devices${query}`);
  }

  async deleteDiscoveredDevice(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/discovery/devices/${seg(id)}`);
  }

  async promoteDevice(id: string, name: string, datacenterId?: string, makeModel?: string): Promise<Device> {
    const body: any = { name };
    if (datacenterId) body.datacenter_id = datacenterId;
    if (makeModel) body.make_model = makeModel;
    return this.request<Device>('POST', `/api/v1/discovery/devices/${seg(id)}/promote`, body);
  }

  async deleteDiscoveredDevicesByNetwork(networkId: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/discovery/devices?network_id=${encodeURIComponent(networkId ?? '')}`);
  }

  async getDiscoveryRules(): Promise<DiscoveryRule[]> {
    return this.request<DiscoveryRule[]>('GET', '/api/v1/discovery/rules');
  }

  async saveDiscoveryRule(rule: Partial<DiscoveryRule> & { network_id: string }): Promise<DiscoveryRule> {
    return this.request<DiscoveryRule>('POST', '/api/v1/discovery/rules', rule);
  }

  // Scan Profiles
  async listScanProfiles(): Promise<ScanProfile[]> {
    return this.request<ScanProfile[]>('GET', '/api/v1/scan-profiles');
  }

  async getScanProfile(id: string): Promise<ScanProfile> {
    return this.request<ScanProfile>('GET', `/api/v1/scan-profiles/${seg(id)}`);
  }

  async createScanProfile(profile: Partial<ScanProfile>): Promise<ScanProfile> {
    return this.request<ScanProfile>('POST', '/api/v1/scan-profiles', profile);
  }

  async updateScanProfile(id: string, profile: Partial<ScanProfile>): Promise<ScanProfile> {
    return this.request<ScanProfile>('PUT', `/api/v1/scan-profiles/${seg(id)}`, profile);
  }

  async deleteScanProfile(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/scan-profiles/${seg(id)}`);
  }

  // Credentials
  async listCredentials(datacenterId?: string): Promise<Credential[]> {
    const query = datacenterId ? `?datacenter_id=${encodeURIComponent(datacenterId)}` : '';
    return this.request<Credential[]>('GET', `/api/v1/credentials${query}`);
  }

  async createCredential(input: CredentialInput): Promise<Credential> {
    return this.request<Credential>('POST', '/api/v1/credentials', input);
  }

  async updateCredential(id: string, input: CredentialInput): Promise<Credential> {
    return this.request<Credential>('PUT', `/api/v1/credentials/${seg(id)}`, input);
  }

  async deleteCredential(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/credentials/${seg(id)}`);
  }

  // Scheduled scans
  async listScheduledScans(networkId?: string): Promise<ScheduledScan[]> {
    const query = networkId ? `?network_id=${encodeURIComponent(networkId)}` : '';
    return this.request<ScheduledScan[]>('GET', `/api/v1/scheduled-scans${query}`);
  }

  async createScheduledScan(input: ScheduledScanInput): Promise<ScheduledScan> {
    return this.request<ScheduledScan>('POST', '/api/v1/scheduled-scans', input);
  }

  async updateScheduledScan(id: string, input: ScheduledScanInput): Promise<ScheduledScan> {
    return this.request<ScheduledScan>('PUT', `/api/v1/scheduled-scans/${seg(id)}`, input);
  }

  async deleteScheduledScan(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/scheduled-scans/${seg(id)}`);
  }

  // OAuth clients
  async listOAuthClients(): Promise<OAuthClient[]> {
    const clients = await this.requestOptionalJSON<OAuthClient[]>('GET', '/api/v1/oauth/clients');
    return clients ?? [];
  }

  async deleteOAuthClient(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/oauth/clients/${seg(id)}`);
  }

  // Auth
  async login(username: string, password: string): Promise<LoginResponse> {
    return this.request<LoginResponse>('POST', '/api/v1/auth/login', { username, password });
  }

  async logout(): Promise<void> {
    return this.request<void>('POST', '/api/v1/auth/logout');
  }

  async getCurrentUser(): Promise<CurrentUser> {
    return this.request<CurrentUser>('GET', '/api/v1/auth/me');
  }

  // Users
  async listUsers(filter?: UserFilter): Promise<User[]> {
    const params = new URLSearchParams();
    if (filter?.username) params.set('username', filter.username);
    if (filter?.email) params.set('email', filter.email);
    if (filter?.is_active !== undefined) params.set('is_active', filter.is_active.toString());
    if (filter?.is_admin !== undefined) params.set('is_admin', filter.is_admin.toString());
    const query = params.toString();
    return this.request<User[]>('GET', `/api/v1/users${query ? `?${query}` : ''}`);
  }

  async getUser(id: string): Promise<User> {
    return this.request<User>('GET', `/api/v1/users/${seg(id)}`);
  }

  async createUser(user: CreateUserRequest): Promise<User> {
    return this.request<User>('POST', '/api/v1/users', user);
  }

  async updateUser(id: string, updates: UpdateUserRequest): Promise<User> {
    return this.request<User>('PUT', `/api/v1/users/${seg(id)}`, updates);
  }

  async deleteUser(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/users/${seg(id)}`);
  }

  async changePassword(id: string, request: ChangePasswordRequest): Promise<void> {
    return this.request<void>('POST', `/api/v1/users/${seg(id)}/password`, request);
  }

  async resetPassword(id: string, newPassword: string): Promise<void> {
    return this.request<void>('POST', `/api/v1/users/${seg(id)}/reset-password`, { new_password: newPassword });
  }

  async grantRole(userId: string, roleId: string): Promise<void> {
    return this.request<void>('POST', '/api/v1/users/grant-role', { user_id: userId, role_id: roleId });
  }

  async revokeRole(userId: string, roleId: string): Promise<void> {
    return this.request<void>('POST', '/api/v1/users/revoke-role', { user_id: userId, role_id: roleId });
  }

  // Roles
  async listRoles(filter?: RoleFilter): Promise<Role[]> {
    const params = new URLSearchParams();
    if (filter?.name) params.set('name', filter.name);
    if (filter?.is_system !== undefined) params.set('is_system', filter.is_system.toString());
    const query = params.toString();
    return this.request<Role[]>('GET', `/api/v1/roles${query ? `?${query}` : ''}`);
  }

  async getRole(id: string): Promise<Role> {
    return this.request<Role>('GET', `/api/v1/roles/${seg(id)}`);
  }

  async createRole(role: CreateRoleRequest): Promise<Role> {
    return this.request<Role>('POST', '/api/v1/roles', role);
  }

  async updateRole(id: string, role: UpdateRoleRequest): Promise<Role> {
    return this.request<Role>('PUT', `/api/v1/roles/${seg(id)}`, role);
  }

  async deleteRole(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/roles/${seg(id)}`);
  }

  async getRolePermissions(id: string): Promise<{ role: Role; permissions: Permission[] }> {
    return this.request<{ role: Role; permissions: Permission[] }>('GET', `/api/v1/roles/${seg(id)}/permissions`);
  }

  async listPermissions(): Promise<Permission[]> {
    return this.request<Permission[]>('GET', '/api/v1/permissions');
  }

  async getUserRoles(id: string): Promise<Role[]> {
    return this.request<Role[]>('GET', `/api/v1/users/${seg(id)}/roles`);
  }

  async getUserPermissions(id: string): Promise<Permission[]> {
    return this.request<Permission[]>('GET', `/api/v1/users/${seg(id)}/permissions`);
  }

  // Conflicts
  async listConflicts(): Promise<Conflict[]> {
    return this.request<Conflict[]>('GET', '/api/v1/conflicts');
  }

  async getConflictSummary(): Promise<{ duplicate_ips: number; overlapping_subnets: number } | null> {
    return this.request<{ duplicate_ips: number; overlapping_subnets: number } | null>('GET', '/api/v1/conflicts/summary');
  }

  async detectConflicts(type?: string): Promise<{ conflicts: Conflict[] }> {
    const query = type ? `?type=${seg(type)}` : '';
    return this.request<{ conflicts: Conflict[] }>('POST', `/api/v1/conflicts/detect${query}`);
  }

  async getConflict(id: string): Promise<Conflict> {
    return this.request<Conflict>('GET', `/api/v1/conflicts/${seg(id)}`);
  }

  async resolveConflict(id: string, resolution: ConflictResolution): Promise<void> {
    return this.request<void>('POST', `/api/v1/conflicts/${seg(id)}/resolve`, resolution);
  }

  async deleteConflict(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/conflicts/${seg(id)}`);
  }

  // Reservations
  async listReservations(filter?: ReservationFilter): Promise<Reservation[]> {
    const params = new URLSearchParams();
    if (filter?.pool_id) params.set('pool_id', filter.pool_id);
    if (filter?.status) params.set('status', filter.status);
    if (filter?.reserved_by) params.set('reserved_by', filter.reserved_by);
    if (filter?.ip) params.set('ip', filter.ip);
    const query = params.toString();
    return this.request<Reservation[]>('GET', `/api/v1/reservations${query ? `?${query}` : ''}`);
  }

  async getReservation(id: string): Promise<Reservation> {
    return this.request<Reservation>('GET', `/api/v1/reservations/${seg(id)}`);
  }

  async createReservation(request: CreateReservationRequest): Promise<Reservation> {
    return this.request<Reservation>('POST', '/api/v1/reservations', request);
  }

  async updateReservation(id: string, request: UpdateReservationRequest): Promise<Reservation> {
    return this.request<Reservation>('PUT', `/api/v1/reservations/${seg(id)}`, request);
  }

  async deleteReservation(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/reservations/${seg(id)}`);
  }

  async releaseReservation(id: string): Promise<void> {
    return this.request<void>('POST', `/api/v1/reservations/${seg(id)}/release`);
  }

  async getPoolReservations(poolId: string): Promise<Reservation[]> {
    return this.request<Reservation[]>('GET', `/api/v1/pools/${seg(poolId)}/reservations`);
  }

  // Dashboard
  async getDashboardStats(staleDays?: number, recentLimit?: number): Promise<DashboardStats> {
    const params = new URLSearchParams();
    if (staleDays) params.set('stale_days', staleDays.toString());
    if (recentLimit) params.set('recent_limit', recentLimit.toString());
    const query = params.toString();
    return this.request<DashboardStats>('GET', `/api/v1/dashboard${query ? `?${query}` : ''}`);
  }

  async getUtilizationTrend(type: 'network' | 'pool', resourceId: string, days?: number): Promise<UtilizationTrendPoint[]> {
    const params = new URLSearchParams();
    params.set('type', type);
    params.set('resource_id', resourceId);
    if (days) params.set('days', days.toString());
    return this.request<UtilizationTrendPoint[]>('GET', `/api/v1/dashboard/trend?${params.toString()}`);
  }

  // Webhooks
  async listWebhooks(active?: boolean): Promise<Webhook[]> {
    const params = active !== undefined ? `?active=${seg(active)}` : '';
    return this.request<Webhook[]>('GET', `/api/v1/webhooks${params}`);
  }

  async getWebhook(id: string): Promise<Webhook> {
    return this.request<Webhook>('GET', `/api/v1/webhooks/${seg(id)}`);
  }

  async createWebhook(request: CreateWebhookRequest): Promise<Webhook> {
    return this.request<Webhook>('POST', '/api/v1/webhooks', request);
  }

  async updateWebhook(id: string, request: UpdateWebhookRequest): Promise<Webhook> {
    return this.request<Webhook>('PUT', `/api/v1/webhooks/${seg(id)}`, request);
  }

  async deleteWebhook(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/webhooks/${seg(id)}`);
  }

  async pingWebhook(id: string): Promise<{ message: string; delivery: WebhookDelivery }> {
    return this.request<{ message: string; delivery: WebhookDelivery }>('POST', `/api/v1/webhooks/${seg(id)}/ping`);
  }

  async getWebhookDeliveries(webhookId: string, status?: DeliveryStatus): Promise<WebhookDelivery[]> {
    const params = status ? `?status=${seg(status)}` : '';
    return this.request<WebhookDelivery[]>('GET', `/api/v1/webhooks/${seg(webhookId)}/deliveries${params}`);
  }

  async getWebhookDelivery(webhookId: string, deliveryId: string): Promise<WebhookDelivery> {
    return this.request<WebhookDelivery>('GET', `/api/v1/webhooks/${seg(webhookId)}/deliveries/${seg(deliveryId)}`);
  }

  async getEventTypes(): Promise<EventTypeOption[]> {
    return this.request<EventTypeOption[]>('GET', '/api/v1/webhooks/events');
  }

  // Custom Fields
  async listCustomFieldDefinitions(type?: string): Promise<CustomFieldDefinition[]> {
    const params = type ? `?type=${seg(type)}` : '';
    return this.request<CustomFieldDefinition[]>('GET', `/api/v1/custom-fields${params}`);
  }

  async getCustomFieldDefinition(id: string): Promise<CustomFieldDefinition> {
    return this.request<CustomFieldDefinition>('GET', `/api/v1/custom-fields/${seg(id)}`);
  }

  async createCustomFieldDefinition(request: CreateCustomFieldDefinitionRequest): Promise<CustomFieldDefinition> {
    return this.request<CustomFieldDefinition>('POST', '/api/v1/custom-fields', request);
  }

  async updateCustomFieldDefinition(id: string, request: UpdateCustomFieldDefinitionRequest): Promise<CustomFieldDefinition> {
    return this.request<CustomFieldDefinition>('PUT', `/api/v1/custom-fields/${seg(id)}`, request);
  }

  async deleteCustomFieldDefinition(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/custom-fields/${seg(id)}`);
  }

  async getCustomFieldTypes(): Promise<{ value: string; label: string }[]> {
    return this.request<{ value: string; label: string }[]>('GET', '/api/v1/custom-fields/types');
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
    return this.request<Circuit[]>('GET', `/api/v1/circuits${query}`);
  }

  async getCircuit(id: string): Promise<Circuit> {
    return this.request<Circuit>('GET', `/api/v1/circuits/${seg(id)}`);
  }

  async createCircuit(data: CreateCircuitRequest): Promise<Circuit> {
    return this.request<Circuit>('POST', '/api/v1/circuits', data);
  }

  async updateCircuit(id: string, data: UpdateCircuitRequest): Promise<Circuit> {
    return this.request<Circuit>('PUT', `/api/v1/circuits/${seg(id)}`, data);
  }

  async deleteCircuit(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/circuits/${seg(id)}`);
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
    return this.request<NATMapping[]>('GET', `/api/v1/nat${query}`);
  }

  async getNATMapping(id: string): Promise<NATMapping> {
    return this.request<NATMapping>('GET', `/api/v1/nat/${seg(id)}`);
  }

  async createNATMapping(data: CreateNATRequest): Promise<NATMapping> {
    return this.request<NATMapping>('POST', '/api/v1/nat', data);
  }

  async updateNATMapping(id: string, data: UpdateNATRequest): Promise<NATMapping> {
    return this.request<NATMapping>('PUT', `/api/v1/nat/${seg(id)}`, data);
  }

  async deleteNATMapping(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/nat/${seg(id)}`);
  }

  // DNS Providers
  async listDNSProviders(filter?: DNSProviderFilter): Promise<DNSProvider[]> {
    const params = new URLSearchParams();
    if (filter?.type) params.set('type', filter.type);
    const query = params.toString();
    return this.request<DNSProvider[]>('GET', `/api/v1/dns/providers${query ? `?${query}` : ''}`);
  }

  async getDNSProvider(id: string): Promise<DNSProvider> {
    return this.request<DNSProvider>('GET', `/api/v1/dns/providers/${seg(id)}`);
  }

  async createDNSProvider(req: CreateDNSProviderRequest): Promise<DNSProvider> {
    return this.request<DNSProvider>('POST', '/api/v1/dns/providers', req);
  }

  async updateDNSProvider(id: string, req: UpdateDNSProviderRequest): Promise<DNSProvider> {
    return this.request<DNSProvider>('PUT', `/api/v1/dns/providers/${seg(id)}`, req);
  }

  async deleteDNSProvider(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/dns/providers/${seg(id)}`);
  }

  async testDNSProvider(id: string): Promise<void> {
    return this.request<void>('POST', `/api/v1/dns/providers/${seg(id)}/test`);
  }

  // DNS Zones
  async listDNSZones(filter?: DNSZoneFilter): Promise<DNSZone[]> {
    const params = new URLSearchParams();
    if (filter?.provider_id) params.set('provider_id', filter.provider_id);
    if (filter?.network_id) params.set('network_id', filter.network_id);
    if (filter?.auto_sync !== undefined) params.set('auto_sync', filter.auto_sync.toString());
    const query = params.toString();
    return this.request<DNSZone[]>('GET', `/api/v1/dns/zones${query ? `?${query}` : ''}`);
  }

  async getDNSZone(id: string): Promise<DNSZone> {
    return this.request<DNSZone>('GET', `/api/v1/dns/zones/${seg(id)}`);
  }

  async createDNSZone(req: CreateDNSZoneRequest): Promise<DNSZone> {
    return this.request<DNSZone>('POST', '/api/v1/dns/zones', req);
  }

  async updateDNSZone(id: string, req: UpdateDNSZoneRequest): Promise<DNSZone> {
    return this.request<DNSZone>('PUT', `/api/v1/dns/zones/${seg(id)}`, req);
  }

  async deleteDNSZone(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/dns/zones/${seg(id)}`);
  }

  async syncDNSZone(id: string): Promise<SyncResult> {
    return this.request<SyncResult>('POST', `/api/v1/dns/zones/${seg(id)}/sync`);
  }

  async importDNSZone(id: string): Promise<ImportResult> {
    return this.request<ImportResult>('POST', `/api/v1/dns/zones/${seg(id)}/import`);
  }

  // DNS Records
  async listDNSRecords(filter?: DNSRecordFilter): Promise<DNSRecord[]> {
    // There is no zone-less list route on the server; a call without zone_id
    // would silently 404.
    if (!filter?.zone_id) {
      throw new Error('listDNSRecords requires a zone_id filter');
    }
    const params = new URLSearchParams();
    if (filter?.type) params.set('type', filter.type);
    if (filter?.sync_status) params.set('sync_status', filter.sync_status);
    if (filter?.link_status) params.set('link_status', filter.link_status);
    const query = params.toString();
    return this.request<DNSRecord[]>('GET', `/api/v1/dns/zones/${encodeURIComponent(filter.zone_id)}/records${query ? `?${query}` : ''}`);
  }

  async getDNSRecord(id: string): Promise<DNSRecord> {
    return this.request<DNSRecord>('GET', `/api/v1/dns/records/${seg(id)}`);
  }

  async updateDNSRecord(id: string, req: UpdateDNSRecordRequest): Promise<DNSRecord> {
    return this.request<DNSRecord>('PUT', `/api/v1/dns/records/${seg(id)}`, req);
  }

  async deleteDNSRecord(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/dns/records/${seg(id)}`);
  }

  async linkDNSRecord(id: string, req: { device_id: string; address_id?: string; add_to_domains?: boolean }): Promise<DNSRecord> {
    return this.request<DNSRecord>('POST', `/api/v1/dns/records/${seg(id)}/link`, req);
  }
  async promoteDNSRecord(id: string, req: { name?: string; datacenter_id?: string; tags?: string[] }): Promise<DNSRecord> {
    return this.request<DNSRecord>('POST', `/api/v1/dns/records/${seg(id)}/promote`, req);
  }

  // API Keys
  async listAPIKeys(): Promise<APIKey[]> {
    return this.request<APIKey[]>('GET', '/api/v1/keys');
  }
  async createAPIKey(req: CreateAPIKeyRequest): Promise<APIKey> {
    return this.request<APIKey>('POST', '/api/v1/keys', req);
  }
  async deleteAPIKey(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/v1/keys/${seg(id)}`);
  }

}

// Singleton instance for request deduplication across components
export const api = new RackdAPI();
