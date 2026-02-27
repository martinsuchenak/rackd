// Dashboard Components for Rackd Web UI

import { api, RackdAPIError } from '../core/api';
import type { DashboardStats } from '../core/types';

export function dashboardComponent() {
  return {
    stats: null as DashboardStats | null,
    loading: true,
    error: '',
    refreshInterval: 60, // 60 seconds default
    autoRefresh: true,
    refreshTimer: null as number | null,

    async init(): Promise<void> {
      await this.loadStats();
      if (this.autoRefresh) {
        this.startAutoRefresh();
      }
    },

    async loadStats(): Promise<void> {
      this.loading = true;
      this.error = '';
      try {
        this.stats = await api.getDashboardStats(7, 10);
      } catch (e) {
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load dashboard';
      } finally {
        this.loading = false;
      }
    },

    startAutoRefresh(): void {
      this.stopAutoRefresh();
      this.autoRefresh = true;
      this.refreshTimer = window.setInterval(() => {
        this.loadStats();
      }, this.refreshInterval * 1000);
    },

    stopAutoRefresh(): void {
      if (this.refreshTimer) {
        window.clearInterval(this.refreshTimer);
        this.refreshTimer = null;
      }
      this.autoRefresh = false;
    },

    toggleAutoRefresh(): void {
      if (this.autoRefresh) {
        this.stopAutoRefresh();
      } else {
        this.startAutoRefresh();
      }
    },

    formatUtilization(value: number): string {
      return value.toFixed(1) + '%';
    },

    getUtilizationColor(value: number): string {
      if (value >= 90) return 'text-red-600 dark:text-red-400';
      if (value >= 70) return 'text-yellow-600 dark:text-yellow-400';
      return 'text-green-600 dark:text-green-400';
    },

    getUtilizationBgColor(value: number): string {
      if (value >= 90) return 'bg-red-100 dark:bg-red-900/30';
      if (value >= 70) return 'bg-yellow-100 dark:bg-yellow-900/30';
      return 'bg-green-100 dark:bg-green-900/30';
    },

    // Cleanup on component destroy
    destroy(): void {
      this.stopAutoRefresh();
    }
  };
}

// Export for use in app.ts
export { formatDate } from '../core/utils';
