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

      this.loading = true;
      this.error = '';

      try {
        const request: LoginRequest = {
          username: this.username.trim(),
          password: this.password,
        };

        await api.login(request.username, request.password);

        // Cookie is set by the server (httpOnly) — just redirect
        const params = new URLSearchParams(window.location.search);
        const redirect = params.get('redirect');
        // Security: only navigate to same-origin paths. Resolving via URL and
        // comparing origins defeats encoded-host tricks like /\evil.com and
        // //evil.com (browsers normalize backslashes to slashes).
        if (redirect) {
          try {
            const url = new URL(redirect, window.location.origin);
            if (url.origin === window.location.origin) {
              window.location.href = url.pathname + url.search + url.hash;
              return;
            }
          } catch {
            // fall through to default
          }
        }
        window.location.href = '/';
      } catch (err) {
        this.showError(err instanceof Error && err.message ? err.message : 'Login failed');
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
