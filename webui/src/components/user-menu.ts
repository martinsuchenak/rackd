// User Menu Component for Rackd Web UI

import type { User, UpdateUserRequest } from '../core/types';
import { api, RackdAPIError } from '../core/api';

export function userMenu() {
  return {
    open: false,
    showEditModal: false,
    showPasswordModal: false,
    saving: false,
    currentUser: null as User | null,
    validationErrors: {} as Record<string, string>,
    editForm: {
      email: '',
      full_name: '',
    },
    passwordForm: {
      old_password: '',
      new_password: '',
      confirm_password: '',
    },
    error: '',

    get username(): string {
      return window.rackdConfig?.user?.username || '';
    },

    get initial(): string {
      return this.username.charAt(0).toUpperCase();
    },

    get isAdmin(): boolean {
      return window.rackdConfig?.user?.roles?.includes('admin') || false;
    },

    toggle(): void {
      this.open = !this.open;
    },

    close(): void {
      this.open = false;
    },

    async openEditProfile(): Promise<void> {
      this.open = false;
      this.validationErrors = {};
      this.error = '';
      try {
        this.currentUser = await api.getCurrentUser();
        this.editForm = {
          email: this.currentUser.email || '',
          full_name: this.currentUser.full_name || '',
        };
        this.showEditModal = true;
      } catch {
        this.error = 'Failed to load profile';
      }
    },

    closeEditModal(): void {
      this.showEditModal = false;
      this.validationErrors = {};
    },

    async saveProfile(): Promise<void> {
      this.validationErrors = {};

      if (this.editForm.email && !this.editForm.email.includes('@')) {
        this.validationErrors.email = 'Invalid email format';
        return;
      }

      this.saving = true;
      try {
        const updates: UpdateUserRequest = {};
        if (this.editForm.email) updates.email = this.editForm.email;
        if (this.editForm.full_name) updates.full_name = this.editForm.full_name;
        await api.updateUser(this.currentUser!.id, updates);
        this.closeEditModal();
      } catch (err) {
        if (err instanceof RackdAPIError) {
          if (err.code === 'EMAIL_EXISTS') {
            this.validationErrors.email = err.message;
          } else {
            this.error = err.message;
          }
        } else {
          this.error = 'Failed to update profile';
        }
      } finally {
        this.saving = false;
      }
    },

    async openChangePassword(): Promise<void> {
      this.open = false;
      this.validationErrors = {};
      this.error = '';
      this.passwordForm = { old_password: '', new_password: '', confirm_password: '' };
      try {
        this.currentUser = await api.getCurrentUser();
        this.showPasswordModal = true;
      } catch {
        this.error = 'Failed to load profile';
      }
    },

    closePasswordModal(): void {
      this.showPasswordModal = false;
      this.validationErrors = {};
      this.passwordForm = { old_password: '', new_password: '', confirm_password: '' };
    },

    async doChangePassword(): Promise<void> {
      this.validationErrors = {};

      if (!this.passwordForm.old_password) {
        this.validationErrors.old_password = 'Current password is required';
      }
      if (!this.passwordForm.new_password) {
        this.validationErrors.new_password = 'New password is required';
      } else if (this.passwordForm.new_password.length < 8) {
        this.validationErrors.new_password = 'Password must be at least 8 characters';
      }
      if (this.passwordForm.new_password !== this.passwordForm.confirm_password) {
        this.validationErrors.confirm_password = 'Passwords do not match';
      }

      if (Object.keys(this.validationErrors).length > 0) return;

      this.saving = true;
      try {
        await api.changePassword(this.currentUser!.id, {
          old_password: this.passwordForm.old_password,
          new_password: this.passwordForm.new_password,
        });
        this.closePasswordModal();
      } catch (err) {
        if (err instanceof RackdAPIError) {
          if (err.code === 'INVALID_PASSWORD') {
            this.validationErrors.old_password = err.message;
          } else {
            this.error = err.message;
          }
        } else {
          this.error = 'Failed to change password';
        }
      } finally {
        this.saving = false;
      }
    },

    async logout(): Promise<void> {
      try {
        await api.logout();
      } catch {
        // Continue with redirect even if server call fails
      }
      window.location.href = '/login';
    },
  };
}
