// Shared types for Rackd Web UI

export interface Address {
  ip: string;
  port?: number;
  type: string;
  label: string;
  network_id?: string;
  switch_port?: string;
  pool_id?: string;
}

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
  tags: string[];
  addresses: Address[];
  domains: string[];
  created_at: string;
  updated_at: string;
}

export interface DeviceFilter {
  tags?: string[];
  datacenter_id?: string;
  network_id?: string;
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
  status: 'available' | 'used' | 'reserved';
  device_id?: string;
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
  required_permissions?: {resource: string; action: string}[];
}

export interface UserInfo {
  id: string;
  username: string;
  email: string;
  roles: string[];
  permissions?: Permission[];
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
