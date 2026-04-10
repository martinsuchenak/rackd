import type { LogEntry, LogFilter } from '../core/types';
import { api, RackdAPIError } from '../core/api';

type ModalType = '' | 'detail';

export function logsPage() {
  return {
    entries: [] as LogEntry[],
    loading: true,
    error: '',
    modalType: '' as ModalType,
    selectedEntry: null as LogEntry | null,
    hasNextPage: false,
    filters: {
      level: '',
      source: '',
      query: '',
      start_time: '',
      end_time: '',
      limit: 100,
      offset: 0,
    },

    get showDetailModal(): boolean {
      return this.modalType === 'detail';
    },

    async init(): Promise<void> {
      this.hydrateFiltersFromQuery();
      await this.load();
    },

    hydrateFiltersFromQuery(): void {
      const params = new URLSearchParams(window.location.search);
      this.filters.level = params.get('level') || '';
      this.filters.source = params.get('source') || '';
      this.filters.query = params.get('query') || '';
      this.filters.start_time = this.toInputDateTime(params.get('start_time'));
      this.filters.end_time = this.toInputDateTime(params.get('end_time'));
      this.filters.limit = Number(params.get('limit') || 100);
      this.filters.offset = Number(params.get('offset') || 0);
    },

    buildFilter(): LogFilter {
      return {
        level: this.filters.level || undefined,
        source: this.filters.source || undefined,
        query: this.filters.query || undefined,
        start_time: this.serializeDateFilter(this.filters.start_time),
        end_time: this.serializeDateFilter(this.filters.end_time),
        limit: this.filters.limit,
        offset: this.filters.offset,
      };
    },

    toInputDateTime(value: string | null): string {
      if (!value) {
        return '';
      }
      const date = new Date(value);
      if (Number.isNaN(date.getTime())) {
        return '';
      }
      const tzOffset = date.getTimezoneOffset() * 60000;
      return new Date(date.getTime() - tzOffset).toISOString().slice(0, 16);
    },

    serializeDateFilter(value: string): string | undefined {
      if (!value) {
        return undefined;
      }
      return new Date(value).toISOString();
    },

    syncQuery(): void {
      const params = new URLSearchParams();
      const filter = this.buildFilter();
      for (const [key, value] of Object.entries(filter)) {
        if (value !== undefined && value !== '') {
          params.set(key, String(value));
        }
      }
      const query = params.toString();
      history.replaceState({}, '', query ? `/logs?${query}` : '/logs');
    },

    normalizePagination(): void {
      this.filters.limit = Number(this.filters.limit) || 100;
      this.filters.offset = Math.max(0, Number(this.filters.offset) || 0);
    },

    async load(): Promise<void> {
      this.loading = true;
      this.error = '';
      this.normalizePagination();
      try {
        this.entries = await api.listLogs(this.buildFilter());
        this.hasNextPage = this.entries.length === this.filters.limit;
        this.syncQuery();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load recent logs';
      } finally {
        this.loading = false;
      }
    },

    async applyFilters(): Promise<void> {
      this.filters.offset = 0;
      await this.load();
    },

    async clearFilters(): Promise<void> {
      this.filters.level = '';
      this.filters.source = '';
      this.filters.query = '';
      this.filters.start_time = '';
      this.filters.end_time = '';
      this.filters.limit = 100;
      this.filters.offset = 0;
      await this.load();
    },

    async previousPage(): Promise<void> {
      this.normalizePagination();
      this.filters.offset = Math.max(0, this.filters.offset - this.filters.limit);
      await this.load();
    },

    async nextPage(): Promise<void> {
      this.normalizePagination();
      this.filters.offset += this.filters.limit;
      await this.load();
    },

    async openDetail(entry: LogEntry): Promise<void> {
      this.error = '';
      try {
        this.selectedEntry = await api.getLogEntry(entry.id);
        this.modalType = 'detail';
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load log entry';
      }
    },

    closeModal(): void {
      this.modalType = '';
      this.selectedEntry = null;
    },

    getSelectedEntryValue(field: keyof LogEntry): string {
      if (!this.selectedEntry) {
        return '-';
      }
      const value = this.selectedEntry[field];
      if (value === undefined || value === null || value === '') {
        return '-';
      }
      return String(value);
    },

    getSelectedEntryTimestamp(): string {
      return this.selectedEntry ? this.formatDate(this.selectedEntry.timestamp) : '-';
    },

    getSelectedEntryMessage(): string {
      if (!this.selectedEntry || !this.selectedEntry.message) {
        return '';
      }
      return this.selectedEntry.message;
    },

    getExportURL(format: 'json' | 'csv'): string {
      return api.getLogsExportURL(this.buildFilter(), format);
    },

    getLevelClass(level: string): string {
      switch (level.toLowerCase()) {
        case 'error':
        case 'fatal':
          return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400';
        case 'warn':
          return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400';
        case 'info':
          return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400';
        default:
          return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400';
      }
    },

    formatDate(value: string): string {
      return new Date(value).toLocaleString();
    },

    formatFields(entry: LogEntry | null): string {
      if (!entry?.fields) {
        return '';
      }
      return JSON.stringify(entry.fields, null, 2);
    },

    getPageStart(): number {
      return this.filters.offset + 1;
    },

    getPageEnd(): number {
      return this.filters.offset + this.entries.length;
    },

    get currentPage(): number {
      return Math.floor(this.filters.offset / this.filters.limit) + 1;
    },

    hasEntries(): boolean {
      return this.entries.length > 0;
    },
  };
}
