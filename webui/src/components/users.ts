// Users Component for Rackd Web UI

import type { User, UserFilter, UpdateUserRequest, Role } from '../core/types';
import { api, RackdAPIError } from '../core/api';
import { formatDate } from '../core/utils';

interface UsersListData {
  users: User[];
  loading: boolean;
  error: string;
  // Filter - flat properties for CSP compatibility
  filterUsername: string;
  filterEmail: string;
  page: number;
  pageSize: number;
  totalPages: number;
  pagedUsers: User[];
  showCreateModal: boolean;
  showEditModal: boolean;
  showDeleteModal: boolean;
  showPasswordModal: boolean;
  showResetPasswordModal: boolean;
  selectedUser: User | null;
  saving: boolean;
  deleting: boolean;
  validationErrors: Record<string, string>;
  // Create form - flat properties for CSP compatibility
  createUsername: string;
  createPassword: string;
  createEmail: string;
  createFullName: string;
  // Edit form - flat properties for CSP compatibility
  editUsername: string;
  editEmail: string;
  editFullName: string;
  editIsActive: boolean;
  // Password form - flat properties for CSP compatibility
  passwordOldPassword: string;
  passwordNewPassword: string;
  passwordConfirmPassword: string;
  // Reset password form - flat properties for CSP compatibility
  resetPasswordNew: string;
  resetPasswordConfirm: string;
  currentUser: User | null;
  availableRoles: Role[];
  userRolesCache: Map<string, Role[]>;
  showRoleManager: boolean;
  roleSaving: boolean;
  init(): Promise<void>;
  loadUsers(): Promise<void>;
  loadAvailableRoles(): Promise<void>;
  loadUserRoles(userId: string): Promise<Role[]>;
  applyFilter(): void;
  clearFilter(): void;
  goToPage(p: number): void;
  openCreateModal(): void;
  closeCreateModal(): void;
  openEditModal(user: User): void;
  closeEditModal(): void;
  openDeleteModal(user: User): void;
  closeDeleteModal(): void;
  openPasswordModal(user: User): void;
  closePasswordModal(): void;
  openResetPasswordModal(user: User): void;
  closeResetPasswordModal(): void;
  openRoleManager(user: User): void;
  closeRoleManager(): void;
  doCreateUser(): Promise<void>;
  doUpdateUser(): Promise<void>;
  doDeleteUser(): Promise<void>;
  doChangePassword(): Promise<void>;
  doResetPassword(): Promise<void>;
  grantRole(role: Role): Promise<void>;
  revokeRole(role: Role): Promise<void>;
  logout(): void;
  getSelectedUsername(): string;
  canResetPassword(user: User): boolean;
  canDeleteUser(user: User): boolean;
  hasUserRoles(user: User): boolean;
  getUserRoles(user: User): Role[];
  hasSelectedUserRoles(): boolean;
}

export function usersList() {
  return {
    users: [] as User[],
    loading: true,
    error: '',
    // Filter - flat properties for CSP compatibility
    filterUsername: '',
    filterEmail: '',
    page: 1,
    pageSize: 10,
    showCreateModal: false,
    showEditModal: false,
    showDeleteModal: false,
    showPasswordModal: false,
    showResetPasswordModal: false,
    selectedUser: null as User | null,
    saving: false,
    deleting: false,
    validationErrors: {} as Record<string, string>,
    // Create form - flat properties for CSP compatibility
    createUsername: '',
    createPassword: '',
    createEmail: '',
    createFullName: '',
    // Edit form - flat properties for CSP compatibility
    editUsername: '',
    editEmail: '',
    editFullName: '',
    editIsActive: true,
    // Password form - flat properties for CSP compatibility
    passwordOldPassword: '',
    passwordNewPassword: '',
    passwordConfirmPassword: '',
    // Reset password form - flat properties for CSP compatibility
    resetPasswordNew: '',
    resetPasswordConfirm: '',
    currentUser: null as User | null,
    availableRoles: [] as Role[],
    userRolesCache: new Map<string, Role[]>(),
    showRoleManager: false,
    roleSaving: false,

    get totalPages(): number {
      return Math.ceil(this.users.length / this.pageSize) || 1;
    },

    get pagedUsers(): User[] {
      const start = (this.page - 1) * this.pageSize;
      return this.users.slice(start, start + this.pageSize);
    },

    get deleteModalTitle(): string {
      return 'Delete User';
    },

    get deleteModalName(): string {
      return this.selectedUser?.username || '';
    },

    async init(): Promise<void> {
      await this.loadUsers();
      await this.loadCurrentUser();
    },

    async loadUsers(): Promise<void> {
      this.loading = true;
      this.error = '';

      const filter: UserFilter = {};
      if (this.filterUsername) filter.username = this.filterUsername;
      if (this.filterEmail) filter.email = this.filterEmail;

      try {
        this.users = await api.listUsers(filter);
      } catch (err) {
        if (err instanceof RackdAPIError) {
          this.error = err.message;
        } else {
          this.error = 'Failed to load users';
        }
      } finally {
        this.loading = false;
      }
    },

    async loadCurrentUser(): Promise<void> {
      try {
        this.currentUser = await api.getCurrentUser();
      } catch {
        this.currentUser = null;
      }
      await this.loadAvailableRoles();
    },

    async loadAvailableRoles(): Promise<void> {
      try {
        this.availableRoles = await api.listRoles({});
      } catch (err) {
        console.error('Failed to load roles:', err);
        if (err instanceof RackdAPIError) {
          this.error = 'Failed to load roles: ' + err.message;
        }
        this.availableRoles = [];
      }
    },

    async loadUserRoles(userId: string): Promise<Role[]> {
      if (this.userRolesCache.has(userId)) {
        return this.userRolesCache.get(userId)!;
      }

      try {
        const roles = await api.getUserRoles(userId);
        this.userRolesCache.set(userId, roles);
        return roles;
      } catch {
        return [];
      }
    },

    applyFilter(): void {

      this.page = 1;
      this.loadUsers();
    },

    clearFilter(): void {
      this.filterUsername = '';
      this.filterEmail = '';
      this.page = 1;
      this.loadUsers();
    },

    goToPage(p: number): void {
      this.page = p;
    },

    openCreateModal(): void {
      this.showCreateModal = true;
      this.validationErrors = {};
      this.createUsername = '';
      this.createPassword = '';
      this.createEmail = '';
      this.createFullName = '';
    },

    closeCreateModal(): void {
      this.showCreateModal = false;
      this.validationErrors = {};
      this.createUsername = '';
      this.createPassword = '';
      this.createEmail = '';
      this.createFullName = '';
    },

    openEditModal(user: User): void {
      this.showEditModal = true;
      this.selectedUser = user;
      this.validationErrors = {};
      // Pre-populate edit form with user's current values
      this.editUsername = user.username || '';
      this.editEmail = user.email || '';
      this.editFullName = user.full_name || '';
      this.editIsActive = user.is_active !== false;
    },

    closeEditModal(): void {
      this.showEditModal = false;
      this.selectedUser = null;
      this.validationErrors = {};
    },

    openDeleteModal(user: User): void {
      this.showDeleteModal = true;
      this.selectedUser = user;
    },

    closeDeleteModal(): void {
      this.showDeleteModal = false;
      this.selectedUser = null;
    },

    openPasswordModal(user: User): void {
      this.showPasswordModal = true;
      this.selectedUser = user;
      this.validationErrors = {};
      this.passwordOldPassword = '';
      this.passwordNewPassword = '';
      this.passwordConfirmPassword = '';
    },

    closePasswordModal(): void {
      this.showPasswordModal = false;
      this.selectedUser = null;
      this.validationErrors = {};
      this.passwordOldPassword = '';
      this.passwordNewPassword = '';
      this.passwordConfirmPassword = '';
    },

    openResetPasswordModal(user: User): void {
      this.showResetPasswordModal = true;
      this.selectedUser = user;
      this.validationErrors = {};
      this.resetPasswordNew = '';
      this.resetPasswordConfirm = '';
    },

    closeResetPasswordModal(): void {
      this.showResetPasswordModal = false;
      this.selectedUser = null;
      this.validationErrors = {};
      this.resetPasswordNew = '';
      this.resetPasswordConfirm = '';
    },

    hasRole(role: Role): boolean {
      if (!this.selectedUser?.roles) {
        return false;
      }
      return this.selectedUser.roles.some((r: Role) => r.id === role.id);
    },

    async openRoleManager(user: User): Promise<void> {
      this.selectedUser = user;
      // Load roles and assign to selectedUser for hasRole() to work
      this.selectedUser.roles = await this.loadUserRoles(user.id);
      this.showRoleManager = true;
    },

    closeRoleManager(): void {
      this.showRoleManager = false;
      this.selectedUser = null;
    },

    async grantRole(role: Role): Promise<void> {
      if (!this.selectedUser) {
        return;
      }

      this.roleSaving = true;

      try {
        await api.grantRole(this.selectedUser.id, role.id);
        this.userRolesCache.delete(this.selectedUser.id);
        this.selectedUser.roles = await this.loadUserRoles(this.selectedUser.id);
        await this.loadUsers();

        // Refresh permissions if user is modifying their own roles
        if (this.currentUser && this.currentUser.id === this.selectedUser.id) {
          window.dispatchEvent(new Event('permissions:refresh'));
        }
      } catch (err) {
        if (err instanceof RackdAPIError) {
          this.error = err.message;
        } else {
          this.error = 'Failed to grant role';
        }
      } finally {
        this.roleSaving = false;
      }
    },

    async revokeRole(role: Role): Promise<void> {
      if (!this.selectedUser) {
        return;
      }

      this.roleSaving = true;

      try {
        await api.revokeRole(this.selectedUser.id, role.id);
        this.userRolesCache.delete(this.selectedUser.id);
        this.selectedUser.roles = await this.loadUserRoles(this.selectedUser.id);
        await this.loadUsers();

        // Refresh permissions if user is modifying their own roles
        if (this.currentUser && this.currentUser.id === this.selectedUser.id) {
          window.dispatchEvent(new Event('permissions:refresh'));
        }
      } catch (err) {
        if (err instanceof RackdAPIError) {
          this.error = err.message;
        } else {
          this.error = 'Failed to revoke role';
        }
      } finally {
        this.roleSaving = false;
      }
    },

    async doCreateUser(): Promise<void> {
      this.validationErrors = {};

      if (!this.createUsername) {
        this.validationErrors.username = 'Username is required';
      }

      if (!this.createPassword) {
        this.validationErrors.password = 'Password is required';
      } else if (this.createPassword.length < 8) {
        this.validationErrors.password = 'Password must be at least 8 characters';
      }

      if (!this.createEmail) {
        this.validationErrors.email = 'Email is required';
      } else if (!this.createEmail.includes('@')) {
        this.validationErrors.email = 'Invalid email format';
      }

      if (Object.keys(this.validationErrors).length > 0) {
        return;
      }

      this.saving = true;

      try {
        await api.createUser({
          username: this.createUsername,
          password: this.createPassword,
          email: this.createEmail,
          full_name: this.createFullName,
        });
        this.closeCreateModal();
        await this.loadUsers();
      } catch (err) {
        if (err instanceof RackdAPIError) {
          if (err.code === 'USERNAME_EXISTS') {
            this.validationErrors.username = err.message;
          } else if (err.code === 'EMAIL_EXISTS') {
            this.validationErrors.email = err.message;
          } else {
            this.error = err.message;
          }
        } else {
          this.error = 'Failed to create user';
        }
      } finally {
        this.saving = false;
      }
    },

    async doUpdateUser(): Promise<void> {
      if (!this.selectedUser) {
        return;
      }

      this.validationErrors = {};
      const updates: UpdateUserRequest = {};

      if (!this.editUsername.trim()) {
        this.validationErrors.username = 'Username is required';
      } else {
        updates.username = this.editUsername.trim();
      }

      if (this.editEmail) {
        if (!this.editEmail.includes('@')) {
          this.validationErrors.email = 'Invalid email format';
        } else {
          updates.email = this.editEmail;
        }
      }

      if (this.editFullName) {
        updates.full_name = this.editFullName;
      }

      updates.is_active = this.editIsActive;

      if (Object.keys(this.validationErrors).length > 0) {
        return;
      }

      this.saving = true;

      try {
        const updatedUser = await api.updateUser(this.selectedUser.id, updates);
        if (this.currentUser && this.currentUser.id === updatedUser.id) {
          this.currentUser = updatedUser;
        }
        if (this.currentUser && this.currentUser.id === updatedUser.id && window.rackdConfig?.user) {
          window.rackdConfig.user = {
            ...window.rackdConfig.user,
            username: updatedUser.username,
            email: updatedUser.email,
            full_name: updatedUser.full_name,
          };
        }
        this.closeEditModal();
        await this.loadUsers();
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
          this.error = 'Failed to update user';
        }
      } finally {
        this.saving = false;
      }
    },

    async doDeleteUser(): Promise<void> {
      if (!this.selectedUser) {
        return;
      }

      if (this.currentUser && this.currentUser.id === this.selectedUser.id) {
        this.error = 'Cannot delete your own account';
        return;
      }

      this.deleting = true;

      try {
        await api.deleteUser(this.selectedUser.id);
        this.closeDeleteModal();
        await this.loadUsers();
      } catch (err) {
        if (err instanceof RackdAPIError) {
          this.error = err.message;
        } else {
          this.error = 'Failed to delete user';
        }
      } finally {
        this.deleting = false;
      }
    },

    async doChangePassword(): Promise<void> {
      if (!this.selectedUser) {
        return;
      }

      this.validationErrors = {};

      if (!this.passwordOldPassword) {
        this.validationErrors.old_password = 'Old password is required';
      }

      if (!this.passwordNewPassword) {
        this.validationErrors.new_password = 'New password is required';
      } else if (this.passwordNewPassword.length < 8) {
        this.validationErrors.new_password = 'Password must be at least 8 characters';
      }

      if (this.passwordNewPassword !== this.passwordConfirmPassword) {
        this.validationErrors.confirm_password = 'Passwords do not match';
      }

      if (Object.keys(this.validationErrors).length > 0) {
        return;
      }

      this.saving = true;

      try {
        await api.changePassword(this.selectedUser.id, {
          old_password: this.passwordOldPassword,
          new_password: this.passwordNewPassword,
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

    async doResetPassword(): Promise<void> {
      if (!this.selectedUser) {
        return;
      }

      this.validationErrors = {};

      if (!this.resetPasswordNew) {
        this.validationErrors.new_password = 'New password is required';
      } else if (this.resetPasswordNew.length < 8) {
        this.validationErrors.new_password = 'Password must be at least 8 characters';
      }

      if (this.resetPasswordNew !== this.resetPasswordConfirm) {
        this.validationErrors.confirm_password = 'Passwords do not match';
      }

      if (Object.keys(this.validationErrors).length > 0) {
        return;
      }

      this.saving = true;

      try {
        await api.resetPassword(this.selectedUser.id, this.resetPasswordNew);
        this.closeResetPasswordModal();
      } catch (err) {
        if (err instanceof RackdAPIError) {
          this.error = err.message;
        } else {
          this.error = 'Failed to reset password';
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

    formatDate: (dateString: string) => {
      return formatDate(dateString);
    },

    getUserInitial(username: string): string {
      return username ? username.charAt(0).toUpperCase() : '';
    },

    getSelectedUsername(): string {
      return this.selectedUser?.username || '';
    },

    canResetPassword(user: User): boolean {
      return !!(this.currentUser && this.currentUser.id !== user.id);
    },

    canDeleteUser(user: User): boolean {
      return !!(this.currentUser && this.currentUser.id !== user.id);
    },

    hasUserRoles(user: User | null): boolean {
      return !!(user && user.roles && user.roles.length > 0);
    },

    getUserRoles(user: User | null): Role[] {
      return user?.roles || [];
    },

    hasSelectedUserRoles(): boolean {
      return (this.selectedUser?.roles || []).length > 0;
    },

    canUpdateRoles(): boolean {
      // @ts-ignore - Alpine store access
      return Alpine.store('permissions').canUpdate('roles');
    },

    canListRoles(): boolean {
      // @ts-ignore - Alpine store access
      return Alpine.store('permissions').canList('roles');
    },

    getLastLoginLabel(user: User): string {
      return user.last_login_at ? this.formatDate(user.last_login_at) : 'Never';
    },

    getSelectedUserRolesCount(): number {
      return this.selectedUser?.roles?.length || 0;
    }
  };
}
