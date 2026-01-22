// Search Component for Rackd Web UI

import type { Device } from '../core/types';
import { RackdAPI, RackdAPIError } from '../core/api';
import { debounce } from '../core/utils';

const api = new RackdAPI();

interface SearchData {
  query: string;
  results: Device[];
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
    results: [],
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
        this.results = await api.searchDevices(this.query.trim());
      } catch (e) {
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
      setTimeout(() => { this.showResults = false; }, 200);
    },

    clear(): void {
      this.query = '';
      this.results = [];
      this.showResults = false;
    },
  };
}
