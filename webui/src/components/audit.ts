import type { AuditFilter, AuditLog } from '../core/types';
import { api, RackdAPIError } from '../core/api';

type ModalType = '' | 'detail';

export function auditLogsPage() {
  return {
    logs: [] as AuditLog[],
    loading: true,
    error: '',
    modalType: '' as ModalType,
    selectedLog: null as AuditLog | null,
    hasNextPage: false,
    filters: {
      resource: '',
      action: '',
      user_id: '',
      source: '',
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
      this.filters.resource = params.get('resource') || '';
      this.filters.action = params.get('action') || '';
      this.filters.user_id = params.get('user_id') || '';
      this.filters.source = params.get('source') || '';
      this.filters.start_time = this.toInputDateTime(params.get('start_time'));
      this.filters.end_time = this.toInputDateTime(params.get('end_time'));
      this.filters.limit = Number(params.get('limit') || 100);
      this.filters.offset = Number(params.get('offset') || 0);
    },

    buildFilter(): AuditFilter {
      return {
        resource: this.filters.resource || undefined,
        action: this.filters.action || undefined,
        user_id: this.filters.user_id || undefined,
        source: this.filters.source || undefined,
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
      history.replaceState({}, '', query ? `/audit?${query}` : '/audit');
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
        this.logs = await api.listAuditLogs(this.buildFilter());
        this.hasNextPage = this.logs.length === this.filters.limit;
        this.syncQuery();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load audit logs';
      } finally {
        this.loading = false;
      }
    },

    async applyFilters(): Promise<void> {
      this.filters.offset = 0;
      await this.load();
    },

    async clearFilters(): Promise<void> {
      this.filters.resource = '';
      this.filters.action = '';
      this.filters.user_id = '';
      this.filters.source = '';
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

    async openDetail(log: AuditLog): Promise<void> {
      this.error = '';
      try {
        this.selectedLog = await api.getAuditLog(log.id);
        this.modalType = 'detail';
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load audit log details';
      }
    },

    closeModal(): void {
      this.modalType = '';
      this.selectedLog = null;
    },

    getSelectedLogValue(field: keyof AuditLog): string {
      if (!this.selectedLog) {
        return '-';
      }
      const value = this.selectedLog[field];
      if (value === undefined || value === null || value === '') {
        return '-';
      }
      return String(value);
    },

    getSelectedLogTimestamp(): string {
      return this.selectedLog ? this.formatDate(this.selectedLog.timestamp) : '-';
    },

    hasSelectedLogError(): boolean {
      return Boolean(this.selectedLog && this.selectedLog.error);
    },

    getSelectedLogError(): string {
      if (!this.selectedLog || !this.selectedLog.error) {
        return '';
      }
      return this.selectedLog.error;
    },

    getPrettyChanges(): string {
      if (!this.selectedLog?.changes) {
        return '';
      }
      try {
        return JSON.stringify(JSON.parse(this.selectedLog.changes), null, 2);
      } catch {
        return this.selectedLog.changes;
      }
    },

    getExportURL(format: 'json' | 'csv'): string {
      return api.getAuditExportURL(this.buildFilter(), format);
    },

    getStatusClass(status: string): string {
      return status === 'success'
        ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
        : 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400';
    },

    formatDate(value: string): string {
      return new Date(value).toLocaleString();
    },

    getPageStart(): number {
      return this.filters.offset + 1;
    },

    getPageEnd(): number {
      return this.filters.offset + this.logs.length;
    },

    get currentPage(): number {
      return Math.floor(this.filters.offset / this.filters.limit) + 1;
    },

    hasLogs(): boolean {
      return this.logs.length > 0;
    },
  };
}
