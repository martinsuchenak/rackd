// User Menu Component for Rackd Web UI

import type { User, UpdateUserRequest, Permission, Role } from '../core/types';
import { api, RackdAPIError } from '../core/api';

export function userMenu() {
  return {
    open: false,
    showEditModal: false,
    showPasswordModal: false,
    showPermissionsModal: false,
    saving: false,
    currentUser: null as User | null,
    validationErrors: {} as Record<string, string>,
    // Edit form - flat properties for CSP compatibility
    editUsername: '',
    editEmail: '',
    editFullName: '',
    // Password form - flat properties for CSP compatibility
    passwordOldPassword: '',
    passwordNewPassword: '',
    passwordConfirmPassword: '',
    error: '',

    get username(): string {
      return window.rackdConfig?.user?.username || '';
    },

    get fullName(): string {
      return window.rackdConfig?.user?.full_name || '';
    },

    get displayName(): string {
      return this.fullName || this.username;
    },

    get initial(): string {
      return this.displayName.charAt(0).toUpperCase();
    },

    get isAdmin(): boolean {
      return window.rackdConfig?.user?.is_admin || false;
    },

    get userRoles(): Role[] {
      return (window.rackdConfig?.user?.roles || []) as any;
    },

    get roleNames(): string {
      return this.userRoles.map((r: Role) => r.name).join(', ');
    },

    get userPermissions(): Permission[] {
      return (window.rackdConfig?.user?.permissions || []) as any;
    },

    get groupedPermissions(): Record<string, Permission[]> {
      const groups: Record<string, Permission[]> = {};
      for (const perm of this.userPermissions) {
        if (!groups[perm.resource]) {
          groups[perm.resource] = [];
        }
        groups[perm.resource].push(perm);
      }
      return groups;
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
        this.editUsername = this.currentUser.username || '';
        this.editEmail = this.currentUser.email || '';
        this.editFullName = this.currentUser.full_name || '';
        this.showEditModal = true;
      } catch {
        this.error = 'Failed to load profile';
      }
    },

    closeEditModal(): void {
      this.showEditModal = false;
      this.validationErrors = {};
    },

    syncUserConfig(user: User): void {
      if (!window.rackdConfig?.user) {
        return;
      }
      window.rackdConfig.user = {
        ...window.rackdConfig.user,
        username: user.username,
        email: user.email,
        full_name: user.full_name,
      };
    },

    async saveProfile(): Promise<void> {
      this.validationErrors = {};

      if (!this.editUsername.trim()) {
        this.validationErrors.username = 'Username is required';
      }
      if (this.editEmail && !this.editEmail.includes('@')) {
        this.validationErrors.email = 'Invalid email format';
      }
      if (Object.keys(this.validationErrors).length > 0) return;

      this.saving = true;
      try {
        const updates: UpdateUserRequest = {};
        updates.username = this.editUsername.trim();
        if (this.editEmail) updates.email = this.editEmail;
        if (this.editFullName) updates.full_name = this.editFullName;
        const user = await api.updateUser(this.currentUser!.id, updates);
        this.currentUser = user;
        this.syncUserConfig(user);
        this.closeEditModal();
      } catch (err) {
        if (err instanceof RackdAPIError) {
          if (Array.isArray(err.details)) {
            for (const detail of err.details) {
              if (detail.field && detail.message) {
                this.validationErrors[detail.field] = detail.message;
              }
            }
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
      this.passwordOldPassword = '';
      this.passwordNewPassword = '';
      this.passwordConfirmPassword = '';
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
      this.passwordOldPassword = '';
      this.passwordNewPassword = '';
      this.passwordConfirmPassword = '';
    },

    async doChangePassword(): Promise<void> {
      this.validationErrors = {};

      if (!this.passwordOldPassword) {
        this.validationErrors.old_password = 'Current password is required';
      }
      if (!this.passwordNewPassword) {
        this.validationErrors.new_password = 'New password is required';
      } else if (this.passwordNewPassword.length < 8) {
        this.validationErrors.new_password = 'Password must be at least 8 characters';
      }
      if (this.passwordNewPassword !== this.passwordConfirmPassword) {
        this.validationErrors.confirm_password = 'Passwords do not match';
      }

      if (Object.keys(this.validationErrors).length > 0) return;

      this.saving = true;
      try {
        await api.changePassword(this.currentUser!.id, {
          old_password: this.passwordOldPassword,
          new_password: this.passwordNewPassword,
        });
        this.closePasswordModal();
      } catch (err) {
        if (err instanceof RackdAPIError) {
          // Handle field-specific validation errors from server (array format)
          if (Array.isArray(err.details)) {
            for (const detail of err.details) {
              if (detail.field && detail.message) {
                this.validationErrors[detail.field] = detail.message;
              } else if (detail.message) {
                // No field specified - show as general error
                this.error = detail.message;
              }
            }
          } else if (err.code === 'INVALID_PASSWORD') {
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

    openPermissionsModal(): void {
      this.open = false;
      this.showPermissionsModal = true;
    },

    closePermissionsModal(): void {
      this.showPermissionsModal = false;
    },

    formatPermissionName(perm: Permission): string {
      return `${perm.resource}:${perm.action}`;
    },
  };
}
