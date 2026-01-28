// Search Component for Rackd Web UI

import type { Device, Network, Datacenter } from '../core/types';
import { api, RackdAPIError } from '../core/api';
import { debounce } from '../core/utils';

interface SearchResult {
  type: 'device' | 'network' | 'datacenter';
  device?: Device;
  network?: Network;
  datacenter?: Datacenter;
}

interface SearchData {
  query: string;
  results: SearchResult[];
  loading: boolean;
  error: string;
  showResults: boolean;
  debouncedSearch: () => void;
  init(): void;
  search(): Promise<void>;
  onInput(): void;
  onFocus(): void;
  onBlur(): void;
  clear(): void;
}

export function globalSearch(): SearchData {
  return {
    query: '',
    results: [] as SearchResult[],
    loading: false,
    error: '',
    showResults: false,
    debouncedSearch: () => {},

    init(): void {
      this.debouncedSearch = debounce(() => this.search(), 300);
    },

    async search(): Promise<void> {
      if (!this.query.trim()) {
        this.results = [];
        return;
      }

      this.loading = true;
      this.error = '';

      try {
        const q = this.query.trim().toLowerCase();

        const [devices, networks, datacenters] = await Promise.all([
          api.listDevices().catch((e) => {
            console.error('Failed to load devices:', e);
            return [];
          }),
          api.listNetworks().catch((e) => {
            console.error('Failed to load networks:', e);
            return [];
          }),
          api.listDatacenters().catch((e) => {
            console.error('Failed to load datacenters:', e);
            return [];
          }),
        ]);

        console.log('Search data loaded:', { devices: devices?.length, networks: networks?.length, datacenters: datacenters?.length });

        const results: SearchResult[] = [];

        if (devices && devices.length > 0) {
          for (const d of devices) {
            const deviceStr = [
              d.name || '',
              d.hostname || '',
              d.make_model || '',
              d.location || '',
              d.description || '',
              ...(d.tags || []).join(' ')
            ].join(' ').toLowerCase();

            if (deviceStr.includes(q)) {
              results.push({ type: 'device', device: d });
            }
          }
        }

        if (networks && networks.length > 0) {
          for (const n of networks) {
            const networkStr = [
              n.name || '',
              n.subnet || '',
              n.description || ''
            ].join(' ').toLowerCase();

            if (networkStr.includes(q)) {
              results.push({ type: 'network', network: n });
            }
          }
        }

        if (datacenters && datacenters.length > 0) {
          for (const dc of datacenters) {
            const dcStr = [
              dc.name || '',
              dc.location || '',
              dc.description || ''
            ].join(' ').toLowerCase();

            if (dcStr.includes(q)) {
              results.push({ type: 'datacenter', datacenter: dc });
            }
          }
        }

        console.log('Search results:', results.length, 'query:', q);
        this.results = results;
      } catch (e) {
        console.error('Search error:', e);
        this.error = e instanceof RackdAPIError ? e.message : 'Search failed';
        this.results = [];
      } finally {
        this.loading = false;
      }
    },

    onInput(): void {
      this.showResults = true;
      this.debouncedSearch();
    },

    onFocus(): void {
      if (this.query.trim()) this.showResults = true;
    },

    onBlur(): void {
      // Keep results open for longer to allow clicking
      setTimeout(() => {
        this.showResults = false;
      }, 500);
    },

    clear(): void {
      this.query = '';
      this.results = [];
      this.showResults = false;
    },
  };
}
