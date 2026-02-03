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
  created_at: string;
}

export interface NavItem {
  label: string;
  path: string;
  icon?: string;
  order: number;
}

export interface UserInfo {
  id: string;
  email: string;
  name: string;
  roles: string[];
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
