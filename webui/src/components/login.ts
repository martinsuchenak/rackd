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

/**
 * Where to go after signing in: the redirect parameter if it is on this
 * origin, otherwise "/". It returns an absolute URL: a path such as
 * "//evil.example" (from "/.//evil.example") would be read as another host.
 */
export function loginRedirectTarget(redirect: string | null, origin: string): string {
  if (!redirect) return '/';
  try {
    const url = new URL(redirect, origin);
    return url.origin === origin ? url.href : '/';
  } catch {
    return '/';
  }
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
        window.location.href = loginRedirectTarget(params.get('redirect'), window.location.origin);
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
