// Shared types for Rackd Web UI

export interface Address {
  id?: string;
  ip: string;
  port?: number;
  type: string;
  label: string;
  network_id?: string;
  switch_port?: string;
  pool_id?: string;
}

export type DeviceStatus = 'planned' | 'active' | 'maintenance' | 'decommissioned';

export interface Device {
  id: string;
  name: string;
  hostname?: string;
  description: string;
  make_model: string;
  os: string;
  datacenter_id?: string;
  username?: string;
  location?: string;
  status: DeviceStatus;
  decommission_date?: string;
  status_changed_at?: string;
  status_changed_by?: string;
  tags: string[];
  addresses: Address[];
  domains: string[];
  custom_fields?: CustomFieldValueInput[];
  created_at: string;
  updated_at: string;
}

export interface DeviceFilter {
  tags?: string[];
  datacenter_id?: string;
  network_id?: string;
  pool_id?: string;
  status?: DeviceStatus;
  stale?: boolean;
  stale_days?: number;
}

export interface DeviceStatusCounts {
  planned: number;
  active: number;
  maintenance: number;
  decommissioned: number;
}

export interface Datacenter {
  id: string;
  name: string;
  location: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface Network {
  id: string;
  name: string;
  subnet: string;
  vlan_id: number;
  datacenter_id: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface NetworkPool {
  id: string;
  network_id: string;
  name: string;
  start_ip: string;
  end_ip: string;
  description: string;
  tags: string[];
  created_at: string;
  updated_at: string;
}

export interface NetworkUtilization {
  network_id: string;
  total_ips: number;
  used_ips: number;
  available_ips: number;
  utilization: number;
}

export interface IPStatus {
  ip: string;
  status: 'available' | 'used' | 'reserved' | 'conflicted';
  device_id?: string;
}

export interface Conflict {
  id: string;
  type: 'duplicate_ip' | 'overlapping_subnet';
  status: 'active' | 'resolved' | 'ignored';
  description: string;
  ip_address?: string;
  device_ids?: string[];
  device_names?: string[];
  network_ids?: string[];
  network_names?: string[];
  subnets?: string[];
  detected_at: string;
  resolved_at?: string;
  resolved_by?: string;
  notes?: string;
}

export interface ConflictResolution {
  conflict_id: string;
  keep_device_id?: string;
  keep_network_id?: string;
  notes: string;
}

export interface ConflictType {
  duplicate_ip: 'duplicate_ip';
  overlapping_subnet: 'overlapping_subnet';
}

export interface Reservation {
  id: string;
  pool_id: string;
  ip_address: string;
  hostname?: string;
  purpose?: string;
  reserved_by: string;
  reserved_at: string;
  expires_at?: string;
  status: 'active' | 'expired' | 'claimed' | 'released';
  notes?: string;
  created_at: string;
  updated_at: string;
}

export interface ReservationFilter {
  pool_id?: string;
  status?: 'active' | 'expired' | 'claimed' | 'released';
  reserved_by?: string;
  ip?: string;
}

export interface CreateReservationRequest {
  pool_id: string;
  ip_address?: string;
  hostname?: string;
  purpose?: string;
  expires_in_days?: number;
  notes?: string;
}

export interface UpdateReservationRequest {
  hostname?: string;
  purpose?: string;
  expires_in_days?: number;
  notes?: string;
}

export interface ServiceInfo {
  port: number;
  protocol: string;
  service: string;
  version: string;
}

export interface DiscoveredDevice {
  id: string;
  ip: string;
  mac_address: string;
  hostname: string;
  network_id: string;
  status: string;
  confidence: number;
  os_guess: string;
  vendor: string;
  open_ports: number[];
  services: ServiceInfo[];
  first_seen: string;
  last_seen: string;
  promoted_to_device_id?: string;
  promoted_at?: string;
  created_at: string;
  updated_at: string;
}

export interface DiscoveryScan {
  id: string;
  network_id: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  scan_type: 'quick' | 'full' | 'deep';
  total_hosts: number;
  scanned_hosts: number;
  found_hosts: number;
  progress_percent: number;
  error_message?: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface DiscoveryRule {
  id: string;
  network_id: string;
  enabled: boolean;
  scan_type: 'quick' | 'full' | 'deep';
  interval_hours: number;
  exclude_ips: string;
  created_at: string;
  updated_at: string;
}

export interface ScanProfile {
  id: string;
  name: string;
  scan_type: 'quick' | 'full' | 'deep' | 'custom';
  ports?: number[];
  enable_snmp?: boolean;
  enable_ssh?: boolean;
  timeout_sec: number;
  max_workers: number;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface DeviceRelationship {
  parent_id: string;
  child_id: string;
  type: 'contains' | 'connected_to' | 'depends_on';
  notes: string;
  created_at: string;
}

export interface NavItem {
  label: string;
  path: string;
  icon?: string;
  order: number;
  required_permissions?: { resource: string; action: string }[];
}

export interface UserInfo {
  id: string;
  username: string;
  email: string;
  roles: string[];
  permissions?: Permission[];
  is_admin?: boolean;
}

export interface User {
  id: string;
  username: string;
  email: string;
  full_name: string;
  is_active: boolean;
  is_admin: boolean;
  created_at: string;
  updated_at: string;
  last_login_at?: string;
  roles?: Role[];
}

export interface CurrentUser extends User {
  permissions: Permission[];
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  user: CurrentUser;
  expires_at: string;
}

export interface UserFilter {
  username?: string;
  email?: string;
  is_active?: boolean;
  is_admin?: boolean;
}

export interface CreateUserRequest {
  username: string;
  password: string;
  email: string;
  full_name?: string;
  is_admin?: boolean;
}

export interface UpdateUserRequest {
  email?: string;
  full_name?: string;
  is_active?: boolean;
  is_admin?: boolean;
}

export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}

export interface Permission {
  id: string;
  name: string;
  resource: string;
  action: string;
  created_at: string;
}

export interface Role {
  id: string;
  name: string;
  description: string;
  is_system: boolean;
  created_at: string;
  updated_at: string;
  permissions?: Permission[];
}

export interface RoleFilter {
  name?: string;
  is_system?: boolean;
}

export interface CreateRoleRequest {
  name: string;
  description?: string;
  permissions?: string[];
}

export interface UpdateRoleRequest {
  description?: string;
  permissions?: string[];
}

export interface UIConfig {
  edition: 'oss';
  features: string[];
  nav_items: NavItem[];
  user?: UserInfo;
}

export interface APIError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}

export interface SearchResult {
  type: 'device' | 'network' | 'datacenter';
  device?: Device;
  network?: Network;
  datacenter?: Datacenter;
}

// Dashboard types
export interface RecentDiscovery {
  id: string;
  ip: string;
  hostname?: string;
  vendor?: string;
  network_id?: string;
  first_seen: string;
  last_seen: string;
}

export interface NetworkUtilizationSummary {
  network_id: string;
  network_name: string;
  subnet: string;
  total_ips: number;
  used_ips: number;
  utilization: number;
}

export interface UtilizationTrendPoint {
  timestamp: string;
  utilization: number;
  used_ips: number;
}

export interface StaleDevice {
  id: string;
  name: string;
  hostname?: string;
}

export interface DashboardStats {
  total_devices: number;
  total_networks: number;
  total_pools: number;
  total_datacenters: number;
  device_status_counts: DeviceStatusCounts;
  discovered_devices: number;
  recent_discoveries: RecentDiscovery[];
  overall_utilization: number;
  network_utilization: NetworkUtilizationSummary[];
  stale_devices: number;
  stale_threshold_days: number;
  stale_device_list: StaleDevice[];
}

// Webhook types
export type EventType =
  | 'device.created'
  | 'device.updated'
  | 'device.deleted'
  | 'device.promoted'
  | 'network.created'
  | 'network.updated'
  | 'network.deleted'
  | 'discovery.started'
  | 'discovery.completed'
  | 'discovery.device_found'
  | 'conflict.detected'
  | 'conflict.resolved'
  | 'pool.utilization_high';

export interface EventTypeOption {
  value: EventType;
  label: string;
}

export type DeliveryStatus = 'pending' | 'success' | 'failed' | 'retrying' | 'abandoned';

export interface Webhook {
  id: string;
  name: string;
  url: string;
  has_secret: boolean;
  events: EventType[];
  active: boolean;
  description?: string;
  created_at: string;
  updated_at: string;
  created_by?: string;
}

export interface WebhookDelivery {
  id: string;
  webhook_id: string;
  event_type: EventType;
  payload: string;
  response_code?: number;
  response_body?: string;
  error?: string;
  duration_ms: number;
  status: DeliveryStatus;
  attempt_number: number;
  next_retry?: string;
  created_at: string;
}

export interface CreateWebhookRequest {
  name: string;
  url: string;
  secret?: string;
  events: EventType[];
  active: boolean;
  description?: string;
}

export interface UpdateWebhookRequest {
  name?: string;
  url?: string;
  secret?: string;
  events?: EventType[];
  active?: boolean;
  description?: string;
}

// Custom Field Types
export type CustomFieldType = 'text' | 'number' | 'boolean' | 'select';

export interface CustomFieldDefinition {
  id: string;
  name: string;
  key: string;
  type: CustomFieldType;
  required: boolean;
  options?: string[];
  description: string;
  created_at: string;
  updated_at: string;
}

export interface CustomFieldValue {
  id: string;
  device_id: string;
  field_id: string;
  string_value: string;
  number_value?: number;
  bool_value?: boolean;
}

export interface CustomFieldValueInput {
  field_id: string;
  value: string | number | boolean | null;
}

export interface CustomFieldWithDefinition {
  definition: CustomFieldDefinition;
  value: string | number | boolean | null;
}

export interface CreateCustomFieldDefinitionRequest {
  name: string;
  key: string;
  type: CustomFieldType;
  required: boolean;
  options?: string[];
  description?: string;
}

export interface UpdateCustomFieldDefinitionRequest {
  name?: string;
  key?: string;
  type?: CustomFieldType;
  required?: boolean;
  options?: string[];
  description?: string;
}

// Circuit Types
export type CircuitType = 'fiber' | 'copper' | 'microwave' | 'dark_fiber';
export type CircuitStatus = 'active' | 'inactive' | 'planned' | 'decommissioned';

export interface Circuit {
  id: string;
  name: string;
  circuit_id: string;
  provider: string;
  type: CircuitType;
  status: CircuitStatus;
  capacity_mbps: number;
  datacenter_a_id?: string;
  datacenter_b_id?: string;
  device_a_id?: string;
  device_b_id?: string;
  port_a?: string;
  port_b?: string;
  ip_address_a?: string;
  ip_address_b?: string;
  vlan_id?: number;
  description?: string;
  install_date?: string;
  terminate_date?: string;
  monthly_cost?: number;
  contract_number?: string;
  contact_name?: string;
  contact_phone?: string;
  contact_email?: string;
  tags?: string[];
  created_at: string;
  updated_at: string;
}

export interface CircuitFilter {
  provider?: string;
  status?: string;
  datacenter_id?: string;
  type?: string;
  tags?: string[];
}

export interface CreateCircuitRequest {
  name: string;
  circuit_id: string;
  provider: string;
  type?: CircuitType;
  status?: CircuitStatus;
  capacity_mbps?: number;
  datacenter_a_id?: string;
  datacenter_b_id?: string;
  device_a_id?: string;
  device_b_id?: string;
  port_a?: string;
  port_b?: string;
  ip_address_a?: string;
  ip_address_b?: string;
  vlan_id?: number;
  description?: string;
  install_date?: string;
  terminate_date?: string;
  monthly_cost?: number;
  contract_number?: string;
  contact_name?: string;
  contact_phone?: string;
  contact_email?: string;
  tags?: string[];
}

export interface UpdateCircuitRequest {
  name?: string;
  circuit_id?: string;
  provider?: string;
  type?: CircuitType;
  status?: CircuitStatus;
  capacity_mbps?: number;
  datacenter_a_id?: string;
  datacenter_b_id?: string;
  device_a_id?: string;
  device_b_id?: string;
  port_a?: string;
  port_b?: string;
  ip_address_a?: string;
  ip_address_b?: string;
  vlan_id?: number;
  description?: string;
  install_date?: string;
  terminate_date?: string;
  monthly_cost?: number;
  contract_number?: string;
  contact_name?: string;
  contact_phone?: string;
  contact_email?: string;
  tags?: string[];
}

// NAT Types
export type NATProtocol = 'tcp' | 'udp' | 'any';

export interface NATMapping {
  id: string;
  name: string;
  external_ip: string;
  external_port: number;
  internal_ip: string;
  internal_port: number;
  protocol: NATProtocol;
  device_id?: string;
  description: string;
  enabled: boolean;
  datacenter_id?: string;
  network_id?: string;
  tags?: string[];
  created_at: string;
  updated_at: string;
}

export interface NATFilter {
  external_ip?: string;
  internal_ip?: string;
  protocol?: NATProtocol;
  device_id?: string;
  datacenter_id?: string;
  network_id?: string;
  enabled?: boolean;
}

export interface CreateNATRequest {
  name: string;
  external_ip: string;
  external_port: number;
  internal_ip: string;
  internal_port: number;
  protocol?: NATProtocol;
  device_id?: string;
  description?: string;
  enabled?: boolean;
  datacenter_id?: string;
  network_id?: string;
  tags?: string[];
}

export interface UpdateNATRequest {
  name?: string;
  external_ip?: string;
  external_port?: number;
  internal_ip?: string;
  internal_port?: number;
  protocol?: NATProtocol;
  device_id?: string;
  description?: string;
  enabled?: boolean;
  datacenter_id?: string;
  network_id?: string;
  tags?: string[];
}

// DNS Types
export type DNSProviderType = 'technitium' | 'powerdns' | 'bind';
export type SyncStatus = 'success' | 'failed' | 'partial';
export type RecordSyncStatus = 'synced' | 'pending' | 'failed';

export interface DNSProvider {
  id: string;
  name: string;
  type: DNSProviderType;
  endpoint: string;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface DNSProviderFilter {
  type?: DNSProviderType;
}

export interface CreateDNSProviderRequest {
  name: string;
  type: DNSProviderType;
  endpoint: string;
  token: string;
  description?: string;
}

export interface UpdateDNSProviderRequest {
  name?: string;
  endpoint?: string;
  token?: string;
  description?: string;
}

export interface DNSZone {
  id: string;
  name: string;
  provider_id: string;
  network_id?: string;
  auto_sync: boolean;
  create_ptr: boolean;
  ptr_zone?: string;
  ttl: number;
  description?: string;
  last_sync_at?: string;
  last_sync_status?: SyncStatus;
  last_sync_error?: string;
  created_at: string;
  updated_at: string;
}

export interface DNSZoneFilter {
  provider_id?: string;
  network_id?: string;
  auto_sync?: boolean;
}

export interface CreateDNSZoneRequest {
  name: string;
  provider_id: string;
  network_id?: string;
  auto_sync: boolean;
  create_ptr: boolean;
  ptr_zone?: string;
  ttl: number;
  description?: string;
}

export interface UpdateDNSZoneRequest {
  name?: string;
  network_id?: string;
  auto_sync?: boolean;
  create_ptr?: boolean;
  ptr_zone?: string;
  ttl?: number;
  description?: string;
}

export interface DNSRecord {
  id: string;
  zone_id: string;
  device_id?: string;
  address_id?: string;
  name: string;
  type: string;
  value: string;
  ttl: number;
  sync_status: RecordSyncStatus;
  last_sync_at?: string;
  error_message?: string;
  created_at: string;
  updated_at: string;
}

export interface DNSRecordFilter {
  zone_id?: string;
  device_id?: string;
  type?: string;
  sync_status?: RecordSyncStatus;
  link_status?: string;
}

export interface CreateDNSRecordRequest {
  zone_id: string;
  device_id?: string;
  name: string;
  type: string;
  value: string;
  ttl: number;
}

export interface UpdateDNSRecordRequest {
  device_id?: string;
  name?: string;
  type?: string;
  value?: string;
  ttl?: number;
}

export interface SyncResult {
  success: boolean;
  total: number;
  synced: number;
  failed: number;
  error?: string;
  failed_ids?: string[];
}

export interface ImportResult {
  success: boolean;
  total: number;
  imported: number;
  linked: number;
  skipped: number;
  failed: number;
  error?: string;
  skipped_ids?: string[];
  failed_ids?: string[];
}
