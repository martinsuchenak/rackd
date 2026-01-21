# Web UI Architecture

The Web UI is built with TypeScript, Alpine.js, and TailwindCSS v4. This document covers the architecture patterns for Enterprise UI extension and mobile-ready code organization.

## Directory Structure

```text
webui/
├── src/
│   ├── core/                 # Shared, extractable code (mobile-ready)
│   │   ├── api.ts            # API client (no DOM dependencies)
│   │   ├── types.ts          # Shared TypeScript types
│   │   └── utils.ts          # Pure utility functions
│   ├── components/           # Alpine.js components
│   │   ├── devices.ts        # Device management UI
│   │   ├── networks.ts       # Network management UI
│   │   ├── pools.ts          # IP pool management UI
│   │   ├── datacenters.ts    # Datacenter management UI
│   │   ├── discovery.ts      # Discovery UI
│   │   ├── search.ts         # Global search
│   │   └── nav.ts            # Navigation component
│   ├── app.ts                # Main app initialization
│   ├── index.html            # Main HTML
│   └── styles.css            # Tailwind base styles
├── dist/                     # Build output
├── package.json
└── tsconfig.json
```

### Core Types Definition

```typescript
// ===== webui/src/core/types.ts =====

// Shared TypeScript types for Rackd entities

export interface Device {
  id: string;
  name: string;
  description?: string;
  make_model?: string;
  os?: string;
  datacenter_id?: string;
  username?: string;
  location?: string;
  tags: string[];
  addresses: Address[];
  domains: string[];
  created_at: string;
  updated_at: string;
}

export interface Address {
  ip: string;
  port: number;
  type: 'ipv4' | 'ipv6';
  label?: string;
  network_id?: string;
  switch_port?: string;
  pool_id?: string;
}

export interface Datacenter {
  id: string;
  name: string;
  location?: string;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface Network {
  id: string;
  name: string;
  subnet: string;
  vlan_id: number;
  datacenter_id?: string;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface NetworkPool {
  id: string;
  network_id: string;
  name: string;
  start_ip: string;
  end_ip: string;
  description?: string;
  tags: string[];
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

export interface DiscoveredDevice {
  id: string;
  ip: string;
  mac_address: string;
  hostname?: string;
  network_id: string;
  status: 'online' | 'offline' | 'unknown';
  confidence: number;
  os_guess?: string;
  vendor?: string;
  open_ports: number[];
  services: ServiceInfo[];
  first_seen: string;
  last_seen: string;
  promoted_to_device_id?: string;
  promoted_at?: string;
}

export interface ServiceInfo {
  port: number;
  protocol: 'tcp' | 'udp';
  service: string;
  version?: string;
}

export interface DeviceFilter {
  tags?: string[];
  datacenter_id?: string;
  network_id?: string;
}

export interface IPStatus {
  ip: string;
  status: 'available' | 'used' | 'reserved';
  device_id?: string;
}

// API response types
export interface APIResponse<T> {
  data: T;
  error?: APIError;
}

export interface APIError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}
```

### Core API Client

```typescript
// ===== webui/src/core/api.ts =====

export class RackdAPI {
  private baseURL: string;

  constructor(options: { baseURL?: string } = {}) {
    this.baseURL = options.baseURL || '';
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown
  ): Promise<T> {
    const url = `${this.baseURL}${path}`;
    const options: RequestInit = {
      method,
      headers: {
        'Content-Type': 'application/json',
      },
    };

    if (body !== undefined) {
      options.body = JSON.stringify(body);
    }

    const response = await fetch(url, options);

    if (!response.ok) {
      const error = await response.json() as APIError;
      throw new APIError(error.code, error.message, error.details);
    }

    return response.json() as Promise<T>;
  }

  // Devices
  async listDevices(params?: DeviceFilter): Promise<Device[]> {
    const query = new URLSearchParams();
    if (params?.tags?.length) query.set('tags', params.tags.join(','));
    if (params?.datacenter_id) query.set('datacenter_id', params.datacenter_id);
    if (params?.network_id) query.set('network_id', params.network_id);

    return this.request<Device[]>('GET', `/api/devices?${query}`);
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

  // Networks
  async listNetworks(datacenter_id?: string): Promise<Network[]> {
    const query = datacenter_id ? `?datacenter_id=${datacenter_id}` : '';
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

  // Network Pools
  async listNetworkPools(networkId: string): Promise<NetworkPool[]> {
    return this.request<NetworkPool[]>('GET', `/api/networks/${networkId}/pools`);
  }

  async getNetworkPool(id: string): Promise<NetworkPool> {
    return this.request<NetworkPool>('GET', `/api/pools/${id}`);
  }

  async createNetworkPool(networkId: string, pool: Partial<NetworkPool>): Promise<NetworkPool> {
    return this.request<NetworkPool>('POST', `/api/networks/${networkId}/pools`, pool);
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
  async listScans(networkId?: string): Promise<DiscoveryScan[]> {
    const query = networkId ? `?network_id=${networkId}` : '';
    return this.request<DiscoveryScan[]>('GET', `/api/discovery/scans${query}`);
  }

  async startScan(networkId: string, scanType: 'quick' | 'full' | 'deep'): Promise<DiscoveryScan> {
    return this.request<DiscoveryScan>('POST', `/api/discovery/networks/${networkId}/scan`, {
      scan_type: scanType,
    });
  }

  async listDiscoveredDevices(networkId?: string): Promise<DiscoveredDevice[]> {
    const query = networkId ? `?network_id=${networkId}` : '';
    return this.request<DiscoveredDevice[]>('GET', `/api/discovery/devices${query}`);
  }

  async promoteDevice(discoveredId: string, name: string): Promise<Device> {
    return this.request<Device>('POST', `/api/discovery/devices/${discoveredId}/promote`, {
      discovered_id: discoveredId,
      name,
    });
  }

  async deleteDiscoveredDevice(id: string): Promise<void> {
    return this.request<void>('DELETE', `/api/discovery/devices/${id}`);
  }

  // Configuration
  async getConfig(): Promise<{
    edition: 'oss' | 'enterprise';
    features: string[];
    nav_items: NavItem[];
    user?: UserInfo;
  }> {
    return this.request('/api/config');
  }
}

// Custom API Error class
export class APIError extends Error {
  constructor(
    public code: string,
    message: string,
    public details?: Record<string, unknown>
  ) {
    super(message);
  }
}
```

### Core Utilities

```typescript
// ===== webui/src/core/utils.ts =====

export function formatDate(date: string | Date): string {
  const d = typeof date === 'string' ? new Date(date) : date;
  return d.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

export function formatDateTime(date: string | Date): string {
  const d = typeof date === 'string' ? new Date(date) : date;
  return d.toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function debounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeout: number | null = null;
  return (...args: Parameters<T>) => {
    if (timeout !== null) clearTimeout(timeout);
    timeout = window.setTimeout(() => func(...args), wait);
  };
}

export function copyToClipboard(text: string): Promise<boolean> {
  if (navigator.clipboard && navigator.clipboard.writeText) {
    return navigator.clipboard.writeText(text).then(() => true);
  }
  return Promise.resolve(false);
}

export function getIPType(ip: string): 'ipv4' | 'ipv6' {
  return ip.includes(':') ? 'ipv6' : 'ipv4';
}

export function isValidIP(ip: string): boolean {
  const ipv4Regex = /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/;
  const ipv6Regex = /^(([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$/;
  return ipv4Regex.test(ip) || ipv6Regex.test(ip);
}

export function isValidCIDR(cidr: string): boolean {
  const parts = cidr.split('/');
  if (parts.length !== 2) return false;
  if (!isValidIP(parts[0])) return false;
  const prefix = parseInt(parts[1], 10);
  return !isNaN(prefix) && prefix >= 0 && prefix <= 128;
}
```

## Enterprise UI Extension Pattern

The Enterprise version extends the OSS UI through two mechanisms:

### 1. API-Driven Feature Discovery

The backend exposes available features to the frontend via a configuration endpoint:

```go
// ===== OSS REPO: internal/api/config_handlers.go =====
package api

import (
    "encoding/json"
    "net/http"
)

// UIConfig represents frontend configuration
type UIConfig struct {
    Edition    string      `json:"edition"`     // "oss" or "enterprise"
    Features   []string    `json:"features"`    // enabled feature names
    NavItems   []NavItem   `json:"nav_items"`   // additional navigation items
    UserInfo   *UserInfo   `json:"user,omitempty"` // authenticated user (Enterprise)
}

type NavItem struct {
    Label string `json:"label"`
    Path  string `json:"path"`
    Icon  string `json:"icon"`
    Order int    `json:"order"` // sort order in nav
}

type UserInfo struct {
    ID       string   `json:"id"`
    Username string   `json:"username"`
    Email    string   `json:"email"`
    Roles    []string `json:"roles"`
}

// UIConfigBuilder collects config from features
type UIConfigBuilder struct {
    config UIConfig
}

func NewUIConfigBuilder() *UIConfigBuilder {
    return &UIConfigBuilder{
        config: UIConfig{
            Edition:  "oss",
            Features: []string{},
            NavItems: []NavItem{},
        },
    }
}

func (b *UIConfigBuilder) SetEdition(edition string) {
    b.config.Edition = edition
}

func (b *UIConfigBuilder) AddFeature(name string) {
    b.config.Features = append(b.config.Features, name)
}

func (b *UIConfigBuilder) AddNavItem(item NavItem) {
    b.config.NavItems = append(b.config.NavItems, item)
}

func (b *UIConfigBuilder) SetUser(user *UserInfo) {
    b.config.UserInfo = user
}

func (b *UIConfigBuilder) Build() UIConfig {
    return b.config
}

func (b *UIConfigBuilder) Handler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(b.config)
    }
}
```

#### Updated Feature Interface

```go
// ===== OSS REPO: internal/server/server.go =====

// Feature interface for Enterprise extension
type Feature interface {
    Name() string
    RegisterRoutes(mux *http.ServeMux)
    RegisterMCPTools(mcpServer interface{})
    ConfigureUI(builder *api.UIConfigBuilder) // UI configuration
}
```

#### Server Integration

```go
// In server.Run()
func Run(cfg *config.Config, store storage.ExtendedStorage, features ...Feature) error {
    mux := http.NewServeMux()

    // Build UI config
    uiConfig := api.NewUIConfigBuilder()

    // Register Enterprise features and collect UI config
    for _, f := range features {
        log.Info("Registering feature", "name", f.Name())
        f.RegisterRoutes(mux)
        f.RegisterMCPTools(mcpServer.Inner())
        f.ConfigureUI(uiConfig)  // Collect UI configuration
    }

    // Set edition based on features
    if len(features) > 0 {
        uiConfig.SetEdition("enterprise")
    }

    // Register config endpoint
    mux.HandleFunc("GET /api/config", uiConfig.Handler())

    // ... rest of server setup
}
```

### 2. Build-Time Asset Composition

The Enterprise repository overlays additional UI assets on top of OSS:

```text
OSS Repository                    Enterprise Repository
webui/src/                        webui/src/
├── core/                         ├── components/
│   ├── api.ts                    │   ├── sso.ts        (new)
│   └── types.ts                  │   ├── rbac.ts       (new)
├── components/                   │   ├── audit.ts      (new)
│   ├── devices.ts                │   └── nav.ts        (override)
│   ├── nav.ts                    └── enterprise.ts        (enterprise init)
│   └── ...
└── app.ts
```

#### Enterprise Build Process

```makefile
# Enterprise Makefile - extends OSS build

OSS_WEBUI := $(shell go list -m -f '{{.Dir}}' github.com/martinsuchenak/rackd)/webui
ENTERPRISE_WEBUI := ./webui
MERGED_WEBUI := ./build/webui

## ui-build: Build merged UI (OSS + Enterprise)
ui-build:
	@echo "Merging UI assets..."
	@rm -rf $(MERGED_WEBUI)
	@mkdir -p $(MERGED_WEBUI)
	# Copy OSS base
	@cp -r $(OSS_WEBUI)/src/* $(MERGED_WEBUI)/
	# Overlay Enterprise (overwrites matching files)
	@cp -r $(ENTERPRISE_WEBUI)/src/* $(MERGED_WEBUI)/
	# Build merged UI
	@cd $(MERGED_WEBUI) && bun install && bun run build
	# Copy to embed directory
	@cp -r $(MERGED_WEBUI)/dist/* ./internal/ui/assets/
```

### Enterprise Feature Example (SSO)

```typescript
// ===== ENTERPRISE REPO: webui/src/components/sso.ts =====

export function initSSO(Alpine: typeof window.Alpine) {
  Alpine.data('sso', () => ({
    loginUrl: '/auth/login',
    logoutUrl: '/auth/logout',

    get user() {
      return window.rackdConfig?.user;
    },

    get isLoggedIn(): boolean {
      return !!this.user;
    },

    login() {
      window.location.href = this.loginUrl;
    },

    logout() {
      window.location.href = this.logoutUrl;
    },
  }));
}
```

```typescript
// ===== ENTERPRISE REPO: webui/src/enterprise.ts =====

import { initSSO } from './components/sso';
import { initRBAC } from './components/rbac';
import { initAudit } from './components/audit';

// Register enterprise initialization
window.rackdEnterprise = {
  init() {
    const config = window.rackdConfig;

    if (config.features.includes('sso')) {
      initSSO(window.Alpine);
    }

    if (config.features.includes('rbac')) {
      initRBAC(window.Alpine);
    }

    if (config.features.includes('audit')) {
      initAudit(window.Alpine);
    }
  }
};
```

### HTML Template with Feature Conditionals

```html
<!-- webui/src/index.html -->
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Rackd - IPAM & Device Inventory</title>
  <link rel="stylesheet" href="/output.css">
</head>
<body class="bg-gray-50 dark:bg-gray-900">
  <div x-data="nav" class="min-h-screen">
    <!-- Navigation -->
    <nav class="bg-white dark:bg-gray-800 shadow">
      <div class="max-w-7xl mx-auto px-4">
        <div class="flex justify-between h-16">
          <!-- Logo -->
          <div class="flex items-center">
            <span class="text-xl font-bold">Rackd</span>
            <span x-show="isEnterprise" class="ml-2 text-xs bg-blue-100 text-blue-800 px-2 py-1 rounded">
              Enterprise
            </span>
          </div>

          <!-- Nav Items -->
          <div class="flex space-x-4">
            <template x-for="item in items" :key="item.path">
              <a :href="item.path"
                 class="px-3 py-2 rounded-md text-sm font-medium"
                 x-text="item.label">
              </a>
            </template>
          </div>

          <!-- User Menu (Enterprise SSO) -->
          <div x-show="hasFeature('sso')" x-data="sso" class="flex items-center">
            <template x-if="isLoggedIn">
              <div class="flex items-center space-x-2">
                <span x-text="user.username"></span>
                <button @click="logout" class="text-sm text-gray-500">Logout</button>
              </div>
            </template>
            <template x-if="!isLoggedIn">
              <button @click="login" class="btn-primary">Login</button>
            </template>
          </div>
        </div>
      </div>
    </nav>

    <!-- Main Content -->
    <main class="max-w-7xl mx-auto py-6 px-4">
      <!-- Content rendered by Alpine components -->
    </main>
  </div>

  <script type="module" src="/app.js"></script>
</body>
</html>
```

## Component Implementations

### Device Management Component

```typescript
// ===== webui/src/components/devices.ts =====

import { Alpine } from 'alpinejs';
import type { Device, DeviceFilter } from '../core/types';

export function initDevices(Alpine: typeof window.Alpine) {
  Alpine.data('devices', () => ({
    devices: [] as Device[],
    filter: { search: '', tags: [] as string[] } as DeviceFilter,
    loading: false,
    showModal: false,
    selectedDevice: null as Device | null,
    error: null as string | null,

    async loadDevices() {
      this.loading = true;
      this.error = null;
      try {
        const query = new URLSearchParams();
        if (this.filter.search) query.set('q', this.filter.search);
        if (this.filter.tags.length) query.set('tags', this.filter.tags.join(','));

        const response = await fetch(`/api/devices?${query}`);
        if (!response.ok) throw new Error('Failed to load devices');
        this.devices = await response.json();
      } catch (e) {
        this.error = e instanceof Error ? e.message : 'Unknown error';
      } finally {
        this.loading = false;
      }
    },

    async deleteDevice(id: string) {
      if (!confirm('Are you sure you want to delete this device?')) return;

      this.loading = true;
      try {
        const response = await fetch(`/api/devices/${id}`, { method: 'DELETE' });
        if (!response.ok) throw new Error('Failed to delete device');
        await this.loadDevices();
      } catch (e) {
        this.error = e instanceof Error ? e.message : 'Unknown error';
      } finally {
        this.loading = false;
      }
    },

    openModal(device: Device) {
      this.selectedDevice = device;
      this.showModal = true;
    },

    closeModal() {
      this.selectedDevice = null;
      this.showModal = false;
    },

    async saveDevice() {
      if (!this.selectedDevice) return;

      this.loading = true;
      this.error = null;
      try {
        const method = this.selectedDevice.id ? 'PUT' : 'POST';
        const url = this.selectedDevice.id ? `/api/devices/${this.selectedDevice.id}` : '/api/devices';
        const response = await fetch(url, {
          method,
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(this.selectedDevice),
        });
        if (!response.ok) throw new Error('Failed to save device');
        await this.loadDevices();
        this.closeModal();
      } catch (e) {
        this.error = e instanceof Error ? e.message : 'Unknown error';
      } finally {
        this.loading = false;
      }
    },
  }));
}
```

### Network Management Component

```typescript
// ===== webui/src/components/networks.ts =====

import { Alpine } from 'alpinejs';
import type { Network } from '../core/types';

export function initNetworks(Alpine: typeof window.Alpine) {
  Alpine.data('networks', () => ({
    networks: [] as Network[],
    loading: false,
    showModal: false,
    selectedNetwork: null as Network | null,

    async loadNetworks() {
      this.loading = true;
      try {
        const response = await fetch('/api/networks');
        if (!response.ok) throw new Error('Failed to load networks');
        this.networks = await response.json();
      } catch (e) {
        console.error(e);
      } finally {
        this.loading = false;
      }
    },

    openModal(network?: Network) {
      this.selectedNetwork = network || { name: '', subnet: '', vlan_id: 0, datacenter_id: '', description: '' } as Network;
      this.showModal = true;
    },

    closeModal() {
      this.showModal = false;
      this.selectedNetwork = null;
    },

    async saveNetwork() {
      if (!this.selectedNetwork) return;

      this.loading = true;
      try {
        const method = this.selectedNetwork.id ? 'PUT' : 'POST';
        const url = this.selectedNetwork.id ? `/api/networks/${this.selectedNetwork.id}` : '/api/networks';
        const response = await fetch(url, {
          method,
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(this.selectedNetwork),
        });
        if (!response.ok) throw new Error('Failed to save network');
        await this.loadNetworks();
        this.closeModal();
      } catch (e) {
        console.error(e);
      } finally {
        this.loading = false;
      }
    },
  }));
}
```

### Datacenter Management Component

```typescript
// ===== webui/src/components/datacenters.ts =====

import { Alpine } from 'alpinejs';
import type { Datacenter } from '../core/types';

export function initDatacenters(Alpine: typeof window.Alpine) {
  Alpine.data('datacenters', () => ({
    datacenters: [] as Datacenter[],
    loading: false,
    showModal: false,
    selectedDatacenter: null as Datacenter | null,

    async loadDatacenters() {
      this.loading = true;
      try {
        const response = await fetch('/api/datacenters');
        if (!response.ok) throw new Error('Failed to load datacenters');
        this.datacenters = await response.json();
      } catch (e) {
        console.error(e);
      } finally {
        this.loading = false;
      }
    },

    openModal(datacenter?: Datacenter) {
      this.selectedDatacenter = datacenter || { name: '', location: '', description: '' } as Datacenter;
      this.showModal = true;
    },

    closeModal() {
      this.showModal = false;
      this.selectedDatacenter = null;
    },

    async saveDatacenter() {
      if (!this.selectedDatacenter) return;

      this.loading = true;
      try {
        const method = this.selectedDatacenter.id ? 'PUT' : 'POST';
        const url = this.selectedDatacenter.id ? `/api/datacenters/${this.selectedDatacenter.id}` : '/api/datacenters';
        const response = await fetch(url, {
          method,
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(this.selectedDatacenter),
        });
        if (!response.ok) throw new Error('Failed to save datacenter');
        await this.loadDatacenters();
        this.closeModal();
      } catch (e) {
        console.error(e);
      } finally {
        this.loading = false;
      }
    },
  }));
}
```

### Discovery Component

```typescript
// ===== webui/src/components/discovery.ts =====

import { Alpine } from 'alpinejs';
import type { DiscoveryScan, DiscoveredDevice } from '../core/types';

export function initDiscovery(Alpine: typeof window.Alpine) {
  Alpine.data('discovery', () => ({
    scans: [] as DiscoveryScan[],
    discoveredDevices: [] as DiscoveredDevice[],
    loading: false,
    selectedNetwork: '',

    async loadScans() {
      this.loading = true;
      try {
        const response = await fetch('/api/discovery/scans');
        if (!response.ok) throw new Error('Failed to load scans');
        this.scans = await response.json();
      } catch (e) {
        console.error(e);
      } finally {
        this.loading = false;
      }
    },

    async loadDiscovered(networkId: string) {
      this.loading = true;
      try {
        const response = await fetch(`/api/discovery/devices?network_id=${networkId}`);
        if (!response.ok) throw new Error('Failed to load discovered devices');
        this.discoveredDevices = await response.json();
      } catch (e) {
        console.error(e);
      } finally {
        this.loading = false;
      }
    },

    async startScan(networkId: string, scanType: string = 'full') {
      this.loading = true;
      try {
        const response = await fetch(`/api/discovery/networks/${networkId}/scan`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ scan_type: scanType }),
        });
        if (!response.ok) throw new Error('Failed to start scan');
        await this.loadScans();
      } catch (e) {
        console.error(e);
      } finally {
        this.loading = false;
      }
    },

    async promoteDevice(discoveredId: string, name: string) {
      this.loading = true;
      try {
        const response = await fetch(`/api/discovery/devices/${discoveredId}/promote`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ discovered_id: discoveredId, name }),
        });
        if (!response.ok) throw new Error('Failed to promote device');
        await this.loadDiscovered(this.selectedNetwork);
      } catch (e) {
        console.error(e);
      } finally {
        this.loading = false;
      }
    },
  }));
}
```

### Global Search Component

```typescript
// ===== webui/src/components/search.ts =====

import { Alpine } from 'alpinejs';

export function initSearch(Alpine: typeof window.Alpine) {
  Alpine.data('search', () => ({
    query: '',
    results: [] as any[],
    showResults: false,
    loading: false,

    async search() {
      if (this.query.length < 2) {
        this.showResults = false;
        return;
      }

      this.loading = true;
      try {
        const response = await fetch(`/api/devices/search?q=${encodeURIComponent(this.query)}`);
        if (!response.ok) throw new Error('Search failed');
        this.results = await response.json();
        this.showResults = this.results.length > 0;
      } catch (e) {
        console.error(e);
      } finally {
        this.loading = false;
      }
    },

    clear() {
      this.query = '';
      this.results = [];
      this.showResults = false;
    },

    selectResult(result: any) {
      window.location.href = `/devices/${result.id}`;
      this.clear();
    },
  }));
}
```

### IP Pool Management Component

```typescript
// ===== webui/src/components/pools.ts =====

import { Alpine } from 'alpinejs';
import type { NetworkPool, IPStatus } from '../core/types';

export function initPools(Alpine: typeof window.Alpine) {
  Alpine.data('pools', () => ({
    pools: [] as NetworkPool[],
    selectedPool: null as NetworkPool | null,
    heatmap: [] as IPStatus[],
    showModal: false,

    async loadPools(networkId: string) {
      try {
        const response = await fetch(`/api/networks/${networkId}/pools`);
        if (!response.ok) throw new Error('Failed to load pools');
        this.pools = await response.json();
      } catch (e) {
        console.error(e);
      }
    },

    async loadHeatmap(poolId: string) {
      try {
        const response = await fetch(`/api/pools/${poolId}/heatmap`);
        if (!response.ok) throw new Error('Failed to load heatmap');
        this.heatmap = await response.json();
      } catch (e) {
        console.error(e);
      }
    },

    async getNextIP(poolId: string) {
      try {
        const response = await fetch(`/api/pools/${poolId}/next-ip`);
        if (!response.ok) throw new Error('No available IPs');
        const { ip } = await response.json();
        navigator.clipboard.writeText(ip);
        alert(`IP ${ip} copied to clipboard!`);
      } catch (e) {
        console.error(e);
      }
    },
  }));
}
```

## Mobile App Extraction (Future)

When mobile apps are confirmed, extract's `core/` directory into a shared package:

```text
Future Structure:
packages/
 ├── api-client/              # Extracted from webui/src/core/
 │   ├── src/
 │   │   ├── api.ts
 │   │   ├── types.ts
 │   │   └── index.ts
 │   ├── package.json         # @rackd/api-client
 │   └── tsconfig.json
 ├── web/                     # Web UI (imports @rackd/api-client)
 │   └── ...
 └── mobile/                  # React Native / Expo app
     └── ...                  # Also imports @rackd/api-client
```

For now, keep `core/` as a directory within `webui/` - extraction is straightforward when needed.
