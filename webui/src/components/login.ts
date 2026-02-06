// Login Component for Rackd Web UI

import type { LoginRequest } from '../core/types';
import { api } from '../core/api';

interface LoginData {
  username: string;
  password: string;
  loading: boolean;
  error: string;
  init(): void;
  submit(): Promise<void>;
  showError(message: string): void;
}

export function login() {
  return {
    username: '',
    password: '',
    loading: false,
    error: '',

    async init(): Promise<void> {
      try {
        const config = await api.getConfig();
        if (config.user) {
          window.location.href = '/';
        }
      } catch {
      }
    },

    async submit(): Promise<void> {
      if (!this.username || !this.password) {
        this.showError('Username and password are required');
        return;
      }

      if (this.password.length < 8) {
        this.showError('Password must be at least 8 characters');
        return;
      }

      this.loading = true;
      this.error = '';

      try {
        const request: LoginRequest = {
          username: this.username.trim(),
          password: this.password,
        };

        const response = await fetch('/api/auth/login', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'same-origin',
          body: JSON.stringify(request),
        });

        const data = await response.json();

        if (!response.ok) {
          this.showError(data.message || 'Login failed');
          return;
        }

        // Cookie is set by the server (httpOnly) — just redirect
        window.location.href = '/';
      } catch (err) {
        this.showError('Network error. Please try again.');
      } finally {
        this.loading = false;
      }
    },

    showError(message: string): void {
      this.error = message;
      setTimeout(() => {
        this.error = '';
      }, 5000);
    },
  };
}
