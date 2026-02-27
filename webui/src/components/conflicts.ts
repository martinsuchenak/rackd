// Conflict Components for Rackd Web UI

import type { Conflict } from '../core/types';
import { api, RackdAPIError } from '../core/api';
import { formatDate } from '../core/utils';

interface ConflictListData {
  conflicts: Conflict[];
  summary: {
    duplicate_ips: number;
    overlapping_subnets: number;
  };
  filter: string;
  loading: boolean;
  detecting: boolean;
  error: string;
  init(): Promise<void>;
  loadConflicts(): Promise<void>;
  loadSummary(): Promise<void>;
  detectAll(): Promise<void>;
  applyFilter(): void;
  openFilter(type: string): void;
  clearFilter(): void;
  formatDate(dateString: string): string;
  get filteredConflicts(): Conflict[];
  get activeConflictCount(): number;
}

export function conflictList(): ConflictListData {
  return {
    conflicts: [],
    summary: { duplicate_ips: 0, overlapping_subnets: 0 },
    filter: '',
    loading: false,
    detecting: false,
    error: '',

    async init(): Promise<void> {
      await new Promise((resolve) => setTimeout(resolve, 0));
      await Promise.all([this.loadConflicts(), this.loadSummary()]);
    },

    async loadConflicts(): Promise<void> {
      this.loading = true;
      this.error = '';
      try {
        let conflicts = await api.listConflicts();
        if (this.filter) {
          conflicts = conflicts.filter((c: Conflict) => c.type === this.filter);
        }
        this.conflicts = conflicts;
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load conflicts';
      } finally {
        this.loading = false;
      }
    },

    async loadSummary(): Promise<void> {
      try {
        const result = await api.getConflictSummary();
        if (result) {
          this.summary = {
            duplicate_ips: result.duplicate_ips || 0,
            overlapping_subnets: result.overlapping_subnets || 0,
          };
        }
      } catch {
        // Non-critical, keep default values
      }
    },

    async detectAll(): Promise<void> {
      this.detecting = true;
      this.error = '';
      try {
        const results = await api.detectConflicts('');
        this.conflicts = results.conflicts || [];
        await this.loadSummary();
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to detect conflicts';
      } finally {
        this.detecting = false;
      }
    },

    applyFilter(): void {
      this.filter = (document.getElementById('conflict-filter') as HTMLSelectElement)?.value || '';
    },

    openFilter(type: string): void {
      this.filter = type;
      const select = document.getElementById('conflict-filter') as HTMLSelectElement;
      if (select) select.value = type;
      this.loadConflicts();
    },

    clearFilter(): void {
      this.filter = '';
      const select = document.getElementById('conflict-filter') as HTMLSelectElement;
      if (select) select.value = '';
      this.loadConflicts();
    },

    formatDate(dateString: string): string {
      return formatDate(dateString);
    },

    get filteredConflicts(): Conflict[] {
      if (this.filter) {
        return this.conflicts.filter((c: Conflict) => c.type === this.filter);
      }
      return this.conflicts;
    },

    get activeConflictCount(): number {
      return (this.summary.duplicate_ips || 0) + (this.summary.overlapping_subnets || 0);
    },
  };
}
