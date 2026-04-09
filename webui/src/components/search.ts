// Search Component for Rackd Web UI

import type { SearchResult } from '../core/types';
import { api, RackdAPIError } from '../core/api';
import { debounce } from '../core/utils';

interface SearchData {
  query: string;
  results: SearchResult[];
  loading: boolean;
  error: string;
  showResults: boolean;
  selectedIndex: number;
  debouncedSearch: () => void;
  init(): void;
  search(): Promise<void>;
  onInput(): void;
  onFocus(): void;
  onBlur(): void;
  onKeyDown(e: KeyboardEvent): void;
  selectResult(result: SearchResult): void;
  clear(): void;
  getResultUrl(result: SearchResult): string;
  getResultTitle(result: SearchResult): string;
  getResultSubtitle(result: SearchResult): string;
  getResultKey(result: SearchResult): string;
  shouldShowResults(): boolean;
  getActiveDescendant(): string | undefined;
  hasResults(): boolean;
  setSelectedIndex(index: number): void;
}

export function globalSearch(): SearchData {
  return {
    query: '',
    results: [] as SearchResult[],
    loading: false,
    error: '',
    showResults: false,
    selectedIndex: -1,
    debouncedSearch: () => { },

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
        this.results = await api.search(this.query.trim());
      } catch (e) {
        if (e instanceof RackdAPIError) {
          this.error = e.message;
        } else {
          this.error = 'Search failed';
        }
        console.error('Search error:', e);
        this.results = [];
      } finally {
        this.loading = false;
      }
    },

    onInput(): void {
      this.showResults = true;
      this.selectedIndex = -1;
      this.debouncedSearch();
    },

    onFocus(): void {
      if (this.query.trim()) this.showResults = true;
    },

    onBlur(): void {
      setTimeout(() => {
        this.showResults = false;
      }, 500);
    },

    onKeyDown(e: KeyboardEvent): void {
      if (!this.showResults || this.results.length === 0) return;

      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault();
          this.selectedIndex = Math.min(this.selectedIndex + 1, this.results.length - 1);
          break;
        case 'ArrowUp':
          e.preventDefault();
          this.selectedIndex = Math.max(this.selectedIndex - 1, -1);
          break;
        case 'Enter':
          e.preventDefault();
          if (this.selectedIndex >= 0) {
            this.selectResult(this.results[this.selectedIndex]);
          }
          break;
        case 'Escape':
          this.clear();
          break;
      }
    },

    selectResult(result: SearchResult): void {
      const path = result.type === 'device'
        ? `/devices/detail?id=${result.device?.id}`
        : result.type === 'network'
          ? `/networks/detail?id=${result.network?.id}`
          : `/datacenters/detail?id=${result.datacenter?.id}`;

      (this as any).$dispatch('nav', path);
      this.clear();
    },

    clear(): void {
      this.query = '';
      this.results = [];
      this.showResults = false;
      this.selectedIndex = -1;
    },

    getResultUrl(result: SearchResult): string {
      if (result.type === 'device') return `/devices/detail?id=${result.device?.id || ''}`;
      if (result.type === 'network') return `/networks/detail?id=${result.network?.id || ''}`;
      if (result.type === 'datacenter') return `/datacenters/detail?id=${result.datacenter?.id || ''}`;
      return '#';
    },

    getResultTitle(result: SearchResult): string {
      if (result.type === 'device') return result.device?.name || '';
      if (result.type === 'network') return result.network?.name || '';
      if (result.type === 'datacenter') return result.datacenter?.name || '';
      return '';
    },

    getResultSubtitle(result: SearchResult): string {
      if (result.type === 'device') return result.device?.make_model || result.device?.hostname || '';
      if (result.type === 'network') return result.network?.subnet || '';
      if (result.type === 'datacenter') return result.datacenter?.location || '';
      return '';
    },

    getResultKey(result: SearchResult): string {
      const id = result.device?.id || result.network?.id || result.datacenter?.id || '';
      return `${result.type}-${id}`;
    },
    shouldShowResults(): boolean {
      return this.showResults && (this.results.length > 0 || this.loading);
    },
    getActiveDescendant(): string | undefined {
      return this.selectedIndex >= 0 ? `search-result-${this.selectedIndex}` : undefined;
    },
    hasResults(): boolean {
      return this.results.length > 0;
    },
    setSelectedIndex(index: number): void {
      this.selectedIndex = index;
    },
  };
}
